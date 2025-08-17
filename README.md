# Qwen3-Coder

Qwen3-Coder is a proxy service that provides an OpenAI-compatible API interface for Qwen AI services. This project extracts APIs from the [qwen-code](https://github.com/QwenLM/qwen-code) project and implements a forwarding service, allowing you to provide the API to other AI tools such as Cline, RooCode, Kilo Code, and more.

The project uses Qwen OAuth, which enables you to utilize the free quota of 2,000 requests per day with no token limits and a rate limit of 60 requests per minute.

## Features

- OpenAI-compatible API interface
- OAuth2 device flow authentication with Qwen services
- Automatic token refresh
- Request forwarding with proper authorization headers
- Built-in model listing endpoint
- Cross-platform support (Linux, macOS, Windows)
- Compatible with popular AI tools like Cline, RooCode, Kilo Code
- Utilizes Qwen OAuth for free quota access (2,000 requests/day)

## Qwen API Usage
This project uses Qwen OAuth to provide access to the Qwen API with the following free quotas:
- 2,000 requests per day with no token limits
- 60 requests per minute rate limit

This makes it possible to use Qwen's powerful coding models in your favorite AI tools without needing to purchase additional API credits.

## Compatible AI Tools
This proxy service allows you to use Qwen's API with various AI tools that support OpenAI-compatible APIs, including:
- Cline
- RooCode
- Kilo Code
- Cursor
- Continue.dev
- And many other tools that support OpenAI API format

Simply configure your AI tool to use this service as an OpenAI-compatible endpoint, and you'll be able to leverage Qwen's powerful coding models.

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

### Docker Usage

You can also run qwen3-coder using Docker:

#### Using Docker Compose

1. Create a `.env` file in the same directory as your `compose.yaml`:

```bash
echo "APP_TOKEN=your-api-key" > .env
```

2. Start the service:

```bash
docker-compose up -d
```

3. View the logs to get the authorization URL for authentication:

```bash
docker-compose logs -f qwen3-coder
```

After starting the container, check the logs to find the `Authorization URL` and complete the OAuth2 authentication process.

#### Using Docker Directly

```bash
docker run -d \
  --name qwen3-coder \
  -p 9527:9527 \
  -e APP_TOKEN="your-api-key" \
  -v qwen3-coder-data:/data \
  --restart unless-stopped \
  damonto/qwen3-coder
```

Then view the logs to get the authorization URL:

```bash
docker logs -f qwen3-coder
```

#### Docker Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `APP_TOKEN` | API key for authentication | Yes |

#### Important Note for Docker Users

After starting the Docker container, you **must** view the container logs to get the `Authorization URL` and complete the OAuth2 authentication process. The service will not be fully functional until this authentication step is completed.

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
