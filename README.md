# trailcamai

AI-powered trail camera image and video organizer. Automatically sorts photos and videos by detected animals using vision models.

## Usage

```bash
./trailcamai -dir /path/to/trail/camera/files
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-dir` | *required* | Directory containing images/videos to sort |
| `-model` | `llava:latest` | Vision model to use |
| `-min-quality` | `3` | Quality threshold (1-5), below which files go to `_lowq/` |
| `-region` | `Michigan` | Geographic region for classification context |
| `-maxW` | `1200` | Maximum image width for processing |

### AI Backend Configuration

**Ollama (default):**
```bash
# Environment variable
export OLLAMA_HOST=http://localhost:11434

# Or command line
./trailcamai -dir images -ollama-endpoint http://remote:11434
```

**OpenAI-compatible:**
```bash
# Environment variables
export OPENAI_BASE_URL=https://api.openai.com/v1
export OPENAI_API_KEY=sk-your-key

# Or command line
./trailcamai -dir images -openai-endpoint https://api.openai.com/v1 -openai-key sk-key
```

## Output Structure

Files are organized into subdirectories by detected content:

```
input-directory/
├── deer/           # Videos/images containing deer
├── crane/          # Videos/images containing cranes
├── none/           # No animals detected
├── _lowq/          # Low quality images (below threshold)
└── undo.sh         # Script to restore original organization
```

**Videos with multiple animals** are hardlinked into each relevant directory.

## Requirements

- **Ollama**: Install locally or specify remote endpoint
- **OpenAI API**: For OpenAI-compatible endpoints
- **FFmpeg**: Required for video frame extraction

## Supported Formats

- **Images**: `.jpg`, `.jpeg`
- **Videos**: `.mp4`, `.avi`, `.mov`, `.mkv`, `.webm`