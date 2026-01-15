import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { stakeholdersApi, CreateStakeholderRequest, UpdateStakeholderRequest } from '@/services/api'

// Hook for listing stakeholders
export function useStakeholders(programId: string, params?: {
  type?: string
  is_internal?: boolean
  engagement_level?: string
  limit?: number
  offset?: number
}) {
  return useQuery({
    queryKey: ['stakeholders', programId, params],
    queryFn: () => stakeholdersApi.listStakeholders(programId, params),
    enabled: !!programId,
  })
}

// Hook for getting a single stakeholder
export function useStakeholder(programId: string, stakeholderId: string) {
  return useQuery({
    queryKey: ['stakeholder', programId, stakeholderId],
    queryFn: () => stakeholdersApi.getStakeholder(programId, stakeholderId),
    enabled: !!programId && !!stakeholderId,
  })
}

// Hook for creating a stakeholder
export function useCreateStakeholder(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: CreateStakeholderRequest) => stakeholdersApi.createStakeholder(programId, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stakeholders', programId] })
      queryClient.invalidateQueries({ queryKey: ['stakeholderSuggestions', programId] })
    },
  })
}

// Hook for updating a stakeholder
export function useUpdateStakeholder(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ stakeholderId, updates }: { stakeholderId: string; updates: UpdateStakeholderRequest }) =>
      stakeholdersApi.updateStakeholder(programId, stakeholderId, updates),
    onSuccess: (_, { stakeholderId }) => {
      queryClient.invalidateQueries({ queryKey: ['stakeholder', programId, stakeholderId] })
      queryClient.invalidateQueries({ queryKey: ['stakeholders', programId] })
    },
  })
}

// Hook for deleting a stakeholder
export function useDeleteStakeholder(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (stakeholderId: string) => stakeholdersApi.deleteStakeholder(programId, stakeholderId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stakeholders', programId] })
    },
  })
}

// Hook for getting AI suggestions
export function useStakeholderSuggestions(programId: string) {
  return useQuery({
    queryKey: ['stakeholderSuggestions', programId],
    queryFn: () => stakeholdersApi.getSuggestions(programId),
    enabled: !!programId,
  })
}

// Hook for linking a person to a stakeholder
export function useLinkPerson(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ personId, stakeholderId }: { personId: string; stakeholderId: string }) =>
      stakeholdersApi.linkPerson(programId, personId, stakeholderId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['stakeholderSuggestions', programId] })
      queryClient.invalidateQueries({ queryKey: ['stakeholders', programId] })
    },
  })
}

// Hook for getting grouped suggestions
export function useGroupedSuggestions(programId: string) {
  return useQuery({
    queryKey: ['groupedSuggestions', programId],
    queryFn: () => stakeholdersApi.getGroupedSuggestions(programId),
    enabled: !!programId,
  })
}

// Hook for confirming a merge group
export function useConfirmMergeGroup(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ groupId, request }: {
      groupId: string
      request: {
        selected_name: string
        selected_role?: string
        selected_organization?: string
        create_stakeholder: boolean
      }
    }) => stakeholdersApi.confirmMergeGroup(programId, groupId, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groupedSuggestions', programId] })
      queryClient.invalidateQueries({ queryKey: ['stakeholderSuggestions', programId] })
      queryClient.invalidateQueries({ queryKey: ['stakeholders', programId] })
    },
  })
}

// Hook for rejecting a merge group
export function useRejectMergeGroup(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (groupId: string) =>
      stakeholdersApi.rejectMergeGroup(programId, groupId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groupedSuggestions', programId] })
      queryClient.invalidateQueries({ queryKey: ['stakeholderSuggestions', programId] })
    },
  })
}

// Hook for modifying group members
export function useModifyGroupMembers(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ groupId, request }: {
      groupId: string
      request: {
        add_person_ids?: string[]
        remove_person_ids?: string[]
      }
    }) => stakeholdersApi.modifyGroupMembers(programId, groupId, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groupedSuggestions', programId] })
    },
  })
}

// Hook for refreshing person grouping
export function useRefreshGrouping(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => stakeholdersApi.refreshGrouping(programId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['groupedSuggestions', programId] })
    },
  })
}

// Hook for getting linked artifacts for a stakeholder
export function useLinkedArtifacts(programId: string, stakeholderId: string) {
  return useQuery({
    queryKey: ['linkedArtifacts', programId, stakeholderId],
    queryFn: () => stakeholdersApi.getLinkedArtifacts(programId, stakeholderId),
    enabled: !!programId && !!stakeholderId,
  })
}
