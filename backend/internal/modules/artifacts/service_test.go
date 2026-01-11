package artifacts

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/cerberus/backend/internal/platform/storage"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Mock storage implementation
type mockStorage struct {
	uploadFunc   func(ctx context.Context, filename string, data []byte) (*storage.FileInfo, error)
	deleteFunc   func(ctx context.Context, fileID string) error
	downloadFunc func(ctx context.Context, fileID string) ([]byte, error)
	getInfoFunc  func(ctx context.Context, fileID string) (*storage.FileInfo, error)
}

func (m *mockStorage) Upload(ctx context.Context, filename string, data []byte) (*storage.FileInfo, error) {
	if m.uploadFunc != nil {
		return m.uploadFunc(ctx, filename, data)
	}
	return &storage.FileInfo{
		ID:          uuid.New().String(),
		Filename:    filename,
		Size:        int64(len(data)),
		ContentHash: "mock-hash",
		Path:        "artifacts/" + uuid.New().String(),
		UploadedAt:  time.Now(),
	}, nil
}

func (m *mockStorage) Delete(ctx context.Context, fileID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, fileID)
	}
	return nil
}

func (m *mockStorage) Download(ctx context.Context, fileID string) ([]byte, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, fileID)
	}
	return []byte("mock data"), nil
}

func (m *mockStorage) GetInfo(ctx context.Context, fileID string) (*storage.FileInfo, error) {
	if m.getInfoFunc != nil {
		return m.getInfoFunc(ctx, fileID)
	}
	return &storage.FileInfo{
		ID:          fileID,
		Filename:    "test.txt",
		Size:        100,
		ContentHash: "mock-hash",
		Path:        "artifacts/" + fileID,
	}, nil
}

// Mock repository implementation
type mockRepository struct {
	createFunc        func(ctx context.Context, artifact *Artifact) error
	getByIDFunc       func(ctx context.Context, artifactID uuid.UUID) (*Artifact, error)
	listByProgramFunc func(ctx context.Context, programID uuid.UUID, limit, offset int) ([]Artifact, error)
	deleteFunc        func(ctx context.Context, artifactID uuid.UUID) error
	updateStatusFunc  func(ctx context.Context, artifactID uuid.UUID, status string) error
	saveChunksFunc    func(ctx context.Context, artifactID uuid.UUID, chunks []Chunk) error
	getMetadataFunc   func(ctx context.Context, artifactID uuid.UUID) (*ArtifactWithMetadata, error)
	db                interface{ ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) }
}

func (m *mockRepository) Create(ctx context.Context, artifact *Artifact) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, artifact)
	}
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, artifactID uuid.UUID) (*Artifact, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, artifactID)
	}
	return &Artifact{
		ArtifactID:       artifactID,
		ProgramID:        uuid.New(),
		Filename:         "test.txt",
		ProcessingStatus: "pending",
		UploadedAt:       time.Now(),
	}, nil
}

func (m *mockRepository) ListByProgram(ctx context.Context, programID uuid.UUID, limit, offset int) ([]Artifact, error) {
	if m.listByProgramFunc != nil {
		return m.listByProgramFunc(ctx, programID, limit, offset)
	}
	return []Artifact{}, nil
}

func (m *mockRepository) Delete(ctx context.Context, artifactID uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, artifactID)
	}
	return nil
}

func (m *mockRepository) UpdateStatus(ctx context.Context, artifactID uuid.UUID, status string) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, artifactID, status)
	}
	return nil
}

func (m *mockRepository) SaveChunks(ctx context.Context, artifactID uuid.UUID, chunks []Chunk) error {
	if m.saveChunksFunc != nil {
		return m.saveChunksFunc(ctx, artifactID, chunks)
	}
	return nil
}

func (m *mockRepository) GetMetadata(ctx context.Context, artifactID uuid.UUID) (*ArtifactWithMetadata, error) {
	if m.getMetadataFunc != nil {
		return m.getMetadataFunc(ctx, artifactID)
	}
	artifact, _ := m.GetByID(ctx, artifactID)
	return &ArtifactWithMetadata{
		Artifact: *artifact,
	}, nil
}

// Mock DB executor for clearMetadata
type mockDBExecutor struct {
	execFunc  func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	queryFunc func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func (m *mockDBExecutor) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, query, args...)
	}
	return &mockResult{rowsAffected: 1}, nil
}

func (m *mockDBExecutor) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	// Return nil rows (tests that use this should provide their own implementation)
	return nil, sql.ErrNoRows
}

type mockResult struct {
	rowsAffected int64
	err          error
}

