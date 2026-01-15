import { useState } from 'react'
import { SparklesIcon, ExclamationTriangleIcon, CheckIcon, XMarkIcon } from '@heroicons/react/24/outline'
import { RiskSuggestion } from '@/services/api'

interface RiskSuggestionCardProps {
  suggestion: RiskSuggestion
  programId: string
  onApprove: (suggestionId: string) => void
  onDismiss: (suggestionId: string, reason: string) => void
}

export function RiskSuggestionCard({ suggestion, onApprove, onDismiss }: RiskSuggestionCardProps) {
  const [showDismissDialog, setShowDismissDialog] = useState(false)
  const [dismissReason, setDismissReason] = useState('not_a_risk')
  const [dismissReasonText, setDismissReasonText] = useState('')

  const severityColors = {
    low: 'border-green-500 bg-green-50',
    medium: 'border-yellow-500 bg-yellow-50',
    high: 'border-orange-500 bg-orange-50',
    critical: 'border-red-500 bg-red-50',
  }

  const severityBadgeColors = {
    low: 'bg-green-100 text-green-800 border-green-300',
    medium: 'bg-yellow-100 text-yellow-800 border-yellow-300',
    high: 'bg-orange-100 text-orange-800 border-orange-300',
    critical: 'bg-red-100 text-red-800 border-red-300',
  }

  const formatLabel = (value: string) => {
    return value.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
  }

  const handleApprove = () => {
    onApprove(suggestion.suggestion_id)
  }

  const handleDismiss = () => {
    const reason = dismissReason === 'other' ? dismissReasonText : dismissReason
    onDismiss(suggestion.suggestion_id, reason)
    setShowDismissDialog(false)
    setDismissReason('not_a_risk')
    setDismissReasonText('')
  }

  return (
    <div className={`relative bg-blue-50 border-2 border-blue-200 rounded-lg shadow p-6 ${severityColors[suggestion.suggested_severity as keyof typeof severityColors]}`}>
      {/* AI Badge */}
      <div className="absolute top-3 right-3">
        <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-blue-100 text-blue-800 border border-blue-300">
          <SparklesIcon className="h-3 w-3 mr-1" />
          AI Suggestion
        </span>
      </div>

      <div className="flex items-start space-x-3">
        <div className={`p-2 rounded-lg border-2 ${severityBadgeColors[suggestion.suggested_severity as keyof typeof severityBadgeColors]}`}>
          <ExclamationTriangleIcon className="h-5 w-5" />
        </div>

        <div className="flex-1 min-w-0 pr-24">
          <h3 className="text-sm font-semibold text-gray-900">{suggestion.title}</h3>

          <p className="text-sm text-gray-700 mt-2">{suggestion.description}</p>

          <div className="mt-3 flex flex-wrap items-center gap-2">
            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${severityBadgeColors[suggestion.suggested_severity as keyof typeof severityBadgeColors]}`}>
              {formatLabel(suggestion.suggested_severity)}
            </span>
            <span className="inline-flex items-center text-xs text-gray-600">
              <span className="font-medium text-gray-700">P:</span> {formatLabel(suggestion.suggested_probability)}
            </span>
            <span className="inline-flex items-center text-xs text-gray-600">
              <span className="font-medium text-gray-700">I:</span> {formatLabel(suggestion.suggested_impact)}
            </span>
            <span className="inline-flex items-center text-xs text-gray-600">
              <span className="font-medium text-gray-700">Category:</span> {formatLabel(suggestion.suggested_category)}
            </span>
          </div>

          {/* AI Rationale */}
          <div className="mt-3 text-sm bg-white p-3 rounded border border-blue-200">
            <p className="font-medium text-gray-900 text-xs">AI Analysis:</p>
            <p className="text-xs mt-1 text-gray-700">{suggestion.rationale}</p>
            {suggestion.ai_confidence_score?.Valid && (
              <p className="text-xs text-gray-500 mt-1">
                Confidence: {(suggestion.ai_confidence_score.Float64 * 100).toFixed(0)}%
              </p>
            )}
          </div>

          {/* Action Buttons */}
          <div className="mt-4 flex space-x-2">
            <button
              onClick={handleApprove}
              className="flex-1 inline-flex items-center justify-center px-4 py-2 bg-green-600 text-white text-sm font-medium rounded hover:bg-green-700 transition-colors"
            >
              <CheckIcon className="h-4 w-4 mr-1" />
              Approve as Risk
            </button>
            <button
              onClick={() => setShowDismissDialog(true)}
              className="flex-1 inline-flex items-center justify-center px-4 py-2 bg-gray-200 text-gray-700 text-sm font-medium rounded hover:bg-gray-300 transition-colors"
            >
              <XMarkIcon className="h-4 w-4 mr-1" />
              Dismiss
            </button>
          </div>
        </div>
      </div>

      {/* Dismiss Dialog */}
      {showDismissDialog && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl p-6 max-w-md w-full mx-4">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Dismiss Suggestion</h3>
            <p className="text-sm text-gray-600 mb-4">
              Please select a reason for dismissing this AI risk suggestion:
            </p>

            <div className="space-y-2 mb-4">
              <label className="flex items-center">
                <input
                  type="radio"
                  value="not_a_risk"
                  checked={dismissReason === 'not_a_risk'}
                  onChange={(e) => setDismissReason(e.target.value)}
                  className="mr-2"
                />
                <span className="text-sm">Not a risk</span>
              </label>
              <label className="flex items-center">
                <input
                  type="radio"
                  value="duplicate"
                  checked={dismissReason === 'duplicate'}
                  onChange={(e) => setDismissReason(e.target.value)}
                  className="mr-2"
                />
                <span className="text-sm">Duplicate</span>
              </label>
              <label className="flex items-center">
                <input
                  type="radio"
                  value="already_mitigated"
                  checked={dismissReason === 'already_mitigated'}
                  onChange={(e) => setDismissReason(e.target.value)}
                  className="mr-2"
                />
                <span className="text-sm">Already mitigated</span>
              </label>
              <label className="flex items-center">
                <input
                  type="radio"
                  value="other"
                  checked={dismissReason === 'other'}
                  onChange={(e) => setDismissReason(e.target.value)}
                  className="mr-2"
                />
                <span className="text-sm">Other</span>
              </label>
            </div>

            {dismissReason === 'other' && (
              <textarea
                value={dismissReasonText}
                onChange={(e) => setDismissReasonText(e.target.value)}
                placeholder="Please provide a reason..."
                className="w-full px-3 py-2 border border-gray-300 rounded text-sm mb-4"
                rows={3}
              />
            )}

            <div className="flex space-x-2">
              <button
                onClick={handleDismiss}
                disabled={dismissReason === 'other' && !dismissReasonText.trim()}
                className="flex-1 px-4 py-2 bg-red-600 text-white text-sm font-medium rounded hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Confirm Dismiss
              </button>
              <button
                onClick={() => {
                  setShowDismissDialog(false)
                  setDismissReason('not_a_risk')
                  setDismissReasonText('')
                }}
                className="flex-1 px-4 py-2 bg-gray-200 text-gray-700 text-sm font-medium rounded hover:bg-gray-300"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
