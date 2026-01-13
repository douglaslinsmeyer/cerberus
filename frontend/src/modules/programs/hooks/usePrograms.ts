import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { programsApi, CreateProgramRequest, Program } from '../../../services/api'

export function usePrograms(params?: { status?: string; search?: string }) {
  return useQuery({
    queryKey: ['programs', params],
    queryFn: () => programsApi.listPrograms(params),
  })
}

export function useProgram(programId: string) {
  return useQuery({
    queryKey: ['program', programId],
    queryFn: () => programsApi.getProgram(programId),
    enabled: !!programId,
  })
}

export function useCreateProgram() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: CreateProgramRequest) => programsApi.createProgram(request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['programs'] })
    },
  })
}

export function useUpdateProgram() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ programId, updates }: { programId: string; updates: Partial<Program> }) =>
      programsApi.updateProgram(programId, updates),
    onSuccess: (_, { programId }) => {
      queryClient.invalidateQueries({ queryKey: ['program', programId] })
      queryClient.invalidateQueries({ queryKey: ['programs'] })
    },
  })
}
