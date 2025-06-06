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

---
