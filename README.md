# RAG Service

A Go-based retrieval-augmented generation (RAG) API for building private
knowledge bases. Users can create collections, upload documents, retrieve
relevant chunks, and ask questions grounded in the uploaded content.

The service uses PostgreSQL for application data and Milvus for vector search.
It supports DashScope-compatible embedding and chat models, with a `mock`
provider available for local pipeline testing without an API key.

## Features

- JWT-based registration, login, and protected API endpoints
- Per-user knowledge base collections
- Upload and asynchronous processing of PDF, DOCX, Markdown, and text files
- Configurable document chunking, vector embeddings, and Milvus retrieval
- Optional DashScope reranking for two-stage retrieval
- Retrieval-only, non-streaming RAG, and SSE streaming RAG endpoints

## Prerequisites

- Go 1.25 or later
- Docker and Docker Compose
- A DashScope API key for real embeddings, reranking, and LLM responses
  (optional when using the mock providers)

## Quick Start

### 1. Start PostgreSQL and Milvus

From the repository root, start the required infrastructure:

```bash
docker compose -f deployments/docker-compose.yml up -d
```

Wait until the containers are healthy, especially `milvus-standalone`:

```bash
docker compose -f deployments/docker-compose.yml ps
```

The default local services are:

| Service | Address |
| --- | --- |
| PostgreSQL | `127.0.0.1:5432` |
| Milvus | `127.0.0.1:19530` |
| MinIO API | `http://127.0.0.1:9000` |
| MinIO Console | `http://127.0.0.1:9001` |

### 2. Configure the application

The default configuration is in `config/config.ini`. Its PostgreSQL and
Milvus values match the Docker Compose setup.

For real RAG requests, export a DashScope API key before starting the service:

```bash
export DASHSCOPE_API_KEY="your_dashscope_api_key"
```

Do not place API keys in `config/config.ini` or commit them to source control.

For a local test run without an API key, change the following providers in
`config/config.ini` to `mock`:

```ini
[embedding]
Provider = mock

[llm]
Provider = mock

[reranker]
Enabled = false
```

### 3. Run the API

```bash
go run cmd/main.go
```

The server listens on `http://localhost:8080` by default. Confirm it is
running with:

```bash
curl http://localhost:8080/api/v1/ping
```

Expected response:

```json
{"message":"pong"}
```

To build a binary instead:

```bash
go build -o main cmd/main.go
./main
```

## Basic Workflow

All endpoints except health checks and authentication require the header
`Authorization: Bearer <token>`.

1. Register an account with `POST /api/v1/auth/register`.
2. Log in with `POST /api/v1/auth/login` and save the returned token.
3. Create a collection with `POST /api/v1/collections`.
4. Upload a supported document to `POST /api/v1/collections/:id/documents`.
5. Wait until the document status indicates it has been processed.
6. Query the collection or start a RAG chat using the endpoints below.

Example registration request:

```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com","password":"change-me"}'
```

Example collection creation request:

```bash
curl -X POST http://localhost:8080/api/v1/collections \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Product Docs","description":"Internal product documentation","is_public":false}'
```

Example document upload:

```bash
curl -X POST http://localhost:8080/api/v1/collections/1/documents \
  -H "Authorization: Bearer $TOKEN" \
  -F 'file=@./example.pdf'
```

Example retrieval request:

```bash
curl -X POST http://localhost:8080/api/v1/collections/1/query \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"query":"What does this document describe?","top_k":5}'
```

## API Overview

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/v1/ping` | Health check |
| `POST` | `/api/v1/auth/register` | Register a user |
| `POST` | `/api/v1/auth/login` | Log in and obtain a JWT |
| `GET` | `/api/v1/user/profile` | Get the authenticated user profile |
| `POST` | `/api/v1/collections` | Create a knowledge base collection |
| `GET` | `/api/v1/collections` | List the current user's collections |
| `GET`, `PUT`, `DELETE` | `/api/v1/collections/:id` | Read, update, or delete a collection |
| `POST` | `/api/v1/collections/:id/documents` | Upload a document (`file` multipart field) |
| `GET` | `/api/v1/collections/:id/documents` | List collection documents |
| `POST` | `/api/v1/collections/:id/query` | Retrieve relevant chunks as JSON |
| `POST` | `/api/v1/collections/:id/chat` | Stream a RAG answer over SSE |
| `POST` | `/api/v1/collections/:id/chat_eval` | Return a non-streaming RAG answer |
| `GET`, `DELETE` | `/api/v1/documents/:id` | Read document metadata or delete a document |
| `GET` | `/api/v1/documents/:id/content` | Preview/download document content |

The query and chat endpoints accept JSON in this form:

```json
{
  "query": "Your question",
  "top_k": 5
}
```

`top_k` is optional and must be between 1 and 50. The streaming chat endpoint
emits SSE events named `message`, `sources`, `done`, and, on failure, `error`.

## Configuration

`config/config.ini` controls the default service behavior. The following
settings can be overridden with environment variables:

| Environment variable | Purpose |
| --- | --- |
| `DASHSCOPE_API_KEY` | DashScope API key for embeddings, LLM, and reranking |
| `APP_MODEL` | Gin mode (`debug` or release mode) |
| `HTTP_PORT` | HTTP listen address, for example `:8080` |
| `DATABASE_URL` | PostgreSQL connection URL, when supported by deployment configuration |
| `DB_DRIVER`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection settings |
| `JWT_SECRET` | JWT signing secret; change this for every non-local deployment |
| `UPLOAD_DIR` | Directory used for uploaded files |
| `MILVUS_ADDRESS`, `MILVUS_COLLECTION` | Milvus connection and collection settings |
| `RERANK_BASE_URL`, `RERANK_MODEL` | Reranking service settings |

The default upload limit is 50 MB. Supported file types are `pdf`, `md`,
`txt`, and `docx`; change `AllowedTypes` in `config/config.ini` to adjust the
list.

## Development

Run all tests:

```bash
go test ./...
```

Run the chunker tests only:

```bash
go test ./pkg/chunker
```

Stop the local infrastructure when finished:

```bash
docker compose -f deployments/docker-compose.yml down
```

The Compose volumes preserve database and vector data between restarts.
