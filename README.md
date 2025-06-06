# GitHub API Fetch

A Go-based service that monitors and fetches data from GitHub repositories using the GitHub API.

## Features

- GitHub repository monitoring
- PostgreSQL database integration
- Configurable polling intervals
- Docker and Docker Compose support
- Comprehensive test coverage

## Prerequisites

- Go 1.24.2 or higher
- Docker and Docker Compose
- PostgreSQL (if running locally)
- GitHub Personal Access Token

## Configuration

The application uses environment variables for configuration. Create a `.env` file in the root directory with the following variables:

```env
GITHUB_TOKEN=your_github_token
POSTGRES_USER=your_db_user
POSTGRES_PASSWORD=your_db_password
POSTGRES_DB=your_db_name
POSTGRES_PORT=5432
POLL_INTERVAL=300  # in seconds
```

## Building and Running

### Using Make

```bash
# Build the application
make build

# Run tests
make test-binary

```

### Using Docker Compose

```bash
# Start the services
docker-compose up -d

# Stop the services
docker-compose down
```

## Project Structure

```
.
├── cmd/            # Application entry points
├── config/         # Configuration management
├── db/            # Database related code and migrations
├── fetcher/       # GitHub API fetching logic
├── github/        # GitHub API client
├── logger/        # Logging utilities
├── models/        # Data models
├── service/       # Business logic services
└── test-binaries/ # Test artifacts
```

## Development

1. Clone the repository
2. Install dependencies: `go mod download`
3. Set up your environment variables
4. Run the application locally or using Docker

## Testing

The project includes comprehensive test coverage. Run tests using:

```bash
make test-binary
```

## License

[Add your license information here]
