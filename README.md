# GitHub API Fetch

**GitHub API Fetch** is a Go-based tool designed to interact with the GitHub API, providing functionalities such as fetching user data, repository information, and commit histories. It aims to simplify the process of integrating GitHub data into applications and services.

## 📘 Table of Contents
- [Project Overview](#project-overview)
- [Project Structure](#project-structure)
- [Getting Started](#getting-started)
- [Usage Example](#usage-example)
- [Design Decisions and Trade-offs](#design-decisions-and-trade-offs)
- [Future Improvements](#future-improvements)
- [Sample Output](#sample-output)

---

## 🛠️ Project Overview

This project provides functionality to:
- Fetch user details from the GitHub API
- Retrieve repository information and statistics
- Access commit histories and associated metadata
- Handle pagination and rate limiting gracefully
- Implement error handling and retries for API requests
- Provide a modular and extensible architecture for future integrations

---

## 🧪 Project Structure

- `github/`: Modules for interacting with the GitHub API
- `config/`: Configuration and environment variable management
- `logger/`: Application logging
- `main.go`: Application entry point
- `Makefile`: Automation tasks
- `Dockerfile`: Docker image build steps
- `docker-compose.yml`: Container orchestration

---

## 🚀 Getting Started

### Prerequisites

Install:
- [Go 1.18+](https://golang.org/dl/)
- [Docker](https://www.docker.com/get-started)
- [Docker Compose](https://docs.docker.com/compose/install/)
- [Make](https://www.gnu.org/software/make/)

### Build and Run

### Building and Running the Application

1. Clone the Repository:
   ```bash
   git clone <repository-url>
   cd Savannahtakehomeassi
   ```

2. Install Dependencies:
   ```bash
   go mod tidy
   ```

3. Build the Application:
   ```bash
   make build
   ```

4. Build Test Binaries:
   ```bash
   make build test-binaries
   ```

5. Start the Environment:
   ```bash
   docker-compose up --build -d
   ```

6. Run this to start the app :
   ```bash
   docker exec -it github_monitor_app ./github-fetch
   ```

7. Run Tests:
   ```bash
   docker exec -it github_monitor_app test-binaries/githubapifetch_<test-binary-name>
   ```

8. For  query to Run :
   ```bash
   psql -U "test-user" -h localhost -p 5434 -d github_monitor
   ```

9. For Query 1  :
    Get the top N commit authors by commit counts from the database
    ```bash 
     SELECT author_name, COUNT(*) AS commit_count
      FROM commits
      GROUP BY author_name
      ORDER BY commit_count DESC
      LIMIT N;  # Where N is number
     ```
10. For Query 2 : 
      Retrieve commits of a repository by repository name from the database
      ```bash 
     SELECT c.*
      FROM commits c
      JOIN repositories r ON c.repository_id = r.id
      WHERE r.name = <name_of_the_repo>;
     ```
---
### Design Decisions and Trade-offs

- Modular code layout for better scalability
- Uses environment variables to manage configuration securely
- Efficient logging system for observability
- Retry mechanisms for API reliability
- Structured separation between GitHub, DB, and logic layers

---

### Future Improvements

- Add web interface for query and visualization
- Enable multi-user OAuth-based GitHub access
- Paginated and filtered commit retrieval
- Background workers for scheduled GitHub syncs
- Support for GitHub webhooks


--- 

### Sample OutPut
```
{"level":"INFO","ts":"2025-06-06T02:16:10.448Z","caller":"logger/logger.go:75","msg":"Connecting to database","dsn":"user=test-user password=2222 dbname=github_monitor port=5432 host=db sslmode=disable"}
{"level":"INFO","ts":"2025-06-06T02:16:10.463Z","caller":"logger/logger.go:75","msg":"Database connection established","max_open_conns":25,"max_idle_conns":25,"conn_max_lifetime":300}
{"level":"INFO","ts":"2025-06-06T02:16:10.463Z","caller":"logger/logger.go:75","msg":"Initializing GitHub client","base_url":"https://api.github.com"}
{"level":"INFO","ts":"2025-06-06T02:16:10.464Z","caller":"logger/logger.go:75","msg":"Service initialized successfully","repo_owner":"barchart","repo_name":"marketdata-api-js","poll_interval":3600}
{"level":"INFO","ts":"2025-06-06T02:16:10.464Z","caller":"logger/logger.go:75","msg":"Processing initial repository","repo_owner":"barchart","repo_name":"marketdata-api-js"}
{"level":"INFO","ts":"2025-06-06T02:16:10.464Z","caller":"logger/logger.go:75","msg":"Fetching repository information","repo_owner":"barchart","repo_name":"marketdata-api-js"}
{"level":"INFO","ts":"2025-06-06T02:16:10.464Z","caller":"logger/logger.go:75","msg":"Fetching repository","owner":"barchart","name":"marketdata-api-js","url":"https://api.github.com/repos/barchart/marketdata-api-js"}
{"level":"INFO","ts":"2025-06-06T02:16:11.333Z","caller":"logger/logger.go:75","msg":"Successfully fetched repository","owner":"barchart","name":"marketdata-api-js","language":"JavaScript","stars":27}
{"level":"INFO","ts":"2025-06-06T02:16:11.333Z","caller":"logger/logger.go:75","msg":"Storing repository","owner":"barchart","name":"marketdata-api-js"}
{"level":"INFO","ts":"2025-06-06T02:16:11.344Z","caller":"logger/logger.go:75","msg":"Repository stored successfully","owner":"barchart","name":"marketdata-api-js","id":1}
{"level":"INFO","ts":"2025-06-06T02:16:11.344Z","caller":"logger/logger.go:75","msg":"Retrieving repository by name","name":"marketdata-api-js"}
{"level":"INFO","ts":"2025-06-06T02:16:11.346Z","caller":"logger/logger.go:75","msg":"Repository retrieved successfully","name":"marketdata-api-js"}
{"level":"INFO","ts":"2025-06-06T02:16:11.346Z","caller":"logger/logger.go:75","msg":"Fetching commits","repo_owner":"barchart","repo_name":"marketdata-api-js","since":"0001-01-01T00:00:00.000Z"}
{"level":"INFO","ts":"2025-06-06T02:16:11.346Z","caller":"logger/logger.go:75","msg":"Fetching commits","owner":"barchart","name":"marketdata-api-js","since":"0001-01-01T00:00:00.000Z","url":"https://api.github.com/repos/barchart/marketdata-api-js/commits"}
{"level":"INFO","ts":"2025-06-06T02:16:11.924Z","caller":"logger/logger.go:75","msg":"Successfully fetched commits","owner":"barchart","name":"marketdata-api-js","count":30}
{"level":"INFO","ts":"2025-06-06T02:16:11.924Z","caller":"logger/logger.go:75","msg":"Storing commits","repo_owner":"barchart","repo_name":"marketdata-api-js","commit_count":30}
{"level":"INFO","ts":"2025-06-06T02:16:11.924Z","caller":"logger/logger.go:75","msg":"Starting batch insertion of commits","count":30}
{"level":"INFO","ts":"2025-06-06T02:16:11.940Z","caller":"logger/logger.go:75","msg":"Successfully inserted commits","count":30}
{"level":"INFO","ts":"2025-06-06T02:16:11.940Z","caller":"logger/logger.go:75","msg":"Successfully processed repository","repo_owner":"barchart","repo_name":"marketdata-api-js","commit_count":30}
{"level":"INFO","ts":"2025-06-06T02:16:11.940Z","caller":"logger/logger.go:75","msg":"Starting repository monitoring","poll_interval":3600}
^Z^X^C{"level":"INFO","ts":"2025-06-06T02:16:14.929Z","caller":"logger/logger.go:75","msg":"Shutdown signal received, initiating graceful shutdown"}
{"level":"INFO","ts":"2025-06-06T02:16:14.930Z","caller":"logger/logger.go:75","msg":"Closing service"}
```
   

## Features

- Fetch repository information and commits from GitHub
- Store data in a SQLite database
- Monitor repositories for changes
- Reset sync points to fetch historical data
- Containerized deployment with Docker

## Installation

### Local Installation

```bash
go get github.com/yourusername/githubapifetch
```

### Docker Installation

1. Create a `.env` file with required environment variables:
```env
GITHUB_TOKEN=your_github_token
POSTGRES_USER=your_db_user
POSTGRES_PASSWORD=your_db_password
POSTGRES_DB=your_db_name
POSTGRES_PORT=5432
POLL_INTERVAL=300
```

2. Start the application:
```bash
docker-compose up -d
```

## Usage

### Starting the Service

The application will start automatically with Docker Compose:
```bash
docker-compose up -d
```

### Resetting Sync Points

You can reset the sync point for a repository using the existing container. Here's how:

1. Reset to default (30 days ago):
```bash
docker exec github_monitor_app ./github-fetch  reset-sync -repo your-repo-name
```

2. Reset to a specific number of days ago:
```bash
docker exec github_monitor_app ./github-fetch reset-sync -repo your-repo-name -days 60
```

Example for Chromium repository:
```bash
docker exec github_monitor_app ./github-fetch  reset-sync -repo chromium -days 60
```

Alternatively, you can run a one-off container for reset-sync:
```bash
docker-compose run --rm app ./github-fetch  reset-sync -repo your-repo-name -days 60
```

### What Happens When You Reset

When you reset a sync point:
1. The application connects to the database
2. Finds the repository by name
3. Resets the sync point to the specified date
4. Fetches all commits from that date forward
5. Stores the commits in the database
6. Continues monitoring from the new sync point

## Development

### Running Tests

```bash
go test ./...
```

### Project Structure

- `cmd/`: Command-line interface
- `config/`: Configuration management
- `db/`: Database operations
- `github/`: GitHub API client
- `models/`: Data models
- `service/`: Core service logic

### Docker Development

1. Build the development image:
```bash
docker-compose build
```

2. Run tests in container:
```bash
docker-compose run --rm app go test ./...
```

3. Run with hot reload:
```bash
docker-compose run --rm app go run main.go
```

## License

MIT License
