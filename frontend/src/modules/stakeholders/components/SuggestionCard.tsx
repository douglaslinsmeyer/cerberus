import { useState } from 'react'
import { PersonSuggestion } from '@/services/api'
import { UserIcon, DocumentTextIcon, SparklesIcon, CheckCircleIcon } from '@heroicons/react/24/outline'
import { StakeholderFormModal } from './StakeholderFormModal'
import { useLinkPerson, useCreateStakeholder } from '../hooks/useStakeholders'

interface SuggestionCardProps {
  suggestion: PersonSuggestion
  programId: string
  onSuccess: () => void
}

export function SuggestionCard({ suggestion, programId, onSuccess }: SuggestionCardProps) {
  const [showCreateModal, setShowCreateModal] = useState(false)
  const [showArtifacts, setShowArtifacts] = useState(false)

  const linkMutation = useLinkPerson(programId)
  const createMutation = useCreateStakeholder(programId)

  const handleCreateStakeholder = () => {
    setShowCreateModal(true)
  }

  const handleLinkToExisting = async () => {
    if (!suggestion.suggested_stakeholder_id) {
      alert('No similar stakeholder found. Please create a new one.')
      return
    }

    if (confirm('Link this person to the suggested stakeholder?')) {
      try {
        await linkMutation.mutateAsync({
          personId: suggestion.person_id,
          stakeholderId: suggestion.suggested_stakeholder_id,
        })
        onSuccess()
      } catch (error) {
        console.error('Failed to link person:', error)
        alert('Failed to link person. Please try again.')
      }
    }
  }

  const handleModalSuccess = async (createdStakeholder?: any) => {
    // After stakeholder is created, link it to the person suggestion
    if (createdStakeholder) {
      try {
        await linkMutation.mutateAsync({
          personId: suggestion.person_id,
          stakeholderId: createdStakeholder.stakeholder_id,
        })
      } catch (error) {
        console.error('Failed to link person to stakeholder:', error)
        // Still continue - stakeholder was created successfully
      }
    }
    onSuccess()
    setShowCreateModal(false)
  }

  const confidence = suggestion.confidence_score?.Valid
    ? Math.round(suggestion.confidence_score.Float64 * 100)
    : 0

  const similarityScore = suggestion.similarity_score
    ? Math.round(suggestion.similarity_score * 100)
    : 0

  return (
    <>
      <div className="bg-white rounded-lg shadow p-6 hover:shadow-md transition-shadow">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <UserIcon className="h-6 w-6 text-blue-600" />
              <h3 className="text-lg font-semibold text-gray-900">{suggestion.person_name}</h3>
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                <SparklesIcon className="h-3 w-3 mr-1" />
                AI Detected
              </span>
            </div>

            <div className="space-y-1 mb-3">
              {suggestion.person_role?.Valid && (
                <p className="text-sm text-gray-600">
                  <span className="font-medium">Role:</span> {suggestion.person_role.String}
                </p>
              )}
              {suggestion.person_organization?.Valid && (
                <p className="text-sm text-gray-600">
                  <span className="font-medium">Organization:</span> {suggestion.person_organization.String}
                </p>
              )}
              <p className="text-sm text-gray-600">
                <span className="font-medium">Mentioned in:</span> {suggestion.artifact_count} document{suggestion.artifact_count !== 1 ? 's' : ''} ({suggestion.total_mentions} total mentions)
              </p>
              {confidence > 0 && (
                <p className="text-sm text-gray-600">
                  <span className="font-medium">Confidence:</span> {confidence}%
                </p>
              )}
            </div>

            {/* Similar stakeholder indicator */}
            {suggestion.suggested_stakeholder_id && similarityScore > 0 && (
              <div className={`mb-3 p-3 rounded-md border ${
                similarityScore >= 95
                  ? 'bg-green-50 border-green-300'
                  : 'bg-yellow-50 border-yellow-200'
              }`}>
                <p className={`text-sm font-medium ${
                  similarityScore >= 95 ? 'text-green-800' : 'text-yellow-800'
                }`}>
                  <CheckCircleIcon className="h-4 w-4 inline mr-1" />
                  {similarityScore >= 95
                    ? 'This person already exists as a stakeholder!'
                    : `Similar to existing stakeholder (${similarityScore}% match)`
                  }
                </p>
                {similarityScore >= 95 && (
                  <p className="text-xs text-green-700 mt-1">
                    Click "Add to Existing" to link this mention to their record.
                  </p>
                )}
              </div>
            )}

            {/* Artifacts list */}
            {suggestion.artifacts && suggestion.artifacts.length > 0 && (
              <div className="mb-4">
                <button
                  onClick={() => setShowArtifacts(!showArtifacts)}
                  className="text-sm font-medium text-blue-600 hover:text-blue-700 flex items-center"
                >
                  <DocumentTextIcon className="h-4 w-4 mr-1" />
                  {showArtifacts ? 'Hide' : 'Show'} documents ({suggestion.artifacts.length})
                </button>

                {showArtifacts && (
                  <div className="mt-2 space-y-2 pl-5">
                    {suggestion.artifacts.map((artifact) => (
                      <div key={artifact.artifact_id} className="text-sm text-gray-600">
                        <span className="font-medium">{artifact.filename}</span>
                        <span className="text-gray-400 ml-2">({artifact.mention_count} mentions)</span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex gap-2 mt-4 pt-4 border-t border-gray-200">
          {suggestion.suggested_stakeholder_id ? (
            <>
              {/* When similar stakeholder exists, make linking primary action */}
              <button
                onClick={handleLinkToExisting}
                disabled={linkMutation.isPending || createMutation.isPending}
                className="flex-1 px-4 py-2 text-sm font-medium text-white bg-green-600 border border-transparent rounded-md shadow-sm hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {similarityScore >= 95 ? 'Add to Existing' : 'Link to Similar'}
              </button>
              <button
                onClick={handleCreateStakeholder}
                disabled={linkMutation.isPending || createMutation.isPending}
                className="flex-1 px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                Create New Instead
              </button>
            </>
          ) : (
            /* No similar stakeholder - create is primary action */
            <button
              onClick={handleCreateStakeholder}
              disabled={linkMutation.isPending || createMutation.isPending}
              className="flex-1 px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Create Stakeholder
            </button>
          )}
        </div>
      </div>

      {showCreateModal && (
        <StakeholderFormModal
          isOpen={showCreateModal}
          onClose={() => setShowCreateModal(false)}
          mode="create"
          programId={programId}
          initialData={{
            person_name: suggestion.person_name,
            role: suggestion.person_role?.Valid ? suggestion.person_role.String : '',
            organization: suggestion.person_organization?.Valid ? suggestion.person_organization.String : '',
            // Use backend-computed classification based on internal org matching
            stakeholder_type: suggestion.suggested_stakeholder_type || 'external',
          }}
          onSuccess={handleModalSuccess}
        />
      )}
    </>
  )
}
