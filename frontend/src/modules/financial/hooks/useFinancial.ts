import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { financialApi } from '@/services/api'

// Hook for listing rate cards
export function useRateCards(programId: string, params?: { limit?: number; offset?: number }) {
  return useQuery({
    queryKey: ['rateCards', programId, params],
    queryFn: () => financialApi.listRateCards(programId, params),
    enabled: !!programId,
  })
}

// Hook for getting a single rate card with items
export function useRateCard(programId: string, rateCardId: string) {
  return useQuery({
    queryKey: ['rateCard', programId, rateCardId],
    queryFn: () => financialApi.getRateCard(programId, rateCardId),
    enabled: !!programId && !!rateCardId,
  })
}

// Hook for creating a rate card
export function useCreateRateCard(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (request: any) => financialApi.createRateCard(programId, request),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rateCards', programId] })
    },
  })
}

// Hook for deleting a rate card
export function useDeleteRateCard(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (rateCardId: string) => financialApi.deleteRateCard(programId, rateCardId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['rateCards', programId] })
    },
  })
}

// Hook for listing invoices
export function useInvoices(programId: string, params?: {
  processing_status?: string
  payment_status?: string
  vendor_name?: string
  date_from?: string
  date_to?: string
  limit?: number
  offset?: number
}) {
  return useQuery({
    queryKey: ['invoices', programId, params],
    queryFn: () => financialApi.listInvoices(programId, params),
    enabled: !!programId,
  })
}

// Hook for getting a single invoice with line items and variances
export function useInvoice(programId: string, invoiceId: string) {
  return useQuery({
    queryKey: ['invoice', programId, invoiceId],
    queryFn: () => financialApi.getInvoice(programId, invoiceId),
    enabled: !!programId && !!invoiceId,
  })
}

// Hook for processing an invoice
export function useProcessInvoice(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (artifactId: string) => financialApi.processInvoice(programId, artifactId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['invoices', programId] })
    },
  })
}

// Hook for approving an invoice
export function useApproveInvoice(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (invoiceId: string) => financialApi.approveInvoice(programId, invoiceId),
    onSuccess: (_, invoiceId) => {
      queryClient.invalidateQueries({ queryKey: ['invoice', programId, invoiceId] })
      queryClient.invalidateQueries({ queryKey: ['invoices', programId] })
    },
  })
}

// Hook for rejecting an invoice
export function useRejectInvoice(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ invoiceId, reason }: { invoiceId: string; reason: string }) =>
      financialApi.rejectInvoice(programId, invoiceId, reason),
    onSuccess: (_, { invoiceId }) => {
      queryClient.invalidateQueries({ queryKey: ['invoice', programId, invoiceId] })
      queryClient.invalidateQueries({ queryKey: ['invoices', programId] })
    },
  })
}

// Hook for listing variances
export function useVariances(programId: string, params?: { severity?: string }) {
  return useQuery({
    queryKey: ['variances', programId, params],
    queryFn: () => financialApi.listVariances(programId, params),
    enabled: !!programId,
  })
}

// Hook for dismissing a variance
export function useDismissVariance(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ varianceId, reason }: { varianceId: string; reason: string }) =>
      financialApi.dismissVariance(programId, varianceId, reason),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['variances', programId] })
    },
  })
}

// Hook for resolving a variance
export function useResolveVariance(programId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({ varianceId, notes }: { varianceId: string; notes: string }) =>
      financialApi.resolveVariance(programId, varianceId, notes),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['variances', programId] })
    },
  })
}

// Hook for budget status
export function useBudgetStatus(programId: string, fiscalYear?: number) {
  return useQuery({
    queryKey: ['budgetStatus', programId, fiscalYear],
    queryFn: () => financialApi.getBudgetStatus(programId, fiscalYear),
    enabled: !!programId,
  })
}

// Hook for listing budget categories
export function useBudgetCategories(programId: string, fiscalYear?: number) {
  return useQuery({
    queryKey: ['budgetCategories', programId, fiscalYear],
    queryFn: () => financialApi.listBudgetCategories(programId, fiscalYear),
    enabled: !!programId,
  })
}
