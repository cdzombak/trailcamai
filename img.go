package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"

	"github.com/nfnt/resize"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func Downscale(imgData []byte, maxWidth uint) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	imgW := img.Bounds().Dx()
	imgH := img.Bounds().Dy()
	if imgH < 1 || imgW < 1 {
		return nil, fmt.Errorf("invalid image dimensions: %dx%d", imgW, imgH)
	}

	var resizedImg image.Image
	if uint(imgW) > maxWidth {
		resizedImg = resize.Resize(
			maxWidth,
			uint(float64(imgH)*(float64(maxWidth)/float64(imgW))),
			img,
			resize.Lanczos3,
		)
	}

	if resizedImg == nil {
		return imgData, nil
	}

	resizedData := bytes.NewBuffer(nil)
	err = jpeg.Encode(resizedData, resizedImg, nil)
	if err != nil {
		return nil, err
	}
	return resizedData.Bytes(), nil
}

// ExtractFramesFromVideo extracts 3 frames from a video at 30%, 50%, and 70% positions
func ExtractFramesFromVideo(videoPath string) ([][]byte, error) {
	// First, probe the video to get duration
	data, err := ffmpeg.Probe(videoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to probe video: %w", err)
	}

	// Parse duration from probe data - this is a simplified approach
	// In practice, you might want to use a proper JSON parser for the ffprobe output
	var duration float64
	if _, err := fmt.Sscanf(data, `"duration":"%f"`, &duration); err != nil {
		// If we can't parse duration, use fixed timestamps
		duration = 10.0 // assume 10 seconds if duration parsing fails
	}

	// Calculate frame extraction positions (30%, 50%, 70%)
	positions := []float64{
		duration * 0.3,
		duration * 0.5,
		duration * 0.7,
	}

	var frames [][]byte
	for _, pos := range positions {
		buf := bytes.NewBuffer(nil)
		err := ffmpeg.Input(videoPath, ffmpeg.KwArgs{"ss": pos}).
			Output("pipe:", ffmpeg.KwArgs{
				"vframes": 1,
				"format":  "image2",
				"vcodec":  "mjpeg",
			}).
			WithOutput(buf, os.Stderr).
			Silent(true).
			Run()
		if err != nil {
			return nil, fmt.Errorf("failed to extract frame at position %f: %w", pos, err)
		}
		frames = append(frames, buf.Bytes())
	}

	return frames, nil
}
