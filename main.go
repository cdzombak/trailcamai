package main

import (
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go"
	"github.com/hashicorp/errwrap"
	ollapi "github.com/ollama/ollama/api"
	"github.com/samber/lo"
	"github.com/sashabaranov/go-openai"
)

const qualityPrompt = "This is a still image from an outdoor trail camera. Rate its image quality on a scale of 1-5 (5 is the best), especially considering motion blur and clarity of the subject. Your response MUST be a single number."

func classPrompt(region string) string {
	return fmt.Sprintf("This is an image frame from an outdoor trail camera in %s. If the image shows an animal, identify what kind of animal it is. Your response MUST be a single word. If there is no animal, reply with \"none\". If there is an animal but you can't guess what it is, reply \"unknown\".", region)
}

// VisionClient interface for both Ollama and OpenAI clients
type VisionClient interface {
	Qualify(frame []byte) (int, error)
	Classify(frame []byte, region string) (string, error)
}

func main() {
	dir := flag.String("dir", "", "directory of images/videos to sort")
	model := flag.String("model", "llava:latest", "multimodal model to use")
	targetW := flag.Uint("maxW", 1200, "max width for images")

	// AI endpoint configuration
	ollamaEndpoint := flag.String("ollama-endpoint", os.Getenv("OLLAMA_HOST"), "Ollama endpoint URL (default from OLLAMA_HOST env var or http://localhost:11434)")
	openaiEndpoint := flag.String("openai-endpoint", os.Getenv("OPENAI_BASE_URL"), "OpenAI-compatible endpoint URL (default from OPENAI_BASE_URL env var)")
	openaiKey := flag.String("openai-key", os.Getenv("OPENAI_API_KEY"), "API key for OpenAI-compatible endpoint (default from OPENAI_API_KEY env var)")
	region := flag.String("region", "Michigan", "region to mention in classification prompt")

	flag.Parse()

	if *dir == "" {
		flag.PrintDefaults()
		return
	}

	var client VisionClient
	var err error

	if *openaiEndpoint != "" {
		// Use OpenAI-compatible endpoint
		client, err = newOpenAIClient(*openaiEndpoint, *openaiKey, *model)
		if err != nil {
			log.Fatalf("Failed to create OpenAI client: %v", err)
		}
	} else {
		// Use Ollama
		var oll *ollapi.Client
		if *ollamaEndpoint != "" {
			// Use specified or environment-provided Ollama endpoint
			parsedURL, err := url.Parse(*ollamaEndpoint)
			if err != nil {
				log.Fatalf("Failed to parse Ollama endpoint URL: %v", err)
			}
			oll = ollapi.NewClient(parsedURL, http.DefaultClient)
		} else {
			// Use default Ollama client (will use OLLAMA_HOST if set, or localhost:11434)
			oll, err = ollapi.ClientFromEnvironment()
			if err != nil {
				log.Fatalf("Failed to create Ollama client: %v", err)
			}
		}
		client = &ollamaClient{client: oll, model: *model}
	}

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatalf("Failed to read directory '%s': %v", *dir, err)
	}

	for _, e := range entries {
		processEntry(*dir, e, client, *targetW, *region)
	}
}

// ollamaClient implements VisionClient for Ollama
type ollamaClient struct {
	client *ollapi.Client
	model  string
}

// openaiClient implements VisionClient for OpenAI-compatible endpoints
type openaiClient struct {
	client *openai.Client
	model  string
}

// newOpenAIClient creates a new OpenAI-compatible client
func newOpenAIClient(endpoint, apiKey, model string) (*openaiClient, error) {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = endpoint
	client := openai.NewClientWithConfig(config)
	return &openaiClient{client: client, model: model}, nil
}

// Qualify implements VisionClient for ollamaClient
func (o *ollamaClient) Qualify(frame []byte) (int, error) {
	var retv int
	err := retry.Do(func() error {
		err := o.client.Generate(
			context.Background(),
			&ollapi.GenerateRequest{
				Model:     o.model,
				Prompt:    qualityPrompt,
				Stream:    lo.ToPtr(false),
				KeepAlive: lo.ToPtr(ollapi.Duration{Duration: 5 * time.Minute}),
				Images:    []ollapi.ImageData{frame},
			},
			func(resp ollapi.GenerateResponse) (err error) {
				retv, err = strconv.Atoi(strings.TrimSpace(resp.Response))
				return
			},
		)
		if err != nil {
			return err
		}
		if retv < 4 {
			return errTryQualityAgain
		}
		return nil
	}, retry.Attempts(3), retry.Delay(50*time.Millisecond))
	if errwrap.ContainsType(err, errTryQualityAgain) {
		err = nil
	}
	return retv, err
}

