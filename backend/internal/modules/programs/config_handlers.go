package programs

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// RegisterConfigRoutes registers program configuration endpoints
func RegisterConfigRoutes(r chi.Router, service *ConfigService) {
	r.Route("/programs/{programId}/config", func(r chi.Router) {
		r.Get("/", handleGetConfig(service))
		r.Put("/", handleUpdateConfig(service))
		r.Post("/vendors", handleAddVendor(service))
		r.Delete("/vendors/{vendorName}", handleRemoveVendor(service))
	})
}

// handleGetConfig returns the current program configuration
func handleGetConfig(service *ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		program, err := service.GetProgram(r.Context(), programID)
		if err != nil {
			respondError(w, http.StatusNotFound, "Program not found")
			return
		}

		respondSuccess(w, map[string]interface{}{
			"program_id":    program.ProgramID,
			"program_name":  program.ProgramName,
			"program_code":  program.ProgramCode,
			"configuration": program.Configuration,
		})
	}
}

// handleUpdateConfig updates the program configuration
func handleUpdateConfig(service *ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var req UpdateConfigRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate the configuration if provided
		if req.Company != nil || req.Taxonomy != nil || req.Vendors != nil {
			// Build a temporary config for validation
			currentConfig, err := service.GetProgramConfig(r.Context(), programID)
			if err != nil {
				respondError(w, http.StatusNotFound, "Program not found")
				return
			}

			testConfig := *currentConfig
			if req.Company != nil {
				testConfig.Company = *req.Company
			}
			if req.Taxonomy != nil {
				testConfig.Taxonomy = *req.Taxonomy
			}
			if req.Vendors != nil {
				testConfig.Vendors = *req.Vendors
			}

			if err := service.ValidateConfig(&testConfig); err != nil {
				respondError(w, http.StatusBadRequest, err.Error())
				return
			}
		}

		if err := service.UpdateProgramConfig(r.Context(), programID, req); err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Return updated configuration
		updatedConfig, err := service.GetProgramConfig(r.Context(), programID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "Failed to retrieve updated config")
			return
		}

		respondSuccess(w, map[string]interface{}{
			"message":       "Configuration updated successfully",
			"configuration": updatedConfig,
		})
	}
}

// handleAddVendor adds a new vendor to the configuration
func handleAddVendor(service *ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		var vendor VendorConfig
		if err := json.NewDecoder(r.Body).Decode(&vendor); err != nil {
			respondError(w, http.StatusBadRequest, "Invalid request body")
			return
		}

		// Validate vendor
		if vendor.Name == "" {
			respondError(w, http.StatusBadRequest, "Vendor name is required")
			return
		}

		if err := service.AddVendor(r.Context(), programID, vendor); err != nil {
			if err.Error() == "vendor already exists: "+vendor.Name {
				respondError(w, http.StatusConflict, err.Error())
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]string{
			"message": "Vendor added successfully",
			"vendor":  vendor.Name,
		})
	}
}

// handleRemoveVendor removes a vendor from the configuration
func handleRemoveVendor(service *ConfigService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		programIDStr := chi.URLParam(r, "programId")
		programID, err := uuid.Parse(programIDStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid program ID")
			return
		}

		vendorName := chi.URLParam(r, "vendorName")
		if vendorName == "" {
			respondError(w, http.StatusBadRequest, "Vendor name is required")
			return
		}

		if err := service.RemoveVendor(r.Context(), programID, vendorName); err != nil {
			if err.Error() == "vendor not found: "+vendorName {
				respondError(w, http.StatusNotFound, err.Error())
				return
			}
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		respondSuccess(w, map[string]string{
			"message": "Vendor removed successfully",
			"vendor":  vendorName,
		})
	}
}

// Response helpers
func respondSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
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
