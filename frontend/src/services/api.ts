import axios from 'axios'

const API_URL = (import.meta as any).env?.VITE_API_URL || 'http://localhost:8080/api/v1'

const api = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Nullable types from Go sql.Null* structures
export interface NullString {
  String: string
  Valid: boolean
}

export interface NullInt32 {
  Int32: number
  Valid: boolean
}

export interface NullFloat64 {
  Float64: number
  Valid: boolean
}

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
  person_role?: NullString
  person_organization?: NullString
  confidence_score: NullFloat64
}

export interface Fact {
  fact_id: string
  fact_type: string
  fact_key: string
  fact_value: string
  confidence_score: NullFloat64
}

export interface Insight {
  insight_id: string
  insight_type: string
  title: string
  description: string
  severity?: NullString
  suggested_action?: NullString
  confidence_score: NullFloat64
}

export interface ArtifactSummary {
  executive_summary: string
  sentiment?: NullString
  priority?: NullInt32
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

// ===========================
// FINANCIAL MODULE TYPES
// ===========================

export interface RateCard {
  rate_card_id: string
  program_id: string
  name: string
  description?: NullString
  effective_start_date: string
  effective_end_date?: NullString
  currency: string
  is_active: boolean
  created_at: string
  created_by: string
  updated_at: string
  updated_by?: NullString
  deleted_at?: NullString
}

export interface RateCardItem {
  item_id: string
  rate_card_id: string
  person_name?: NullString
  role_title?: NullString
  seniority_level?: NullString
  rate_type: string
  rate_amount: number
  currency: string
  expected_hours_per_week?: NullFloat64
  expected_hours_per_month?: NullFloat64
  notes?: NullString
  created_at: string
}

export interface RateCardWithItems extends RateCard {
  items?: RateCardItem[]
}

export interface Invoice {
  invoice_id: string
  program_id: string
  artifact_id?: NullString
  invoice_number?: NullString
  vendor_name: string
  vendor_id?: NullString
  invoice_date: string
  due_date?: NullString
  period_start_date?: NullString
  period_end_date?: NullString
  subtotal_amount?: NullFloat64
  tax_amount?: NullFloat64
  total_amount: number
  currency: string
  processing_status: 'pending' | 'processing' | 'completed' | 'failed'
  payment_status: 'pending' | 'approved' | 'rejected' | 'paid'
  ai_model_version?: NullString
  ai_confidence_score?: NullFloat64
  ai_processing_time_ms?: NullInt32
  submitted_by?: NullString
  submitted_at: string
  approved_by?: NullString
  approved_at?: NullString
  rejected_reason?: NullString
  deleted_at?: NullString
}

export interface InvoiceLineItem {
  line_item_id: string
  invoice_id: string
  line_number: number
  description: string
  quantity?: NullFloat64
  unit_rate?: NullFloat64
  line_amount: number
  person_name?: NullString
  role_description?: NullString
  matched_rate_card_item_id?: NullString
  expected_rate?: NullFloat64
  rate_variance_amount?: NullFloat64
  rate_variance_percentage?: NullFloat64
  billed_hours?: NullFloat64
  expected_hours?: NullFloat64
  hours_variance?: NullFloat64
  spend_category?: NullString
  budget_category_id?: NullString
  has_variance: boolean
  variance_severity?: NullString
  needs_review: boolean
  review_notes?: NullString
  ai_confidence_score?: NullFloat64
}

export interface FinancialVariance {
  variance_id: string
  program_id: string
  invoice_id?: NullString
  line_item_id?: NullString
  variance_type: string
  severity: 'low' | 'medium' | 'high' | 'critical'
  title: string
  description: string
  expected_value?: NullFloat64
  actual_value?: NullFloat64
  variance_amount?: NullFloat64
  variance_percentage?: NullFloat64
  source_artifact_ids: string[]
  conflicting_values?: any
  ai_confidence_score?: NullFloat64
  ai_detected_at: string
  is_dismissed: boolean
  dismissed_by?: NullString
  dismissed_at?: NullString
  dismissal_reason?: NullString
  resolution_notes?: NullString
  resolved_at?: NullString
  resolved_by?: NullString
}

export interface InvoiceWithMetadata extends Invoice {
  line_items?: InvoiceLineItem[]
  variances?: FinancialVariance[]
}

export interface BudgetCategory {
  category_id: string
  program_id: string
  category_name: string
  description?: NullString
  budgeted_amount: number
  currency: string
  fiscal_year: number
  fiscal_quarter?: NullInt32
  actual_spend: number
  committed_spend: number
  variance_amount: number
  variance_percentage: number
  created_at: string
  updated_at: string
  deleted_at?: NullString
}

export interface CreateRateCardRequest {
  name: string
  description?: string
  effective_start_date: string
  effective_end_date?: string
  currency: string
  items: {
    person_name?: string
    role_title?: string
    seniority_level?: string
    rate_type: string
    rate_amount: number
    currency: string
    expected_hours_per_week?: number
    expected_hours_per_month?: number
    notes?: string
  }[]
}

// Financial API
export const financialApi = {
  // Rate Cards
  listRateCards: async (programId: string, params?: { limit?: number; offset?: number }) => {
    const response = await api.get(`/programs/${programId}/financial/rate-cards`, { params })
    return response.data.data.rate_cards as RateCard[]
  },

  getRateCard: async (programId: string, rateCardId: string) => {
    const response = await api.get(`/programs/${programId}/financial/rate-cards/${rateCardId}`)
    return response.data.data as RateCardWithItems
  },

  createRateCard: async (programId: string, request: CreateRateCardRequest) => {
    const response = await api.post(`/programs/${programId}/financial/rate-cards`, request)
    return response.data.data
  },

  updateRateCard: async (programId: string, rateCardId: string, rateCard: Partial<RateCard>) => {
    await api.put(`/programs/${programId}/financial/rate-cards/${rateCardId}`, rateCard)
  },

  deleteRateCard: async (programId: string, rateCardId: string) => {
    await api.delete(`/programs/${programId}/financial/rate-cards/${rateCardId}`)
  },

  // Invoices
  listInvoices: async (programId: string, params?: {
    processing_status?: string
    payment_status?: string
    vendor_name?: string
    date_from?: string
    date_to?: string
    limit?: number
    offset?: number
  }) => {
    const response = await api.get(`/programs/${programId}/financial/invoices`, { params })
    return response.data.data.invoices as Invoice[]
  },

  getInvoice: async (programId: string, invoiceId: string) => {
    const response = await api.get(`/programs/${programId}/financial/invoices/${invoiceId}`)
    return response.data.data as InvoiceWithMetadata
  },

  processInvoice: async (programId: string, artifactId: string) => {
    const response = await api.post(`/programs/${programId}/financial/invoices/process`, {
      artifact_id: artifactId,
    })
    return response.data.data
  },

  approveInvoice: async (programId: string, invoiceId: string) => {
    await api.post(`/programs/${programId}/financial/invoices/${invoiceId}/approve`)
  },

  rejectInvoice: async (programId: string, invoiceId: string, reason: string) => {
    await api.post(`/programs/${programId}/financial/invoices/${invoiceId}/reject`, { reason })
  },

  // Variances
  listVariances: async (programId: string, params?: { severity?: string }) => {
    const response = await api.get(`/programs/${programId}/financial/variances`, { params })
    return response.data.data.variances as FinancialVariance[]
  },

  dismissVariance: async (programId: string, varianceId: string, reason: string) => {
    await api.post(`/programs/${programId}/financial/variances/${varianceId}/dismiss`, { reason })
  },

  resolveVariance: async (programId: string, varianceId: string, notes: string) => {
    await api.post(`/programs/${programId}/financial/variances/${varianceId}/resolve`, { notes })
  },

  // Budget
  getBudgetStatus: async (programId: string, fiscalYear?: number) => {
    const params = fiscalYear ? { fiscal_year: fiscalYear } : undefined
    const response = await api.get(`/programs/${programId}/financial/budget/status`, { params })
    return response.data.data as {
      categories: BudgetCategory[]
      total_budgeted: number
      total_actual: number
      total_committed: number
      total_variance: number
      variance_percentage: number
    }
  },

  listBudgetCategories: async (programId: string, fiscalYear?: number) => {
    const params = fiscalYear ? { fiscal_year: fiscalYear } : undefined
    const response = await api.get(`/programs/${programId}/financial/budget/categories`, { params })
    return response.data.data.categories as BudgetCategory[]
  },
}

// ===========================
// RISK MODULE TYPES
// ===========================

export interface Risk {
  risk_id: string
  program_id: string
  title: string
  description: string
  probability: 'very_low' | 'low' | 'medium' | 'high' | 'very_high'
  impact: 'very_low' | 'low' | 'medium' | 'high' | 'very_high'
  severity: 'low' | 'medium' | 'high' | 'critical'
  category: string
  status: 'identified' | 'assessing' | 'mitigating' | 'monitoring' | 'closed' | 'realized'
  owner_user_id?: NullString
  owner_name?: NullString
  identified_date: string
  target_resolution_date?: NullString
  closed_date?: NullString
  realized_date?: NullString
  ai_confidence_score?: NullFloat64
  ai_detected_at?: NullString
  created_by: string
  created_at: string
  updated_at: string
  deleted_at?: NullString
}

export interface RiskSuggestion {
  suggestion_id: string
  program_id: string
  title: string
  description: string
  rationale: string
  suggested_probability: string
  suggested_impact: string
  suggested_severity: string
  suggested_category: string
  source_type: string
  source_artifact_ids: string[]
  source_insight_id?: NullString
  source_variance_id?: NullString
  ai_confidence_score?: NullFloat64
  ai_detected_at: string
  is_approved: boolean
  is_dismissed: boolean
  approved_by?: NullString
  approved_at?: NullString
  dismissed_by?: NullString
  dismissed_at?: NullString
  dismissal_reason?: NullString
  created_risk_id?: NullString
}

export interface RiskMitigation {
  mitigation_id: string
  risk_id: string
  strategy: 'avoid' | 'mitigate' | 'transfer' | 'accept'
  action_description: string
  expected_probability_reduction?: NullString
  expected_impact_reduction?: NullString
  effectiveness_rating?: NullInt32
  status: 'planned' | 'in_progress' | 'completed' | 'cancelled'
  assigned_to?: NullString
  target_completion_date?: NullString
  actual_completion_date?: NullString
  estimated_cost?: NullFloat64
  actual_cost?: NullFloat64
  currency: string
  created_by: string
  created_at: string
  updated_at: string
  deleted_at?: NullString
}

export interface RiskArtifactLink {
  link_id: string
  risk_id: string
  artifact_id: string
  link_type: string
  description?: NullString
  created_by: string
  created_at: string
}

export interface ConversationThread {
  thread_id: string
  risk_id: string
  title: string
  thread_type: string
  is_resolved: boolean
  resolved_at?: NullString
  resolved_by?: NullString
  message_count: number
  last_message_at?: NullString
  created_by: string
  created_at: string
  deleted_at?: NullString
}

export interface ConversationMessage {
  message_id: string
  thread_id: string
  message_text: string
  message_format: 'plain' | 'markdown'
  mentioned_user_ids: string[]
  created_by: string
  created_at: string
  edited_at?: NullString
  deleted_at?: NullString
}

export interface RiskWithMetadata extends Risk {
  mitigations?: RiskMitigation[]
  linked_artifacts?: RiskArtifactLink[]
  threads?: ConversationThread[]
}

export interface ThreadWithMessages extends ConversationThread {
  messages?: ConversationMessage[]
}

export interface CreateRiskRequest {
  title: string
  description: string
  probability: string
  impact: string
  category: string
  owner_user_id?: string
  owner_name?: string
  target_resolution_date?: string
}

export interface CreateMitigationRequest {
  strategy: string
  action_description: string
  expected_probability_reduction?: string
  expected_impact_reduction?: string
  assigned_to?: string
  target_completion_date?: string
  estimated_cost?: number
  currency: string
}

// Risk API
export const riskApi = {
  // Risks
  listRisks: async (programId: string, params?: {
    status?: string
    category?: string
    severity?: string
    owner_user_id?: string
    limit?: number
    offset?: number
  }) => {
    const response = await api.get(`/programs/${programId}/risks`, { params })
    return response.data.data.risks as Risk[]
  },

  getRisk: async (programId: string, riskId: string) => {
    const response = await api.get(`/programs/${programId}/risks/${riskId}`)
    return response.data.data as RiskWithMetadata
  },

  createRisk: async (programId: string, request: CreateRiskRequest) => {
    const response = await api.post(`/programs/${programId}/risks`, request)
    return response.data.data
  },

  updateRisk: async (programId: string, riskId: string, updates: Partial<Risk>) => {
    await api.put(`/programs/${programId}/risks/${riskId}`, updates)
  },

  deleteRisk: async (programId: string, riskId: string) => {
    await api.delete(`/programs/${programId}/risks/${riskId}`)
  },

  // Suggestions
  listSuggestions: async (programId: string) => {
    const response = await api.get(`/programs/${programId}/risks/suggestions`)
    return response.data.data.suggestions as RiskSuggestion[]
  },

  approveSuggestion: async (programId: string, suggestionId: string, data?: {
    owner_user_id?: string
    target_resolution_date?: string
    override_probability?: string
    override_impact?: string
    override_category?: string
  }) => {
    const response = await api.post(`/programs/${programId}/risks/suggestions/${suggestionId}/approve`, data)
    return response.data.data
  },

  dismissSuggestion: async (programId: string, suggestionId: string, reason: string) => {
    await api.post(`/programs/${programId}/risks/suggestions/${suggestionId}/dismiss`, { reason })
  },

  // Mitigations
  listMitigations: async (programId: string, riskId: string) => {
    const response = await api.get(`/programs/${programId}/risks/${riskId}/mitigations`)
    return response.data.data.mitigations as RiskMitigation[]
  },

  createMitigation: async (programId: string, riskId: string, request: CreateMitigationRequest) => {
    const response = await api.post(`/programs/${programId}/risks/${riskId}/mitigations`, request)
    return response.data.data
  },

  updateMitigation: async (programId: string, riskId: string, mitigationId: string, updates: Partial<RiskMitigation>) => {
    await api.put(`/programs/${programId}/risks/${riskId}/mitigations/${mitigationId}`, updates)
  },

  deleteMitigation: async (programId: string, riskId: string, mitigationId: string) => {
    await api.delete(`/programs/${programId}/risks/${riskId}/mitigations/${mitigationId}`)
  },

  // Artifact Links
  getLinkedArtifacts: async (programId: string, riskId: string) => {
    const response = await api.get(`/programs/${programId}/risks/${riskId}/artifacts`)
    return response.data.data.artifacts as RiskArtifactLink[]
  },

  linkArtifact: async (programId: string, riskId: string, artifactId: string, linkType: string, description?: string) => {
    const response = await api.post(`/programs/${programId}/risks/${riskId}/artifacts`, {
      artifact_id: artifactId,
      link_type: linkType,
      description,
    })
    return response.data.data
  },

  unlinkArtifact: async (programId: string, riskId: string, linkId: string) => {
    await api.delete(`/programs/${programId}/risks/${riskId}/artifacts/${linkId}`)
  },

  // Conversations
  listThreads: async (programId: string, riskId: string) => {
    const response = await api.get(`/programs/${programId}/risks/${riskId}/threads`)
    return response.data.data.threads as ConversationThread[]
  },

  createThread: async (programId: string, riskId: string, title: string, threadType: string) => {
    const response = await api.post(`/programs/${programId}/risks/${riskId}/threads`, {
      title,
      thread_type: threadType,
    })
    return response.data.data
  },

  getThreadMessages: async (programId: string, riskId: string, threadId: string) => {
    const response = await api.get(`/programs/${programId}/risks/${riskId}/threads/${threadId}/messages`)
    return response.data.data as ThreadWithMessages
  },

  addMessage: async (programId: string, riskId: string, threadId: string, messageText: string, messageFormat: string = 'markdown') => {
    const response = await api.post(`/programs/${programId}/risks/${riskId}/threads/${threadId}/messages`, {
      message_text: messageText,
      message_format: messageFormat,
    })
    return response.data.data
  },
}

export default api
