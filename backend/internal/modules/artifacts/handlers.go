package artifacts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RegisterRoutes registers all artifact endpoints
func RegisterRoutes(r chi.Router, service *Service) {
	r.Route("/programs/{programId}/artifacts", func(r chi.Router) {
		r.Post("/upload", handleUpload(service))
		r.Get("/", handleList(service))
		r.Post("/search", handleSearch(service))

		r.Route("/{artifactId}", func(r chi.Router) {
			r.Get("/", handleGet(service))
			r.Get("/metadata", handleGetMetadata(service))
			r.Get("/download", handleDownload(service))
			r.Post("/reanalyze", handleReanalyze(service))
			r.Delete("/", handleDelete(service))
		})
	})
}

// handleUpload handles artifact upload
func handleUpload(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse program ID from URL
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		// Parse multipart form (50MB max)
		if err := r.ParseMultipartForm(50 << 20); err != nil {
			respondError(w, http.StatusBadRequest, "Failed to parse upload")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			respondError(w, http.StatusBadRequest, "No file provided")
			return
		}
		defer file.Close()

		// Read file data
		data, err := io.ReadAll(file)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to read file")
			return
		}

		// TODO: Get uploadedBy from JWT claims (for now use hardcoded value)
		uploadedBy := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		// Upload artifact
		artifactID, err := service.UploadArtifact(r.Context(), UploadRequest{
			ProgramID:  programID,
			Filename:   header.Filename,
			MimeType:   header.Header.Get("Content-Type"),
			Data:       data,
			UploadedBy: uploadedBy,
		})

		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondCreated(w, map[string]string{
			"artifact_id": artifactID.String(),
			"message":     "Artifact uploaded successfully. AI analysis queued.",
		})
	}
}

// handleList lists all artifacts for a program
func handleList(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		// Parse query parameters
		limit := 50
		offset := 0
		status := r.URL.Query().Get("status")

		// Parse limit
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			var l int
			if _, err := fmt.Sscanf(limitStr, "%d", &l); err == nil && l > 0 {
				limit = l
			}
		}

		// Parse offset
		if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
			var o int
			if _, err := fmt.Sscanf(offsetStr, "%d", &o); err == nil && o >= 0 {
				offset = o
			}
		}

		artifacts, err := service.ListArtifacts(r.Context(), programID, status, limit, offset)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]interface{}{
			"artifacts": artifacts,
			"limit":     limit,
			"offset":    offset,
		})
	}
}

// handleGet retrieves a single artifact
func handleGet(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifactIDStr := chi.URLParam(r, "artifactId")
		artifactID, err := uuid.Parse(artifactIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid artifact ID")
			return
		}

		artifact, err := service.GetArtifact(r.Context(), artifactID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Artifact not found")
			return
		}

		respondSuccess(w, artifact)
	}
}

// handleGetMetadata retrieves artifact with all AI-extracted metadata
func handleGetMetadata(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifactIDStr := chi.URLParam(r, "artifactId")
		artifactID, err := uuid.Parse(artifactIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid artifact ID")
			return
		}

		artifact, err := service.GetArtifactWithMetadata(r.Context(), artifactID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Artifact not found")
			return
		}

		respondSuccess(w, artifact)
	}
}

// handleDownload downloads the original artifact file
func handleDownload(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifactIDStr := chi.URLParam(r, "artifactId")
		artifactID, err := uuid.Parse(artifactIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid artifact ID")
			return
		}

		// Get artifact
		artifact, err := service.GetArtifact(r.Context(), artifactID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Artifact not found")
			return
		}

		// Download from storage
		data, err := service.storage.Download(r.Context(), extractFileID(artifact.StoragePath))
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to download file")
			return
		}

		// Set headers
		w.Header().Set("Content-Type", artifact.MimeType)
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", artifact.Filename))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", artifact.FileSizeBytes))

		// Write file data
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}
}

// handleDelete deletes an artifact
func handleDelete(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifactIDStr := chi.URLParam(r, "artifactId")
		artifactID, err := uuid.Parse(artifactIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid artifact ID")
			return
		}

		if err := service.DeleteArtifact(r.Context(), artifactID); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondNoContent(w)
	}
}

// handleReanalyze queues an artifact for reanalysis
func handleReanalyze(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		artifactIDStr := chi.URLParam(r, "artifactId")
		artifactID, err := uuid.Parse(artifactIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid artifact ID")
			return
		}

		if err := service.QueueForReanalysis(r.Context(), artifactID); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]string{
			"message": "Artifact queued for reanalysis",
		})
	}
}

// handleSearch performs semantic search across artifacts
func handleSearch(service *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		_, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var req struct {
			Query string `json:"query"`
			Limit int    `json:"limit"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		if req.Limit == 0 {
			req.Limit = 20
		}

		// TODO: Implement semantic search when embeddings are ready
		// For now, return empty results
		respondSuccess(w, map[string]interface{}{
			"results": []interface{}{},
			"query":   req.Query,
			"message": "Semantic search will be available after AI analysis implementation",
		})
	}
}

// Helper function to extract file ID from storage path
func extractFileID(storagePath string) string {
	// Storage path format: "artifacts/{fileID}"
	parts := strings.Split(storagePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return storagePath
}

// Response helpers (using shared API response functions)
func respondSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func respondCreated(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func respondNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