func (m *mockResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (m *mockResult) RowsAffected() (int64, error) {
	return m.rowsAffected, m.err
}

// Test UploadArtifact - Happy Path
func TestUploadArtifact_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	req := UploadRequest{
		ProgramID:  uuid.New(),
		Filename:   "test.txt",
		MimeType:   "text/plain",
		Data:       []byte("This is test content for the artifact upload."),
		UploadedBy: uuid.New(),
	}

	artifactID, err := service.UploadArtifact(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if artifactID == uuid.Nil {
		t.Fatal("expected valid artifact ID, got nil UUID")
	}
}

// Test UploadArtifact - Missing ProgramID
func TestUploadArtifact_MissingProgramID(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	req := UploadRequest{
		ProgramID:  uuid.Nil,
		Filename:   "test.txt",
		MimeType:   "text/plain",
		Data:       []byte("content"),
		UploadedBy: uuid.New(),
	}

	_, err := service.UploadArtifact(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing program_id, got nil")
	}
	if err.Error() != "program_id is required" {
		t.Fatalf("expected 'program_id is required', got: %v", err)
	}
}

// Test UploadArtifact - Missing UploadedBy
func TestUploadArtifact_MissingUploadedBy(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	req := UploadRequest{
		ProgramID:  uuid.New(),
		Filename:   "test.txt",
		MimeType:   "text/plain",
		Data:       []byte("content"),
		UploadedBy: uuid.Nil,
	}

	_, err := service.UploadArtifact(ctx, req)
	if err == nil {
		t.Fatal("expected error for missing uploaded_by, got nil")
	}
	if err.Error() != "uploaded_by is required" {
		t.Fatalf("expected 'uploaded_by is required', got: %v", err)
	}
}

// Test UploadArtifact - Empty Data
func TestUploadArtifact_EmptyData(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	req := UploadRequest{
		ProgramID:  uuid.New(),
		Filename:   "test.txt",
		MimeType:   "text/plain",
		Data:       []byte{},
		UploadedBy: uuid.New(),
	}

	_, err := service.UploadArtifact(ctx, req)
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
}

// Test UploadArtifact - Unsupported File Type
func TestUploadArtifact_UnsupportedFileType(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	req := UploadRequest{
		ProgramID:  uuid.New(),
		Filename:   "test.xyz",
		MimeType:   "application/x-unknown",
		Data:       []byte("content"),
		UploadedBy: uuid.New(),
	}

	_, err := service.UploadArtifact(ctx, req)
	if err == nil {
		t.Fatal("expected error for unsupported file type, got nil")
	}
}

// Test UploadArtifact - Duplicate Content Hash
func TestUploadArtifact_DuplicateHash(t *testing.T) {
	ctx := context.Background()

	pqErr := &pq.Error{
		Code:       "23505",
		Constraint: "artifacts_content_hash_program_unique",
	}

	mockRepo := &mockRepository{
		createFunc: func(ctx context.Context, artifact *Artifact) error {
			return pqErr
		},
	}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	req := UploadRequest{
		ProgramID:  uuid.New(),
		Filename:   "duplicate.txt",
		MimeType:   "text/plain",
		Data:       []byte("This is duplicate content."),
		UploadedBy: uuid.New(),
	}

	_, err := service.UploadArtifact(ctx, req)
	if err == nil {
		t.Fatal("expected error for duplicate content, got nil")
	}
	if err.Error() != "duplicate artifact: file with same content already exists in this program" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// Test UploadArtifact - Storage Upload Failure
func TestUploadArtifact_StorageUploadFailure(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{
		uploadFunc: func(ctx context.Context, filename string, data []byte) (*storage.FileInfo, error) {
			return nil, errors.New("storage service unavailable")
		},
	}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	req := UploadRequest{
		ProgramID:  uuid.New(),
		Filename:   "test.txt",
		MimeType:   "text/plain",
		Data:       []byte("content"),
		UploadedBy: uuid.New(),
	}

	_, err := service.UploadArtifact(ctx, req)
	if err == nil {
		t.Fatal("expected error for storage upload failure, got nil")
	}
}

// Test GetArtifact - Success
func TestGetArtifact_Success(t *testing.T) {
	ctx := context.Background()
	artifactID := uuid.New()

	mockRepo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*Artifact, error) {
			return &Artifact{
				ArtifactID: id,
				Filename:   "test.txt",
			}, nil
		},
	}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	artifact, err := service.GetArtifact(ctx, artifactID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if artifact.ArtifactID != artifactID {
		t.Fatalf("expected artifact ID %v, got %v", artifactID, artifact.ArtifactID)
	}
}

// Test GetArtifact - Invalid ID
func TestGetArtifact_InvalidID(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	_, err := service.GetArtifact(ctx, uuid.Nil)
	if err == nil {
		t.Fatal("expected error for nil UUID, got nil")
	}
}

// Test GetArtifactWithMetadata - Success
func TestGetArtifactWithMetadata_Success(t *testing.T) {
	ctx := context.Background()
	artifactID := uuid.New()

	mockRepo := &mockRepository{
		getMetadataFunc: func(ctx context.Context, id uuid.UUID) (*ArtifactWithMetadata, error) {
			return &ArtifactWithMetadata{
				Artifact: Artifact{
					ArtifactID: id,
					Filename:   "test.txt",
				},
				Topics: []Topic{
					{TopicID: uuid.New(), TopicName: "Test Topic"},
				},
			}, nil
		},
	}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	metadata, err := service.GetArtifactWithMetadata(ctx, artifactID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(metadata.Topics) != 1 {
		t.Fatalf("expected 1 topic, got %d", len(metadata.Topics))
	}
}

// Test ListArtifacts - Success
func TestListArtifacts_Success(t *testing.T) {
	ctx := context.Background()
	programID := uuid.New()

	mockRepo := &mockRepository{
		listByProgramFunc: func(ctx context.Context, pid uuid.UUID, limit, offset int) ([]Artifact, error) {
			return []Artifact{
				{ArtifactID: uuid.New(), Filename: "test1.txt"},
				{ArtifactID: uuid.New(), Filename: "test2.txt"},
			}, nil
		},
	}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	artifacts, err := service.ListArtifacts(ctx, programID, "", 50, 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}
}

// Test ListArtifacts - Invalid ProgramID
func TestListArtifacts_InvalidProgramID(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	_, err := service.ListArtifacts(ctx, uuid.Nil, "", 50, 0)
	if err == nil {
		t.Fatal("expected error for nil program ID, got nil")
	}
}

// Test ListArtifacts - Limit Validation
func TestListArtifacts_LimitValidation(t *testing.T) {
	ctx := context.Background()
	programID := uuid.New()

	mockRepo := &mockRepository{
		listByProgramFunc: func(ctx context.Context, pid uuid.UUID, limit, offset int) ([]Artifact, error) {
			// Verify limit is capped at 1000
			if limit > 1000 {
				t.Errorf("limit should be capped at 1000, got %d", limit)
			}
			return []Artifact{}, nil
		},
	}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	// Test with limit > 1000
	_, err := service.ListArtifacts(ctx, programID, "", 5000, 0)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// Test DeleteArtifact - Success
func TestDeleteArtifact_Success(t *testing.T) {
	ctx := context.Background()
	artifactID := uuid.New()

	mockRepo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*Artifact, error) {
			return &Artifact{
				ArtifactID:  id,
				StoragePath: "artifacts/test-file-id",
			}, nil
		},
		deleteFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	err := service.DeleteArtifact(ctx, artifactID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// Test DeleteArtifact - Invalid ID
func TestDeleteArtifact_InvalidID(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	err := service.DeleteArtifact(ctx, uuid.Nil)
	if err == nil {
		t.Fatal("expected error for nil UUID, got nil")
	}
}

// Test QueueForReanalysis - Success
func TestQueueForReanalysis_Success(t *testing.T) {
	ctx := context.Background()
	artifactID := uuid.New()

	mockDB := &mockDBExecutor{}
	mockRepo := &mockRepository{
		getByIDFunc: func(ctx context.Context, id uuid.UUID) (*Artifact, error) {
			return &Artifact{ArtifactID: id}, nil
		},
		updateStatusFunc: func(ctx context.Context, id uuid.UUID, status string) error {
			if status != "pending" {
				t.Errorf("expected status 'pending', got '%s'", status)
			}
			return nil
		},
		db: mockDB,
	}
	mockStore := &mockStorage{}

	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	err := service.QueueForReanalysis(ctx, artifactID)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// Test QueueForReanalysis - Invalid ID
func TestQueueForReanalysis_InvalidID(t *testing.T) {
	ctx := context.Background()
	mockRepo := &mockRepository{}
	mockStore := &mockStorage{}

	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(mockRepo, mockDB, mockStore)

	err := service.QueueForReanalysis(ctx, uuid.Nil)
	if err == nil {
		t.Fatal("expected error for nil UUID, got nil")
	}
}

// Test inferFileType
func TestInferFileType(t *testing.T) {
	mockDB := &mockDBExecutor{}
	service := NewServiceWithMocks(&mockRepository{}, mockDB, &mockStorage{})

	tests := []struct {
		mimeType string
		filename string
		expected string
	}{
		{"application/pdf", "test.pdf", "pdf"},
		{"text/plain", "test.txt", "text"},
		{"application/vnd.openxmlformats-officedocument.wordprocessingml.document", "test.docx", "docx"},
		{"text/csv", "data.csv", "csv"},
		{"application/json", "config.json", "json"},
		{"application/octet-stream", "data.xlsx", "xlsx"},
		{"", "unknown.xyz", "xyz"},
		{"", "noextension", "unknown"},
	}

	for _, tt := range tests {
		result := service.inferFileType(tt.mimeType, tt.filename)
		if result != tt.expected {
			t.Errorf("inferFileType(%q, %q) = %q, expected %q", tt.mimeType, tt.filename, result, tt.expected)
		}
	}
}
