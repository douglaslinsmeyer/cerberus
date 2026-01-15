import { useState } from 'react'
import { GroupedSuggestion } from '@/services/api'
import {
  UserGroupIcon,
  SparklesIcon,
  ExclamationTriangleIcon,
  ChevronDownIcon,
  ChevronUpIcon,
} from '@heroicons/react/24/outline'
import { MergeDossierModal } from './MergeDossierModal'
import { useRejectMergeGroup } from '../hooks/useStakeholders'

interface GroupedSuggestionCardProps {
  group: GroupedSuggestion
  programId: string
  onSuccess: () => void
}

export function GroupedSuggestionCard({ group, programId, onSuccess }: GroupedSuggestionCardProps) {
  const [showMergeModal, setShowMergeModal] = useState(false)
  const [showDetails, setShowDetails] = useState(false)
  const rejectMutation = useRejectMergeGroup(programId)

  const handleReject = async () => {
    if (confirm(`Mark these ${group.total_persons} mentions as different people?`)) {
      try {
        await rejectMutation.mutateAsync(group.group_id)
        onSuccess()
      } catch (error) {
        console.error('Failed to reject merge group:', error)
        alert('Failed to reject grouping. Please try again.')
      }
    }
  }

  const avgConfidencePercent = Math.round(group.average_confidence * 100)

  return (
    <>
      <div className="bg-white rounded-lg shadow p-6 hover:shadow-md transition-shadow">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <UserGroupIcon className="h-6 w-6 text-purple-600" />
              <h3 className="text-lg font-semibold text-gray-900">{group.suggested_name}</h3>
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-purple-100 text-purple-800">
                <SparklesIcon className="h-3 w-3 mr-1" />
                {group.total_persons} grouped
              </span>
            </div>

            {/* Conflict Indicators */}
            <div className="flex flex-wrap gap-2 mb-3">
              {group.has_role_conflicts && (
                <span className="inline-flex items-center px-2 py-1 rounded text-xs font-medium bg-yellow-100 text-yellow-800">
                  <ExclamationTriangleIcon className="h-3 w-3 mr-1" />
                  {group.role_options?.length || 0} roles found
                </span>
              )}
              {group.has_org_conflicts && (
                <span className="inline-flex items-center px-2 py-1 rounded text-xs font-medium bg-yellow-100 text-yellow-800">
                  <ExclamationTriangleIcon className="h-3 w-3 mr-1" />
                  {group.org_options?.length || 0} organizations found
                </span>
              )}
            </div>

            {/* Stats */}
            <div className="space-y-1 mb-3">
              <p className="text-sm text-gray-600">
                <span className="font-medium">Mentioned in:</span> {group.total_artifacts} document
                {group.total_artifacts !== 1 ? 's' : ''} ({group.total_mentions} total mentions)
              </p>
              {avgConfidencePercent > 0 && (
                <p className="text-sm text-gray-600">
                  <span className="font-medium">Average Confidence:</span> {avgConfidencePercent}%
                </p>
              )}
            </div>

            {/* Expandable Details */}
            <button
              onClick={() => setShowDetails(!showDetails)}
              className="text-sm font-medium text-blue-600 hover:text-blue-700 flex items-center mb-3"
            >
              {showDetails ? (
                <>
                  <ChevronUpIcon className="h-4 w-4 mr-1" />
                  Hide Details
                </>
              ) : (
                <>
                  <ChevronDownIcon className="h-4 w-4 mr-1" />
                  View Details ({group.members.length} variants)
                </>
              )}
            </button>

            {showDetails && (
              <div className="mb-4 p-4 bg-gray-50 rounded-lg space-y-3">
                <h5 className="text-xs font-semibold text-gray-700 uppercase tracking-wide mb-2">
                  Person Variants
                </h5>
                {group.members.map((member) => (
                  <div key={member.person_id} className="border-l-2 border-blue-300 pl-3 text-sm">
                    <p className="font-medium text-gray-900">{member.person_name}</p>
                    {member.person_role?.Valid && (
                      <p className="text-gray-600">Role: {member.person_role.String}</p>
                    )}
                    {member.person_organization?.Valid && (
                      <p className="text-gray-600">Org: {member.person_organization.String}</p>
                    )}
                    <p className="text-xs text-gray-500">
                      {member.artifact_count} doc{member.artifact_count !== 1 ? 's' : ''} • {member.mention_count}{' '}
                      mention{member.mention_count !== 1 ? 's' : ''} • {Math.round(member.confidence_score * 100)}%
                      confident
                    </p>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex gap-2 mt-4 pt-4 border-t border-gray-200">
          <button
            onClick={() => setShowMergeModal(true)}
            disabled={rejectMutation.isPending}
            className="flex-1 px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Review & Merge
          </button>

          <button
            onClick={handleReject}
            disabled={rejectMutation.isPending}
            className="flex-1 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Not Same Person
          </button>
        </div>
      </div>

      {showMergeModal && (
        <MergeDossierModal
          isOpen={showMergeModal}
          onClose={() => setShowMergeModal(false)}
          group={group}
          programId={programId}
          onSuccess={() => {
            setShowMergeModal(false)
            onSuccess()
          }}
        />
      )}
    </>
  )
}