// Classify implements VisionClient for ollamaClient
func (o *ollamaClient) Classify(frame []byte, region string) (string, error) {
	var retv string
	err := retry.Do(func() error {
		err := o.client.Generate(
			context.Background(),
			&ollapi.GenerateRequest{
				Model:     o.model,
				Prompt:    classPrompt(region),
				Stream:    lo.ToPtr(false),
				KeepAlive: lo.ToPtr(ollapi.Duration{Duration: 5 * time.Minute}),
				Images:    []ollapi.ImageData{frame},
			},
			func(resp ollapi.GenerateResponse) error {
				retv = strings.ToLower(strings.TrimSpace(resp.Response))
				return nil
			},
		)
		if err != nil {
			return err
		}
		if len(strings.Split(retv, " ")) > 1 {
			return errTryClassifyingAgain
		}
		if retv == "unknown" || retv == "none" {
			return errTryClassifyingAgain
		}
		return nil
	}, retry.Attempts(5), retry.Delay(50*time.Millisecond))
	if errwrap.ContainsType(err, errTryClassifyingAgain) {
		err = nil
	}
	return retv, err
}

// Qualify implements VisionClient for openaiClient
func (o *openaiClient) Qualify(frame []byte) (int, error) {
	var retv int
	err := retry.Do(func() error {
		base64Image := base64.StdEncoding.EncodeToString(frame)
		resp, err := o.client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: o.model,
				Messages: []openai.ChatCompletionMessage{
					{
						Role: openai.ChatMessageRoleUser,
						MultiContent: []openai.ChatMessagePart{
							{
								Type: openai.ChatMessagePartTypeText,
								Text: qualityPrompt,
							},
							{
								Type: openai.ChatMessagePartTypeImageURL,
								ImageURL: &openai.ChatMessageImageURL{
									URL: "data:image/jpeg;base64," + base64Image,
								},
							},
						},
					},
				},
			},
		)
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			return errors.New("no response from OpenAI")
		}
		retv, err = strconv.Atoi(strings.TrimSpace(resp.Choices[0].Message.Content))
		if err != nil {
			return err
		}
		if retv < 4 {
			return errTryQualityAgain
		}
		return nil
	}, retry.Attempts(3), retry.Delay(50*time.Millisecond))
	if errwrap.ContainsType(err, errTryQualityAgain) {
		err = nil
	}
	return retv, err
}

// Classify implements VisionClient for openaiClient
func (o *openaiClient) Classify(frame []byte, region string) (string, error) {
	var retv string
	err := retry.Do(func() error {
		base64Image := base64.StdEncoding.EncodeToString(frame)
		resp, err := o.client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: o.model,
				Messages: []openai.ChatCompletionMessage{
					{
						Role: openai.ChatMessageRoleUser,
						MultiContent: []openai.ChatMessagePart{
							{
								Type: openai.ChatMessagePartTypeText,
								Text: classPrompt(region),
							},
							{
								Type: openai.ChatMessagePartTypeImageURL,
								ImageURL: &openai.ChatMessageImageURL{
									URL: "data:image/jpeg;base64," + base64Image,
								},
							},
						},
					},
				},
			},
		)
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			return errors.New("no response from OpenAI")
		}
		retv = strings.ToLower(strings.TrimSpace(resp.Choices[0].Message.Content))
		if len(strings.Split(retv, " ")) > 1 {
			return errTryClassifyingAgain
		}
		if retv == "unknown" || retv == "none" {
			return errTryClassifyingAgain
		}
		return nil
	}, retry.Attempts(5), retry.Delay(50*time.Millisecond))
	if errwrap.ContainsType(err, errTryClassifyingAgain) {
		err = nil
	}
	return retv, err
}

