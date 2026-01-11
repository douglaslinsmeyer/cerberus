import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { artifactsApi } from '@/services/api'

// Hook for listing artifacts
export function useArtifacts(programId: string, params?: { status?: string }) {
  return useQuery({
    queryKey: ['artifacts', programId, params],
    queryFn: () => artifactsApi.list(programId, params),
    enabled: !!programId,
  })
}

// Hook for getting a single artifact
export function useArtifact(programId: string, artifactId: string) {
  return useQuery({
    queryKey: ['artifact', programId, artifactId],
    queryFn: () => artifactsApi.get(programId, artifactId),
    enabled: !!programId && !!artifactId,
  })
}

// Hook for getting artifact with metadata
export function useArtifactMetadata(programId: string, artifactId: string) {
  return useQuery({
    queryKey: ['artifact', programId, artifactId, 'metadata'],
    queryFn: () => artifactsApi.getMetadata(programId, artifactId),
    enabled: !!programId && !!artifactId,
    refetchInterval: (data) => {
      // Auto-refresh if processing
      if (data?.processing_status === 'pending' || data?.processing_status === 'processing') {
        return 3000 // Poll every 3 seconds
      }
      return false // Don't poll if completed or failed
    },
  })
}

// Hook for uploading artifacts
export function useUploadArtifact(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => artifactsApi.upload(programId, file),
    onSuccess: () => {
      // Invalidate artifacts list to refresh
      queryClient.invalidateQueries({ queryKey: ['artifacts', programId] })
    },
  })
}

// Hook for deleting artifacts
export function useDeleteArtifact(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (artifactId: string) => artifactsApi.delete(programId, artifactId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['artifacts', programId] })
    },
  })
}

// Hook for reanalyzing artifacts
export function useReanalyzeArtifact(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (artifactId: string) => artifactsApi.reanalyze(programId, artifactId),
    onSuccess: (_, artifactId) => {
      queryClient.invalidateQueries({ queryKey: ['artifact', programId, artifactId] })
      queryClient.invalidateQueries({ queryKey: ['artifacts', programId] })
    },
  })
}

// Hook for searching artifacts
export function useSearchArtifacts(programId: string, query: string) {
  return useQuery({
    queryKey: ['artifacts', 'search', programId, query],
    queryFn: () => artifactsApi.search(programId, query),
    enabled: !!programId && !!query && query.length > 2,
  })
}
