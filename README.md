# trailcamai

AI-powered trail camera image and video organizer. `trailcamai` automatically sorts photos and videos by the animals present in them using vision LLMs.

## Features

- **AI-Powered Classification**: Uses vision models to identify animals in trail camera footage
- **Supports Images and Videos**: Processes both images and videos from trail cameras
- **Quality Assessment**: Filters out low-quality images based on motion blur and clarity
- **Flexible AI Backends**: Supports both Ollama and OpenAI-compatible endpoints

### Output Structure

Generates organized directory structure:
```
input-directory/
├── deer/           # Videos/images containing deer
├── crane/          # Videos/images containing cranes
├── none/           # No animals detected
├── _lowq/          # Low quality images (below threshold)
```

## Requirements

- **FFmpeg**: Required for video frame extraction
- **Ollama** or **OpenAI-compatible** API endpoint and key

## Usage

```bash
trailcamai [options] -dir /path/to/trail/camera/files
```

### Options

- `-dir`: Directory containing images/videos to sort (required)
- `-model`: Vision model to use (default: `llava:latest`)
- `-min-quality`: Quality threshold 1-5; files under this threshold go to `_lowq/` (default: `3`)
- `-region`: Geographic region for classification context (default: `Michigan`)
- `-maxW`: Maximum image width sent to the LLM (default: `1200`)
- `-ollama-endpoint`: Ollama endpoint URL
- `-openai-endpoint`: OpenAI-compatible endpoint URL
- `-openai-key`: API key for OpenAI-compatible endpoint

### AI Backend Configuration

#### Ollama

```bash
# Environment variable
export OLLAMA_HOST=http://localhost:11434

# Or command line
trailcamai -dir images -ollama-endpoint http://remote:11434
```

#### OpenAI-compatible

```bash
# Environment variables
export OPENAI_BASE_URL=https://api.openai.com/v1
export OPENAI_API_KEY=sk-your-key

# Or command line
trailcamai -dir images -openai-endpoint https://api.openai.com/v1 -openai-key sk-key
```

## Installation

### Debian via apt repository

Set up my `oss` apt repository:

```shell
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://dist.cdzombak.net/keys/dist-cdzombak-net.gpg -o /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo chmod 644 /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo mkdir -p /etc/apt/sources.list.d
sudo curl -fsSL https://dist.cdzombak.net/cdzombak-oss.sources -o /etc/apt/sources.list.d/cdzombak-oss.sources
sudo chmod 644 /etc/apt/sources.list.d/cdzombak-oss.sources
sudo apt update
```

Then install `trailcamai` via `apt-get`:

```shell
sudo apt-get install trailcamai
```

### Homebrew

```shell
brew install cdzombak/oss/trailcamai
```

### Manual from build artifacts

Pre-built binaries for Linux and macOS on various architectures are downloadable from each [GitHub Release](https://github.com/cdzombak/trailcamai/releases). Debian packages for each release are available as well.

## License

GNU GPL v3; see [LICENSE](LICENSE) in this repo for details.

## Author

Chris Dzombak
- [dzombak.com](https://www.dzombak.com)
- [GitHub @cdzombak](https://github.com/cdzombak)