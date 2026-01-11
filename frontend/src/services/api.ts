import axios from 'axios'

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1'

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Types
export interface Artifact {
  artifact_id: string
  program_id: string
  filename: string
  file_type: string
  file_size_bytes: number
  mime_type: string
  processing_status: 'pending' | 'processing' | 'completed' | 'failed'
  uploaded_at: string
  processed_at?: string
}

export interface Topic {
  topic_id: string
  topic_name: string
  confidence_score: number
}

export interface Person {
  person_id: string
  person_name: string
  person_role?: string
  person_organization?: string
  confidence_score: number
}

export interface Fact {
  fact_id: string
  fact_type: string
  fact_key: string
  fact_value: string
  confidence_score: number
}

export interface Insight {
  insight_id: string
  insight_type: string
  title: string
  description: string
  severity?: string
  suggested_action?: string
  confidence_score: number
}

export interface ArtifactSummary {
  executive_summary: string
  sentiment?: string
  priority?: number
}

export interface ArtifactWithMetadata extends Artifact {
  summary?: ArtifactSummary
  topics?: Topic[]
  persons?: Person[]
  facts?: Fact[]
  insights?: Insight[]
}

export interface SearchResult {
  artifact: Artifact
  similarity: number
  snippet: string
}

// Artifacts API
export const artifactsApi = {
  // Upload artifact
  upload: async (programId: string, file: File) => {
    const formData = new FormData()
    formData.append('file', file)

    const response = await api.post(`/programs/${programId}/artifacts/upload`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    })

    return response.data.data as Artifact
  },

  // List artifacts
  list: async (programId: string, params?: { limit?: number; offset?: number; status?: string }) => {
    const response = await api.get(`/programs/${programId}/artifacts`, { params })
    return response.data.data.artifacts as Artifact[]
  },

  // Get artifact
  get: async (programId: string, artifactId: string) => {
    const response = await api.get(`/programs/${programId}/artifacts/${artifactId}`)
    return response.data.data as Artifact
  },

  // Get artifact with metadata
  getMetadata: async (programId: string, artifactId: string) => {
    const response = await api.get(`/programs/${programId}/artifacts/${artifactId}/metadata`)
    return response.data.data as ArtifactWithMetadata
  },

  // Download artifact
  download: async (programId: string, artifactId: string) => {
    const response = await api.get(`/programs/${programId}/artifacts/${artifactId}/download`, {
      responseType: 'blob',
    })
    return response.data
  },

  // Delete artifact
  delete: async (programId: string, artifactId: string) => {
    await api.delete(`/programs/${programId}/artifacts/${artifactId}`)
  },

  // Reanalyze artifact
  reanalyze: async (programId: string, artifactId: string) => {
    const response = await api.post(`/programs/${programId}/artifacts/${artifactId}/reanalyze`)
    return response.data.data
  },

  // Search artifacts
  search: async (programId: string, query: string, limit = 20) => {
    const response = await api.post(`/programs/${programId}/artifacts/search`, {
      query,
      limit,
    })
    return response.data.data.results as SearchResult[]
  },
}

export default api
