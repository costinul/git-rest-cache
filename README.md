# Git REST Cache

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Git REST Cache is an open-source Go project designed to act as a caching proxy for online Git repositories. It provides fast, read-only access to repository file content without relying on remote API rate limits. The project supports on‐demand shallow cloning, periodic background updates, token validation, and TTL-based pruning of stale caches, ensuring repositories are always up-to-date while minimizing API calls and network overhead.

This makes it an ideal solution for developer tools, AI-powered code analysis, CI/CD systems, and automation scripts that require efficient, low-latency access to source code files. By caching repositories locally, Git REST Cache avoids provider rate limits and accelerates file retrieval, making it particularly useful for large-scale projects and high-frequency operations.

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Authentication & Security](#authentication-security)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [API Endpoints](#api-endpoints)
- [Testing](#testing)
- [License](#license)

## Features

- **On-Demand Cloning:** Clones a repository using `--depth=1` when a file is first requested.
- **Branch-Level Caching:** Each repository is cached by a unique hash (computed from provider, owner, repo, and token) with each branch stored in its own subfolder.
- **Token-Based Access:** Supports PAT/OAuth token validation to access private repositories.
- **Background Updates:** Periodically fetches updates for cached repositories.
- **TTL & Pruning:** Automatically removes caches that have not been accessed for a configurable time.
- **Extensible Provider Support:** Easily add support for GitHub, GitLab, Bitbucket, Azure DevOps, etc.
- **Pluggable Architecture:** Uses a `GitCacheManager` interface to abstract Git operations (cloning, fetching, reading files, deletion) for easier testing and extension.
- **Concurrency:** Employs per-repo locking to ensure thread-safe operations without blocking unrelated repositories.

## Architecture

The core idea is to maintain a local cache of Git repositories. Each repository is identified by a unique hash (derived from its provider, owner, repository name, and optionally the token used for cloning). Within each repository cache folder, branches are stored as separate subfolders. For example, the folder structure might look like:

```
/cached-repos/
  └── <repo-hash>/  
       ├── main/         # Cache for branch "main"
       ├── dev/          # Cache for branch "dev"
       └── feature-x/    # Cache for branch "feature-x"
```

A central `GitCacheManager` interface handles Git operations such as:
- **CloneRepo:** Clones the repository for a specific branch.
- **FetchUpdates:** Fetches new commits from the remote for a branch.
- **ReadFile:** Returns file content from a cached branch.
- **DeleteRepo:** Removes a stale repository cache.

This separation makes it easy to test the folder-scanning logic and Git operations without directly coupling to internal cache structures.

## Authentication & Security

For private repositories, authentication tokens are **validated against the provider’s API** before access is granted. If valid, the token is **cached** to minimize repeated API calls and improve response time. 

### Token Caching & Expiry
- Tokens are **temporarily stored** to avoid excessive API requests.
- **TTL (Time-to-Live) is configurable**, ensuring tokens are refreshed at regular intervals.
- When a token expires, it is **revalidated automatically** upon the next request.

This system ensures **secure and efficient authentication**, reducing latency while maintaining repository access control.


## Installation

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/costinul/git-rest-cache.git
   cd git-rest-cache
   ```

2. **Build the Project:**

   ```bash
   go build -o git-rest-cache ./cmd
   ```

3. **Run the Binary:**

   ```bash
   ./git-rest-cache
   ```

## Configuration

The project uses [Viper](https://github.com/spf13/viper) and [Cobra](https://github.com/spf13/cobra) to support configuration via a YAML file, environment variables, and CLI flags.

**Example `config/config.yaml`:**

```yaml
port: 8080
log-level: "info"
storage-folder: "./cached-repos"
repo-ttl: "24h"
token-ttl: "24h"
repo-check-interval: "5m"
```

Environment variables are prefixed with `GIT_REST_CACHE_` (e.g., `GIT_REST_CACHE_PORT=9090`).

## Usage

Git REST Cache exposes a REST API to retrieve file content from cached Git repositories. When a request is made, the service:

1. Validates the token (if provided) via the appropriate Git provider.
2. Clones the repository for the specified branch into a cache folder (if not already cached).
3. Returns the requested file content from the cached branch.

### Example Request

To fetch the README.md file from the GitHub repository `https://github.com/costinul/git-rest-cache` on the `main` branch, call:

```
http://localhost:8080/github/costinul/git-rest-cache/main/README.md
```

- If the repository is public, the request can be made without an `X-Token` header.
- For private repositories, include the token in the `X-Token` header.

## API Endpoints

Each Git provider has its own specific URL pattern for accessing repositories. Below is the current and planned support for various providers.

### Implemented

#### **GitHub**
- **URL Pattern:**  
  `/github/:owner/:repo/:branch/*filepath`
- **Example Request:**  
http://localhost:8080/github/costinul/git-rest-cache/main/README.md

- **Notes:**  
- The `:owner` represents either an organization or a personal user account.
- The repository is cloned if not already cached.
- A valid `X-Token` is required for private repositories.

---

### Planned Support

#### **GitLab** (Planned)
- **Expected URL Pattern:**  
`/gitlab/:namespace/:repo/:branch/*filepath`
- **Notes:**  
- GitLab supports **nested groups**, so `:namespace` may contain multiple levels (e.g., `gitlab.com/acme/widgets/frontend-repo`).
- Private repositories will require authentication.

#### **Bitbucket** (Planned)
- **Expected URL Pattern:**  
`/bitbucket/:workspace/:repo/:branch/*filepath`
- **Notes:**  
- Older Bitbucket URLs used `/:owner/:repo`, but newer ones prefer **workspaces**.
- The `:workspace` ID can be a team, a user, or an organization.

#### **Azure DevOps** (Planned)
- **Expected URL Pattern:**  
`/devops/:organization/:project/:repo/:branch/*filepath`
- **Notes:**  
- Azure DevOps URLs include both an `:organization` and a `:project`, before the `_git/:repo` path.
- The repository is uniquely referenced within an organization/project context.

---

### Request Headers
- **`X-Token` (optional):** A valid authentication token required for accessing private repositories.



### Response
- Returns the file content with `Content-Type: application/octet-stream`.
- If the repository or branch is not cached, it will be **cloned on demand**.
- Subsequent requests will fetch the file **from the cache** unless an update occurs.



## Testing

Run the following commands to execute tests:

- **Git Cache Module Tests (with race detection):**

  ```bash
  go test -race -timeout 10m -count=5 ./gitcache -v
  ```

- **API Module Tests:**

  ```bash
  go test -timeout 10s ./api -v
  ```

These tests cover unit tests and integration tests for caching logic, folder scanning, Git operations, and API endpoints.

## License

This project is licensed under the [MIT License](LICENSE).
