# Qwen3-Coder

Qwen3-Coder is a proxy service that provides an OpenAI-compatible API interface for Qwen AI services. It handles OAuth2 device flow authentication and forwards requests to the Qwen API, making it easy to integrate Qwen AI capabilities into applications that support OpenAI APIs.

## Features

- OpenAI-compatible API interface
- OAuth2 device flow authentication with Qwen services
- Automatic token refresh
- Request forwarding with proper authorization headers
- Built-in model listing endpoint
- Cross-platform support (Linux, macOS, Windows)

## Installation

### Download Pre-built Binaries

Pre-built binaries are available for Linux, macOS, and Windows on the [releases page](https://github.com/damonto/qwen3-coder/releases).

### Building from Source

To build from source, you need Go 1.25 or later:

```bash
git clone https://github.com/damonto/qwen3-coder.git
cd qwen3-coder
go build -o qwen3-coder
```

## Usage

### Basic Usage

```bash
./qwen3-coder -token "your-api-key"
```

### Command Line Options

| Option | Description | Default Value |
|--------|-------------|---------------|
| `-listen` | Listen address | `:9527` |
| `-token` | API key for authentication | (required) |
| `-token-path` | Path to store token information | `./data/token.json` |

### Running as a Service

You can run qwen3-coder as a background service:

```bash
# Run in background
nohup ./qwen3-coder -token "your-api-key" > qwen3-coder.log 2>&1 &

# Or with a specific listen address
./qwen3-coder -listen ":8080" -token "your-api-key"
```

## Configuration

### Token Storage

The application stores authentication tokens in a JSON file (default: `./data/token.json`). This file contains sensitive information and should be protected with appropriate file permissions.

## API Endpoints

### Models

- `GET /v1/models` - List available models

### Chat Completions

- `POST /v1/chat/completions` - Chat completions endpoint

### Other Endpoints

All other `/v1/*` endpoints are forwarded to the Qwen API service.

## Authentication Flow

1. When starting the service, if no valid token is found, the application will initiate the OAuth2 device flow
2. You'll be prompted with a URL to visit and a code to enter
3. After successful authentication, the token is stored locally
4. The service automatically refreshes the token when it expires

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For support, please open an issue on the GitHub repository.
