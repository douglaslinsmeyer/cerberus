# Artifacts Service Usage

## Overview

The artifacts service layer provides a complete solution for managing document artifacts in Cerberus. It handles file upload, storage, content extraction, chunking, and AI processing orchestration.

## Initialization

```go
import (
    "github.com/cerberus/backend/internal/modules/artifacts"
    "github.com/cerberus/backend/internal/platform/db"
    "github.com/cerberus/backend/internal/platform/storage"
)

// Initialize dependencies
database := db.New(dbConfig)
rustfsStorage := storage.NewRustFS(storageConfig)

// Create repository and service
repo := artifacts.NewRepository(database)
service := artifacts.NewService(repo, rustfsStorage)
```

## Upload an Artifact

```go
ctx := context.Background()

req := artifacts.UploadRequest{
    ProgramID:  programID,           // UUID of the program
    Filename:   "requirements.pdf",
    MimeType:   "application/pdf",
    Data:       fileBytes,           // Raw file data
    UploadedBy: userID,              // UUID of uploader
}

artifactID, err := service.UploadArtifact(ctx, req)
if err != nil {
    // Handle error (duplicate, unsupported type, storage failure, etc.)
    log.Printf("Upload failed: %v", err)
    return
}

log.Printf("Artifact uploaded successfully: %s", artifactID)
```

## Get Artifact Details

```go
// Get basic artifact info
artifact, err := service.GetArtifact(ctx, artifactID)
if err != nil {
    log.Printf("Failed to get artifact: %v", err)
    return
}

fmt.Printf("Filename: %s, Status: %s\n", artifact.Filename, artifact.ProcessingStatus)
```

## Get Artifact with Full Metadata

```go
// Get artifact with AI-extracted metadata
metadata, err := service.GetArtifactWithMetadata(ctx, artifactID)
if err != nil {
    log.Printf("Failed to get metadata: %v", err)
    return
}

// Access extracted information
if metadata.Summary != nil {
    fmt.Printf("Summary: %s\n", metadata.Summary.ExecutiveSummary)
}

for _, topic := range metadata.Topics {
    fmt.Printf("Topic: %s (confidence: %.2f)\n", topic.TopicName, topic.ConfidenceScore)
}

for _, person := range metadata.Persons {
    fmt.Printf("Person: %s, Role: %s\n", person.PersonName, person.PersonRole.String)
}
```

## List Artifacts

```go
// List all artifacts for a program
artifacts, err := service.ListArtifacts(ctx, programID, "", 50, 0)
if err != nil {
    log.Printf("Failed to list artifacts: %v", err)
    return
}

// List only pending artifacts
pendingArtifacts, err := service.ListArtifacts(ctx, programID, "pending", 50, 0)
if err != nil {
    log.Printf("Failed to list pending artifacts: %v", err)
    return
}

// Pagination
page2, err := service.ListArtifacts(ctx, programID, "", 50, 50)
```

## Delete an Artifact

```go
// Soft-delete artifact (also removes from storage)
err := service.DeleteArtifact(ctx, artifactID)
if err != nil {
    log.Printf("Failed to delete artifact: %v", err)
    return
}

log.Printf("Artifact deleted successfully")
```

## Queue for Reanalysis

```go
// Reset artifact for reprocessing
// This clears all AI-generated metadata and resets status to 'pending'
err := service.QueueForReanalysis(ctx, artifactID)
if err != nil {
    log.Printf("Failed to queue for reanalysis: %v", err)
    return
}

log.Printf("Artifact queued for reanalysis")
```

## Supported File Types

The service currently supports:
- **PDF**: `application/pdf`
- **Text**: `text/plain`, `text/markdown`
- **Office**: `application/vnd.openxmlformats-officedocument.wordprocessingml.document` (DOCX)
- **Spreadsheets**: `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` (XLSX)
- **CSV**: `text/csv`
- **JSON**: `application/json`
- **HTML/XML**: `text/html`, `application/xml`

## Error Handling

The service returns descriptive errors for common failure scenarios:

```go
artifactID, err := service.UploadArtifact(ctx, req)
if err != nil {
    switch {
    case strings.Contains(err.Error(), "duplicate artifact"):
        // Handle duplicate content hash
        log.Printf("This file has already been uploaded")
    case strings.Contains(err.Error(), "unsupported file type"):
        // Handle unsupported MIME type
        log.Printf("File type not supported")
    case strings.Contains(err.Error(), "required"):
        // Handle validation error
        log.Printf("Missing required field: %v", err)
    default:
        // Handle other errors
        log.Printf("Upload failed: %v", err)
    }
}
```

## Processing Status

Artifacts go through the following statuses:
- `pending`: Uploaded and awaiting AI processing
- `processing`: Currently being analyzed by AI workers
- `completed`: AI processing finished successfully
- `failed`: AI processing encountered an error

## Content Chunking

The service automatically chunks document content using the following strategy:
- **Chunk size**: 6,000 tokens (~4,500 words)
- **Overlap**: 200 tokens between chunks
- **Boundary preservation**: Chunks break at natural boundaries (paragraphs, sentences)

This chunking is optimized for:
1. LLM context window efficiency
2. Semantic coherence
3. Embedding generation
4. Search result retrieval

## Best Practices

1. **Duplicate Detection**: The service automatically detects duplicates based on content hash within the same program.

2. **Storage Cleanup**: When deletion or upload failures occur, the service automatically cleans up storage to prevent orphaned files.

3. **Status Monitoring**: Poll the artifact status to track processing progress:
   ```go
   for {
       artifact, _ := service.GetArtifact(ctx, artifactID)
       if artifact.ProcessingStatus == "completed" {
           break
       }
       time.Sleep(5 * time.Second)
   }
   ```

4. **Error Recovery**: Use `QueueForReanalysis` if AI processing fails or needs to be re-run with updated models.

5. **Pagination**: Always use pagination when listing artifacts to avoid memory issues with large datasets.
