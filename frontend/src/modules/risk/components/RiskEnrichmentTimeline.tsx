import { useState } from 'react'
import { Link } from 'react-router-dom'
import { format } from 'date-fns'
import {
  DocumentTextIcon,
  CheckIcon,
  XMarkIcon,
  SparklesIcon,
  ExclamationCircleIcon,
  InformationCircleIcon,
  ArrowTrendingUpIcon,
  UsersIcon,
} from '@heroicons/react/24/outline'
import { RiskEnrichment } from '@/services/api'
import { useAcceptEnrichment, useRejectEnrichment } from '../hooks/useRisks'

interface RiskEnrichmentTimelineProps {
  riskId: string
  programId: string
  enrichments: RiskEnrichment[]
}

export function RiskEnrichmentTimeline({ riskId, programId, enrichments }: RiskEnrichmentTimelineProps) {
  const acceptMutation = useAcceptEnrichment(programId, riskId)
  const rejectMutation = useRejectEnrichment(programId, riskId)

  // Mock user ID - in production this would come from auth context
  const currentUserId = '00000000-0000-0000-0000-000000000001'

  const handleAccept = (enrichmentId: string) => {
    acceptMutation.mutate({
      enrichmentId,
      reviewedBy: currentUserId,
    })
  }

  const handleReject = (enrichmentId: string) => {
    rejectMutation.mutate({
      enrichmentId,
      reviewedBy: currentUserId,
    })
  }

  const getEnrichmentIcon = (type: string) => {
    switch (type) {
      case 'new_evidence':
        return <DocumentTextIcon className="h-5 w-5 text-blue-600" />
      case 'impact_update':
        return <ArrowTrendingUpIcon className="h-5 w-5 text-orange-600" />
      case 'status_change':
        return <ExclamationCircleIcon className="h-5 w-5 text-yellow-600" />
      case 'mitigation_idea':
        return <InformationCircleIcon className="h-5 w-5 text-green-600" />
      default:
        return <SparklesIcon className="h-5 w-5 text-purple-600" />
    }
  }

  const getEnrichmentLabel = (type: string) => {
    const labels: Record<string, string> = {
      new_evidence: 'New Evidence',
      impact_update: 'Impact Update',
      status_change: 'Status Change',
      mitigation_idea: 'Mitigation Idea',
    }
    return labels[type] || type
  }

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'MMM d, yyyy h:mm a')
    } catch {
      return dateString
    }
  }

  const getPendingEnrichments = () => enrichments.filter((e) => !e.is_relevant || !e.is_relevant.Valid)
  const getAcceptedEnrichments = () => enrichments.filter((e) => e.is_relevant?.Valid && e.is_relevant.Bool === true)
  const getRejectedEnrichments = () => enrichments.filter((e) => e.is_relevant?.Valid && e.is_relevant.Bool === false)

  const pendingCount = getPendingEnrichments().length

  if (enrichments.length === 0) {
    return null
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900 flex items-center">
          <SparklesIcon className="h-5 w-5 text-blue-600 mr-2" />
          Risk Enrichment History
          {pendingCount > 0 && (
            <span className="ml-2 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
              {pendingCount} pending
            </span>
          )}
        </h3>
      </div>

      <div className="space-y-4">
        {enrichments.map((enrichment) => {
          const isPending = !enrichment.is_relevant || !enrichment.is_relevant.Valid
          const isAccepted = enrichment.is_relevant?.Valid && enrichment.is_relevant.Bool === true
          const isRejected = enrichment.is_relevant?.Valid && enrichment.is_relevant.Bool === false

          const borderColor = isPending
            ? 'border-yellow-300'
            : isAccepted
            ? 'border-green-300'
            : 'border-gray-300'
          const bgColor = isPending ? 'bg-yellow-50' : isAccepted ? 'bg-green-50' : 'bg-gray-50'

          return (
            <div key={enrichment.enrichment_id} className={`border-l-4 ${borderColor} ${bgColor} rounded-r-lg p-4`}>
              <div className="flex items-start justify-between">
                <div className="flex items-start space-x-3 flex-1">
                  <div className="flex-shrink-0 mt-1">{getEnrichmentIcon(enrichment.enrichment_type)}</div>

                  <div className="flex-1 min-w-0">
                    <div className="flex items-center space-x-2 mb-1">
                      <span className="text-xs font-medium text-gray-600">
                        {getEnrichmentLabel(enrichment.enrichment_type)}
                      </span>
                      <span className="text-xs text-gray-500">•</span>
                      <span className="text-xs text-gray-500">{formatDate(enrichment.created_at)}</span>
                      {enrichment.match_score < 1 && (
                        <>
                          <span className="text-xs text-gray-500">•</span>
                          <span className="text-xs text-gray-500">
                            {(enrichment.match_score * 100).toFixed(0)}% match
                          </span>
                        </>
                      )}
                    </div>

                    <h4 className="text-sm font-semibold text-gray-900 mb-1">{enrichment.title}</h4>
                    <p className="text-sm text-gray-700 whitespace-pre-wrap">{enrichment.content}</p>

                    <div className="mt-2 flex items-center gap-2 flex-wrap">
                      {/* Source artifact */}
                      <Link
                        to={`/programs/${programId}/artifacts/${enrichment.artifact_id}`}
                        className="text-xs text-blue-600 hover:text-blue-800 hover:underline"
                      >
                        <DocumentTextIcon className="h-3 w-3 inline mr-1" />
                        {enrichment.artifact_filename}
                      </Link>

                      <span className="text-xs text-gray-300">•</span>

                      <span className="text-xs text-gray-500">
                        Method: {enrichment.match_method.replace('_', ' ')}
                      </span>

                      {/* Shared indicator */}
                      {enrichment.related_risk_count > 0 && (
                        <>
                          <span className="text-xs text-gray-300">•</span>
                          <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
                            <UsersIcon className="h-3 w-3 mr-1" />
                            Also relevant to {enrichment.related_risk_count} other risk{enrichment.related_risk_count !== 1 ? 's' : ''}
                          </span>
                        </>
                      )}
                    </div>
                  </div>
                </div>

                <div className="flex-shrink-0 ml-4">
                  {isPending && (
                    <div className="flex space-x-2">
                      <button
                        onClick={() => handleAccept(enrichment.enrichment_id)}
                        disabled={acceptMutation.isPending}
                        className="inline-flex items-center px-3 py-1.5 bg-green-600 text-white text-xs font-medium rounded hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        <CheckIcon className="h-3 w-3 mr-1" />
                        Accept
                      </button>
                      <button
                        onClick={() => handleReject(enrichment.enrichment_id)}
                        disabled={rejectMutation.isPending}
                        className="inline-flex items-center px-3 py-1.5 bg-gray-200 text-gray-700 text-xs font-medium rounded hover:bg-gray-300 disabled:opacity-50 disabled:cursor-not-allowed"
                      >
                        <XMarkIcon className="h-3 w-3 mr-1" />
                        Reject
                      </button>
                    </div>
                  )}
                  {isAccepted && (
                    <div className="flex items-center text-xs text-green-700">
                      <CheckIcon className="h-4 w-4 mr-1" />
                      Accepted
                    </div>
                  )}
                  {isRejected && (
                    <div className="flex items-center text-xs text-gray-500">
                      <XMarkIcon className="h-4 w-4 mr-1" />
                      Rejected
                    </div>
                  )}
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
