import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeftIcon, FunnelIcon, SparklesIcon } from '@heroicons/react/24/outline'
import { useRisksWithSuggestions, useApproveSuggestion, useDismissSuggestion } from '../hooks/useRisks'
import { RiskCard } from '../components/RiskCard'
import { RiskSuggestionCard } from '../components/RiskSuggestionCard'

export function RiskListPage() {
  const { programId } = useParams<{ programId: string }>()
  const [status, setStatus] = useState<string>('')
  const [severity, setSeverity] = useState<string>('')
  const [category, setCategory] = useState<string>('')
  const [showSuggestions, setShowSuggestions] = useState<boolean>(true)

  const { data, isLoading } = useRisksWithSuggestions(programId || '', {
    status: status || undefined,
    severity: severity || undefined,
    category: category || undefined,
  })

  const risks = data?.risks || []
  const suggestions = data?.suggestions || []

  const approveMutation = useApproveSuggestion(programId || '')
  const dismissMutation = useDismissSuggestion(programId || '')

  const handleApprove = (suggestionId: string) => {
    approveMutation.mutate({ suggestionId })
  }

  const handleDismiss = (suggestionId: string, reason: string) => {
    dismissMutation.mutate({ suggestionId, reason })
  }

  const risksByStatus = {
    identified: risks.filter((r) => r.status === 'identified').length,
    assessing: risks.filter((r) => r.status === 'assessing').length,
    mitigating: risks.filter((r) => r.status === 'mitigating').length,
    monitoring: risks.filter((r) => r.status === 'monitoring').length,
    closed: risks.filter((r) => r.status === 'closed').length,
    realized: risks.filter((r) => r.status === 'realized').length,
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="mb-2">
            <Link
              to={`/programs/${programId}/risks`}
              className="inline-flex items-center text-sm text-gray-600 hover:text-gray-900"
            >
              <ArrowLeftIcon className="h-4 w-4 mr-1" />
              Back to Risk Dashboard
            </Link>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">All Risks</h1>
              <p className="mt-1 text-sm text-gray-500">
                Complete risk register for this program
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Stats */}
        <div className="grid grid-cols-2 md:grid-cols-7 gap-4 mb-6">
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Total</p>
            <p className="text-xl font-semibold text-gray-900">{risks.length}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Identified</p>
            <p className="text-xl font-semibold text-gray-600">{risksByStatus.identified}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Assessing</p>
            <p className="text-xl font-semibold text-blue-600">{risksByStatus.assessing}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Mitigating</p>
            <p className="text-xl font-semibold text-yellow-600">{risksByStatus.mitigating}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Monitoring</p>
            <p className="text-xl font-semibold text-purple-600">{risksByStatus.monitoring}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Closed</p>
            <p className="text-xl font-semibold text-green-600">{risksByStatus.closed}</p>
          </div>
          <div className="bg-gradient-to-br from-blue-50 to-blue-100 rounded-lg shadow p-4 border-2 border-blue-200">
            <p className="text-xs text-blue-700 font-medium">AI Suggestions</p>
            <p className="text-xl font-semibold text-blue-900">{suggestions.length}</p>
          </div>
        </div>

        {/* Filters */}
        <div className="bg-white rounded-lg shadow p-4 mb-6">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center space-x-2">
              <FunnelIcon className="h-5 w-5 text-gray-400" />
              <h3 className="text-sm font-medium text-gray-700">Filters</h3>
            </div>

            <label className="flex items-center space-x-2 text-sm">
              <input
                type="checkbox"
                checked={showSuggestions}
                onChange={(e) => setShowSuggestions(e.target.checked)}
                className="rounded text-blue-600 focus:ring-blue-500"
              />
              <span className="text-gray-700">Show AI Suggestions</span>
            </label>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label htmlFor="status" className="block text-sm font-medium text-gray-700 mb-1">
                Status
              </label>
              <select
                id="status"
                value={status}
                onChange={(e) => setStatus(e.target.value)}
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="">All</option>
                <option value="identified">Identified</option>
                <option value="assessing">Assessing</option>
                <option value="mitigating">Mitigating</option>
                <option value="monitoring">Monitoring</option>
                <option value="closed">Closed</option>
                <option value="realized">Realized</option>
              </select>
            </div>

            <div>
              <label htmlFor="severity" className="block text-sm font-medium text-gray-700 mb-1">
                Severity
              </label>
              <select
                id="severity"
                value={severity}
                onChange={(e) => setSeverity(e.target.value)}
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="">All</option>
                <option value="low">Low</option>
                <option value="medium">Medium</option>
                <option value="high">High</option>
                <option value="critical">Critical</option>
              </select>
            </div>

            <div>
              <label htmlFor="category" className="block text-sm font-medium text-gray-700 mb-1">
                Category
              </label>
              <select
                id="category"
                value={category}
                onChange={(e) => setCategory(e.target.value)}
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="">All</option>
                <option value="technical">Technical</option>
                <option value="schedule">Schedule</option>
                <option value="budget">Budget</option>
                <option value="resource">Resource</option>
                <option value="quality">Quality</option>
                <option value="compliance">Compliance</option>
                <option value="external">External</option>
              </select>
            </div>
          </div>
        </div>

        {/* Combined Grid */}
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          </div>
        ) : (
          <div className="space-y-8">
            {/* AI Suggestions Section */}
            {showSuggestions && suggestions.length > 0 && (
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-4 flex items-center">
                  <SparklesIcon className="h-5 w-5 text-blue-600 mr-2" />
                  AI-Recommended Risks ({suggestions.length})
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {suggestions.map((suggestion) => (
                    <RiskSuggestionCard
                      key={suggestion.suggestion_id}
                      suggestion={suggestion}
                      programId={programId || ''}
                      onApprove={handleApprove}
                      onDismiss={handleDismiss}
                    />
                  ))}
                </div>
              </div>
            )}

            {/* Active Risks Section */}
            {risks.length > 0 && (
              <div>
                <h2 className="text-lg font-semibold text-gray-900 mb-4">
                  Active Risks ({risks.length})
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                  {risks.map((risk) => (
                    <RiskCard key={risk.risk_id} risk={risk} programId={programId || ''} />
                  ))}
                </div>
              </div>
            )}

            {/* Empty State */}
            {risks.length === 0 && suggestions.length === 0 && (
              <div className="bg-white rounded-lg shadow p-12 text-center">
                <p className="text-gray-500">No risks or suggestions found</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