func processEntry(dir string, e os.DirEntry, client VisionClient, targetW uint, region string) {
	log.Printf("Processing '%s' ...", e.Name())
	stat, err := e.Info()
	if err != nil {
		log.Fatalf("\tFailed to stat '%s': %v", e.Name(), err)
	}

	if stat.IsDir() {
		log.Println("\tIs a directory; skipping")
		return
	}

	fName := filepath.Join(dir, e.Name())
	fExt := strings.ToLower(filepath.Ext(e.Name()))
	var frames [][]byte
	var isVideo bool

	// Handle different file types
	if fExt == ".jpg" || fExt == ".jpeg" {
		content, err := os.ReadFile(fName)
		if err != nil {
			log.Fatalf("\tFailed to read file '%s': %v", e.Name(), err)
		}
		frames = append(frames, content)
		isVideo = false
	} else if fExt == ".mp4" || fExt == ".avi" || fExt == ".mov" || fExt == ".mkv" || fExt == ".webm" {
		// Extract frames from video
		videoFrames, err := ExtractFramesFromVideo(fName)
		if err != nil {
			log.Printf("\tFailed to extract frames from video '%s': %v", e.Name(), err)
			return
		}
		frames = videoFrames
		isVideo = true
	} else {
		log.Printf("\tFile type '%s'; skipping", fExt)
		return
	}

	qualityResult := 0
	detectedAnimals := make(map[string]bool) // Track all detected animals

	for _, frame := range frames {
		frame, err := Downscale(frame, targetW)
		if err != nil {
			log.Fatalf("\tFailed to downscale image: %v", err)
		}

		frameQualityResult, err := client.Qualify(frame)
		if err != nil {
			log.Printf("\tVision quality query failed for '%s': %v", e.Name(), err)
			return
		}
		qualityResult = max(qualityResult, frameQualityResult)
		if frameQualityResult < 3 {
			continue
		}

		frameClassResult, err := client.Classify(frame, region)
		if err != nil {
			log.Printf("\tVision classification query failed for '%s': %v", e.Name(), err)
			return
		}

		// Add detected animal to our set (excluding none/unknown)
		if frameClassResult != "" && frameClassResult != "none" && frameClassResult != "unknown" {
			detectedAnimals[frameClassResult] = true
			log.Printf("\tDetected in frame: %s", frameClassResult)
		}
	}

	if qualityResult < 3 {
		log.Println("\tLow quality; moving to '_lowq'...")
		if err := move(dir, e, "_lowq"); err != nil {
			log.Fatalf("\t%v", err)
		}
		return
	}

	// Handle classification results
	if len(detectedAnimals) == 0 {
		log.Println("\tNo animals detected")
		if err := move(dir, e, "none"); err != nil {
			log.Fatalf("\t%v", err)
		}
	} else if len(detectedAnimals) == 1 {
		// Single animal detected
		for animal := range detectedAnimals {
			log.Printf("\tDetected: %s", animal)
			if err := move(dir, e, animal); err != nil {
				log.Fatalf("\t%v", err)
			}
		}
	} else {
		// Multiple animals detected - hardlink to multiple directories
		log.Printf("\tMultiple animals detected: %v", getKeys(detectedAnimals))
		if isVideo {
			if err := hardlinkToMultipleDirectories(dir, e, getKeys(detectedAnimals)); err != nil {
				log.Fatalf("\t%v", err)
			}
		} else {
			// For images, just pick the first one (existing behavior)
			for animal := range detectedAnimals {
				log.Printf("\tUsing first detection: %s", animal)
				if err := move(dir, e, animal); err != nil {
					log.Fatalf("\t%v", err)
				}
				break
			}
		}
	}
}

// getKeys returns the keys of a map as a slice
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// hardlinkToMultipleDirectories creates hardlinks of the file in multiple directories
func hardlinkToMultipleDirectories(fromDir string, e os.DirEntry, animals []string) error {
	sourcePath := filepath.Join(fromDir, e.Name())

	for _, animal := range animals {
		destDir := filepath.Join(fromDir, animal)
		destPath := filepath.Join(destDir, e.Name())

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", destDir, err)
		}

		// Create hardlink
		if err := os.Link(sourcePath, destPath); err != nil {
			return fmt.Errorf("failed to create hardlink from '%s' to '%s': %w", sourcePath, destPath, err)
		}
		log.Printf("\tHardlinked to '%s'", animal)
	}

	// Remove the original file after creating all hardlinks
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("failed to remove original file '%s': %w", sourcePath, err)
	}

	return nil
}

var errTryQualityAgain = errors.New("got low quality; try again")
var errTryClassifyingAgain = errors.New("got unknown or none; try classifying again")

func move(fromDir string, e os.DirEntry, toDir string) error {
	destDir := filepath.Join(fromDir, toDir)
	destFName := filepath.Join(destDir, e.Name())
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory '%s': %w", destDir, err)
	}
	if err := os.Rename(filepath.Join(fromDir, e.Name()), destFName); err != nil {
		return fmt.Errorf("failed to move '%s' to '%s': %w", e.Name(), destDir, err)
	}
	return nil
}
