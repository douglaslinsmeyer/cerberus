import { Link, useParams, useNavigate } from 'react-router-dom'
import { ArrowLeftIcon, ExclamationTriangleIcon, LightBulbIcon } from '@heroicons/react/24/outline'
import { useRisks, useRiskSuggestions } from '../hooks/useRisks'
import { RiskMatrix } from '../components/RiskMatrix'
import { RiskCard } from '../components/RiskCard'
import { Risk } from '@/services/api'

export function RiskDashboard() {
  const { programId } = useParams<{ programId: string }>()
  const navigate = useNavigate()

  const { data: risks = [], isLoading: risksLoading } = useRisks(programId || '')
  const { data: suggestions = [], isLoading: suggestionsLoading } = useRiskSuggestions(programId || '')

  const pendingSuggestions = suggestions.filter((s) => !s.is_approved && !s.is_dismissed)

  const risksByStatus = {
    identified: risks.filter((r) => r.status === 'identified').length,
    assessing: risks.filter((r) => r.status === 'assessing').length,
    mitigating: risks.filter((r) => r.status === 'mitigating').length,
    monitoring: risks.filter((r) => r.status === 'monitoring').length,
  }

  const risksBySeverity = {
    critical: risks.filter((r) => r.severity === 'critical').length,
    high: risks.filter((r) => r.severity === 'high').length,
    medium: risks.filter((r) => r.severity === 'medium').length,
    low: risks.filter((r) => r.severity === 'low').length,
  }

  const handleRiskClick = (risk: Risk) => {
    navigate(`/programs/${programId}/risks/${risk.risk_id}`)
  }

  if (risksLoading || suggestionsLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="mb-2">
            <Link
              to={`/programs/${programId}`}
              className="inline-flex items-center text-sm text-gray-600 hover:text-gray-900"
            >
              <ArrowLeftIcon className="h-4 w-4 mr-1" />
              Back to Program Dashboard
            </Link>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Risk & Issues Register</h1>
              <p className="mt-1 text-sm text-gray-500">
                AI-powered risk identification and mitigation tracking
              </p>
            </div>

            <Link
              to={`/programs/${programId}/risks`}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
            >
              View All Risks
            </Link>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Summary Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <ExclamationTriangleIcon className="h-8 w-8 text-gray-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Total Risks</p>
                <p className="text-2xl font-semibold text-gray-900">{risks.length}</p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <ExclamationTriangleIcon className="h-8 w-8 text-red-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Critical</p>
                <p className="text-2xl font-semibold text-red-600">{risksBySeverity.critical}</p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <ExclamationTriangleIcon className="h-8 w-8 text-orange-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">High</p>
                <p className="text-2xl font-semibold text-orange-600">{risksBySeverity.high}</p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <LightBulbIcon className="h-8 w-8 text-blue-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">AI Suggestions</p>
                <p className="text-2xl font-semibold text-blue-600">{pendingSuggestions.length}</p>
              </div>
            </div>
          </div>
        </div>

        {/* Risk Matrix */}
        <div className="mb-8">
          <RiskMatrix risks={risks} onRiskClick={handleRiskClick} />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          {/* AI Suggestions */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">AI Risk Suggestions</h3>
              {pendingSuggestions.length > 0 && (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                  {pendingSuggestions.length} pending
                </span>
              )}
            </div>

            {pendingSuggestions.length === 0 ? (
              <div className="text-center text-gray-500 py-8">
                <LightBulbIcon className="h-12 w-12 mx-auto text-gray-300 mb-2" />
                <p>No pending risk suggestions</p>
              </div>
            ) : (
              <div className="space-y-3">
                {pendingSuggestions.slice(0, 5).map((suggestion) => (
                  <div
                    key={suggestion.suggestion_id}
                    className="p-4 bg-blue-50 border border-blue-200 rounded-lg"
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <h4 className="text-sm font-semibold text-gray-900">{suggestion.title}</h4>
                        <p className="text-sm text-gray-600 mt-1">{suggestion.description}</p>
                        <p className="text-xs text-gray-500 mt-2">{suggestion.rationale}</p>

                        <div className="mt-3 flex items-center space-x-2">
                          <span className="text-xs text-gray-600">
                            Severity: <strong>{suggestion.suggested_severity}</strong>
                          </span>
                          {suggestion.ai_confidence_score?.Valid && (
                            <>
                              <span>•</span>
                              <span className="text-xs text-gray-600">
                                Confidence: {(suggestion.ai_confidence_score.Float64 * 100).toFixed(0)}%
                              </span>
                            </>
                          )}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}

                {pendingSuggestions.length > 5 && (
                  <div className="text-center pt-2">
                    <Link
                      to={`/programs/${programId}/risks/suggestions`}
                      className="text-sm text-blue-600 hover:text-blue-700"
                    >
                      View all {pendingSuggestions.length} suggestions
                    </Link>
                  </div>
                )}
              </div>
            )}
          </div>

          {/* Status Breakdown */}
          <div className="bg-white rounded-lg shadow p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Risk Status</h3>

            <div className="space-y-4">
              <div>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium text-gray-700">Identified</span>
                  <span className="text-sm font-semibold text-gray-900">{risksByStatus.identified}</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-gray-500 h-2 rounded-full"
                    style={{ width: `${(risksByStatus.identified / risks.length) * 100}%` }}
                  ></div>
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium text-gray-700">Assessing</span>
                  <span className="text-sm font-semibold text-gray-900">{risksByStatus.assessing}</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-500 h-2 rounded-full"
                    style={{ width: `${(risksByStatus.assessing / risks.length) * 100}%` }}
                  ></div>
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium text-gray-700">Mitigating</span>
                  <span className="text-sm font-semibold text-gray-900">{risksByStatus.mitigating}</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-yellow-500 h-2 rounded-full"
                    style={{ width: `${(risksByStatus.mitigating / risks.length) * 100}%` }}
                  ></div>
                </div>
              </div>

              <div>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-medium text-gray-700">Monitoring</span>
                  <span className="text-sm font-semibold text-gray-900">{risksByStatus.monitoring}</span>
                </div>
                <div className="w-full bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-purple-500 h-2 rounded-full"
                    style={{ width: `${(risksByStatus.monitoring / risks.length) * 100}%` }}
                  ></div>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Top Risks */}
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-lg font-semibold text-gray-900">Top Risks</h3>
            <Link
              to={`/programs/${programId}/risks`}
              className="text-sm text-blue-600 hover:text-blue-700"
            >
              View All
            </Link>
          </div>

          {risks.length === 0 ? (
            <div className="text-center text-gray-500 py-8">
              <p>No risks identified yet</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {risks
                .filter((r) => r.severity === 'critical' || r.severity === 'high')
                .slice(0, 4)
                .map((risk) => (
                  <RiskCard key={risk.risk_id} risk={risk} programId={programId || ''} />
                ))}
            </div>
          )}
        </div>

        {/* Alert */}
        {(risksBySeverity.critical > 0 || risksBySeverity.high > 0) && (
          <div className="mt-8 bg-red-50 border border-red-200 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-red-900 mb-2">High-Priority Risks Detected</h3>
            <p className="text-sm text-red-700">
              There are {risksBySeverity.critical + risksBySeverity.high} high-priority risks requiring
              immediate attention. Review and assign mitigation plans as needed.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
