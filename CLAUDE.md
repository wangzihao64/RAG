# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Go-based RAG (Retrieval-Augmented Generation) system that provides document upload, processing, and vector storage capabilities. The system uses an asynchronous pipeline to process documents: extract text → chunk → embed → store in Milvus vector database.

## Commands

### Running the Application

**Prerequisites — start these BEFORE the app, or it will exit:**

1. **PostgreSQL** must be reachable (see `[postgresql]` in `config/config.ini`).
2. **Milvus must be fully up.** `main()` builds the document-processing worker at startup, which connects to Milvus. If Milvus is unreachable, `worker.NewProcessorWorker` fails and `main()` calls `log.Fatalf` — the app runs DB migrations, then exits. A log that stops right after the migration queries with no "worker started" / Gin banner means Milvus wasn't ready.

```bash
# 1. Start Milvus (and its etcd + minio deps) and WAIT until it's healthy
docker-compose -f deployments/docker-compose.yml up -d
docker ps | grep milvus-standalone            # must show "(healthy)"
# milvus-etcd and milvus-minio alone are NOT enough — milvus-standalone must be running

# 2. Run the app
go run cmd/main.go

# Build and run
go build -o main cmd/main.go
./main
```

The server port is set by `HttpPort` in `config/config.ini` (currently `:8080`).

> Port note: `:3000` was previously the default but collides with a local Open WebUI Docker container (returns `405` with `server: uvicorn` — a sign the request never reached this Go app). Pick a free port if `:8080` is also taken.

### Testing

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./pkg/chunker

# Run tests with verbose output
go test -v ./...
```

### Dependencies

```bash
# Install/update dependencies
go mod tidy
```

Milvus is a hard startup dependency — see **Running the Application** above for how to start it and why the app exits without it.

## Architecture

### Layered Structure

The codebase follows a standard three-layer architecture:

1. **Handler** (`internal/handler/`) - HTTP request/response handling, parameter validation
2. **Service** (`internal/service/`) - Business logic, orchestration
3. **Repository** (`internal/repository/`) - Database access via GORM

### Document Processing Pipeline

The core of this system is an **asynchronous document processing pipeline**:

```
Upload → Pending → [Worker polls] → Processing → Extract → Chunk → Embed → Store → Ready
```

**Key Components:**

- **`internal/service/processor.go`**: Implements the `Pipeline` that orchestrates document processing
  - `ProcessDocument()`: Main entry point that coordinates the entire flow
  - `process()`: Executes Extract → Chunk → Embed → Store steps
  
- **`internal/worker/processor.go`**: Background worker that polls for pending documents
  - Runs on a configurable interval (default 10s)
  - Processes documents serially to avoid overwhelming external services
  - Updates document status atomically (pending → processing → ready/failed)

- **`pkg/extractor/`**: Extracts plain text from various file types (PDF, MD, TXT, DOCX)
- **`pkg/chunker/`**: Splits text into overlapping chunks (sliding window algorithm)
- **`pkg/embedding/`**: Calls DashScope API to generate embeddings
- **`pkg/vectordb/`**: Milvus client wrapper for vector operations

**Flow Details:**

1. User uploads document via `/api/v1/collections/:id/documents`
2. Document record created with `status = "pending"`, file saved to disk
3. Worker polls database every 10s for pending documents
4. For each pending document:
   - Status → "processing"
   - Extract text from file
   - Split into chunks (size/overlap from config)
   - Generate embeddings via DashScope
   - Delete old vectors for this document (if re-processing)
   - Insert new vectors into Milvus
   - Status → "ready" or "failed" (with error message)

### Configuration

`config/config.ini` contains all service configuration:

- **PostgreSQL**: Database connection
- **Milvus**: Vector database address and collection name
- **Embedding**: Provider (dashscope/mock), model, dimensions
- **JWT**: Secret and expiration
- **Storage**: Upload directory, file size limits, allowed types

**Environment Variables Required:**

- `DASHSCOPE_API_KEY`: API key for DashScope embedding service (not stored in config file)

### Authentication

JWT-based authentication with middleware in `internal/middleware/auth.go`:

- Login returns JWT token
- Protected routes use `middleware.JWTAuth()`
- Token includes user ID for permission checks
- Token stored in `Authorization: Bearer <token>` header

## Key Patterns

### Worker Pattern

The system uses a background worker instead of synchronous processing to:
- Avoid HTTP timeouts for large documents
- Rate-limit calls to external embedding API
- Provide retry capability for failed documents

The worker is started in `main()` and runs for the application's lifetime.

### Status State Machine

Documents follow a strict state machine:

```
pending → processing → ready
                    ↘ failed
```

Status transitions are atomic to prevent race conditions when multiple workers might exist in the future.

### Error Handling

- Document processing errors are **non-fatal** - they mark the document as `failed` and record `error_msg`
- The worker continues processing other documents
- Failed documents remain in the database for manual inspection/retry

## Development Notes

### Adding New File Types

To support new document formats:

1. Add file type to `AllowedTypes` in `config/config.ini`
2. Implement extraction logic in `pkg/extractor/extractor.go`
3. Test with various file samples to ensure text quality

### Modifying Chunking Strategy

Chunking parameters in `config/config.ini`:
- `ChunkSize`: Number of **runes** (not bytes) per chunk
- `ChunkOverlap`: Overlap between consecutive chunks
- Algorithm uses sliding window: `step = size - overlap`

The chunker in `pkg/chunker/` handles UTF-8 properly (won't split multi-byte characters).

### Testing Embedding Pipeline

For development without a DashScope API key:
- Set `Provider = mock` in `config/config.ini`
- Mock provider returns zero vectors (for pipeline testing only)
- Switch to `dashscope` for real embeddings

### Vector Database Schema

Milvus collection schema (defined in `pkg/vectordb/milvus.go`):
- `id`: Auto-generated unique ID
- `document_id`: Foreign key to documents table
- `collection_id`: Foreign key to collections table
- `chunk_index`: Position of chunk within document
- `content`: Original text chunk
- `vector`: Embedding vector (dimension from config)

## API Structure

All routes are under `/api/v1`:

- `POST /auth/register` - User registration
- `POST /auth/login` - User login (returns JWT)
- `GET /user/profile` - Get current user (authenticated)
- `POST /collections` - Create knowledge base (authenticated)
- `GET /collections` - List user's collections (authenticated)
- `POST /collections/:id/documents` - Upload document (authenticated)
- `GET /collections/:id/documents` - List documents in collection (authenticated)
- `GET /documents/:id` - Get document details (authenticated)
- `DELETE /documents/:id` - Delete document (authenticated)

Document status can be checked via `GET /documents/:id` - look for `status` field.
