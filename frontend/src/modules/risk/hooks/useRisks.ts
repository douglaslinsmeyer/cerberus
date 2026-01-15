import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { riskApi } from '@/services/api'

// Hook for listing risks
export function useRisks(programId: string, params?: {
  status?: string
  category?: string
  severity?: string
  owner_user_id?: string
  limit?: number
  offset?: number
}) {
  return useQuery({
    queryKey: ['risks', programId, params],
    queryFn: () => riskApi.listRisks(programId, params),
    enabled: !!programId,
  })
}

// Hook for listing risks with AI suggestions
export function useRisksWithSuggestions(programId: string, params?: {
  status?: string
  category?: string
  severity?: string
  owner_user_id?: string
  limit?: number
  offset?: number
}) {
  return useQuery({
    queryKey: ['risksWithSuggestions', programId, params],
    queryFn: () => riskApi.listRisksWithSuggestions(programId, params),
    enabled: !!programId,
  })
}

// Hook for getting a single risk with metadata
export function useRisk(programId: string, riskId: string) {
  return useQuery({
    queryKey: ['risk', programId, riskId],
    queryFn: () => riskApi.getRisk(programId, riskId),
    enabled: !!programId && !!riskId,
  })
}

// Hook for creating a risk
export function useCreateRisk(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: any) => riskApi.createRisk(programId, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['risks', programId] })
    },
  })
}

// Hook for updating a risk
export function useUpdateRisk(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ riskId, updates }: { riskId: string; updates: any }) =>
      riskApi.updateRisk(programId, riskId, updates),
    onSuccess: (_, { riskId }) => {
      queryClient.invalidateQueries({ queryKey: ['risk', programId, riskId] })
      queryClient.invalidateQueries({ queryKey: ['risks', programId] })
    },
  })
}

// Hook for deleting a risk
export function useDeleteRisk(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (riskId: string) => riskApi.deleteRisk(programId, riskId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['risks', programId] })
    },
  })
}

// Hook for listing risk suggestions
export function useRiskSuggestions(programId: string) {
  return useQuery({
    queryKey: ['riskSuggestions', programId],
    queryFn: () => riskApi.listSuggestions(programId),
    enabled: !!programId,
  })
}

// Hook for approving a risk suggestion
export function useApproveSuggestion(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ suggestionId, data }: { suggestionId: string; data?: any }) =>
      riskApi.approveSuggestion(programId, suggestionId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['riskSuggestions', programId] })
      queryClient.invalidateQueries({ queryKey: ['risks', programId] })
      queryClient.invalidateQueries({ queryKey: ['risksWithSuggestions', programId] })
    },
  })
}

// Hook for dismissing a risk suggestion
export function useDismissSuggestion(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ suggestionId, reason }: { suggestionId: string; reason: string }) =>
      riskApi.dismissSuggestion(programId, suggestionId, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['riskSuggestions', programId] })
      queryClient.invalidateQueries({ queryKey: ['risksWithSuggestions', programId] })
    },
  })
}

// Hook for listing mitigations for a risk
export function useMitigations(programId: string, riskId: string) {
  return useQuery({
    queryKey: ['mitigations', programId, riskId],
    queryFn: () => riskApi.listMitigations(programId, riskId),
    enabled: !!programId && !!riskId,
  })
}

// Hook for creating a mitigation
export function useCreateMitigation(programId: string, riskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: any) => riskApi.createMitigation(programId, riskId, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['mitigations', programId, riskId] })
      queryClient.invalidateQueries({ queryKey: ['risk', programId, riskId] })
    },
  })
}

// Hook for updating a mitigation
export function useUpdateMitigation(programId: string, riskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ mitigationId, updates }: { mitigationId: string; updates: any }) =>
      riskApi.updateMitigation(programId, riskId, mitigationId, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['mitigations', programId, riskId] })
    },
  })
}

// Hook for getting linked artifacts
export function useLinkedArtifacts(programId: string, riskId: string) {
  return useQuery({
    queryKey: ['linkedArtifacts', programId, riskId],
    queryFn: () => riskApi.getLinkedArtifacts(programId, riskId),
    enabled: !!programId && !!riskId,
  })
}

// Hook for linking an artifact
export function useLinkArtifact(programId: string, riskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ artifactId, linkType, description }: { artifactId: string; linkType: string; description?: string }) =>
      riskApi.linkArtifact(programId, riskId, artifactId, linkType, description),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['linkedArtifacts', programId, riskId] })
      queryClient.invalidateQueries({ queryKey: ['risk', programId, riskId] })
    },
  })
}

// Hook for unlinking an artifact
export function useUnlinkArtifact(programId: string, riskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (linkId: string) => riskApi.unlinkArtifact(programId, riskId, linkId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['linkedArtifacts', programId, riskId] })
    },
  })
}

// Hook for listing conversation threads
export function useConversations(programId: string, riskId: string) {
  return useQuery({
    queryKey: ['threads', programId, riskId],
    queryFn: () => riskApi.listThreads(programId, riskId),
    enabled: !!programId && !!riskId,
  })
}

// Hook for creating a conversation thread
export function useCreateThread(programId: string, riskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ title, threadType }: { title: string; threadType: string }) =>
      riskApi.createThread(programId, riskId, title, threadType),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['threads', programId, riskId] })
    },
  })
}

// Hook for getting thread messages
export function useThreadMessages(programId: string, riskId: string, threadId: string) {
  return useQuery({
    queryKey: ['threadMessages', programId, riskId, threadId],
    queryFn: () => riskApi.getThreadMessages(programId, riskId, threadId),
    enabled: !!programId && !!riskId && !!threadId,
    refetchInterval: 5000, // Poll every 5 seconds for new messages
  })
}

// Hook for adding a message to a thread
export function useAddMessage(programId: string, riskId: string, threadId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ messageText, messageFormat }: { messageText: string; messageFormat?: string }) =>
      riskApi.addMessage(programId, riskId, threadId, messageText, messageFormat),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['threadMessages', programId, riskId, threadId] })
      queryClient.invalidateQueries({ queryKey: ['threads', programId, riskId] })
    },
  })
}

// Hook for getting enrichments for a risk
export function useEnrichments(programId: string, riskId: string) {
  return useQuery({
    queryKey: ['enrichments', programId, riskId],
    queryFn: () => riskApi.getEnrichments(programId, riskId),
    enabled: !!programId && !!riskId,
  })
}

// Hook for accepting an enrichment
export function useAcceptEnrichment(programId: string, riskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ enrichmentId, reviewedBy }: { enrichmentId: string; reviewedBy: string }) =>
      riskApi.acceptEnrichment(programId, riskId, enrichmentId, reviewedBy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['enrichments', programId, riskId] })
      queryClient.invalidateQueries({ queryKey: ['risk', programId, riskId] })
    },
  })
}

// Hook for rejecting an enrichment
export function useRejectEnrichment(programId: string, riskId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ enrichmentId, reviewedBy }: { enrichmentId: string; reviewedBy: string }) =>
      riskApi.rejectEnrichment(programId, riskId, enrichmentId, reviewedBy),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['enrichments', programId, riskId] })
      queryClient.invalidateQueries({ queryKey: ['risk', programId, riskId] })
    },
  })
}
