import { useMemo } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useStakeholderSuggestions, useGroupedSuggestions } from '../hooks/useStakeholders'
import { SuggestionCard } from '../components/SuggestionCard'
import { GroupedSuggestionCard } from '../components/GroupedSuggestionCard'
import { SparklesIcon, ArrowLeftIcon } from '@heroicons/react/24/outline'

export function StakeholderSuggestionsPage() {
  const { programId } = useParams<{ programId: string }>()

  const { data: suggestions, isLoading: loadingIndividual, error: errorIndividual, refetch: refetchIndividual } = useStakeholderSuggestions(programId!)
  const { data: groupedSuggestions, isLoading: loadingGrouped, error: errorGrouped, refetch: refetchGrouped } = useGroupedSuggestions(programId!)

  const isLoading = loadingIndividual || loadingGrouped
  const error = errorIndividual || errorGrouped

  // Filter out individual suggestions that are already in groups
  const individualSuggestions = useMemo(() => {
    if (!suggestions || !groupedSuggestions) return suggestions || []

    // Get all person_ids that are in groups
    const groupedPersonIds = new Set(
      groupedSuggestions.flatMap(group => group.members.map(m => m.person_id))
    )

    // Return only suggestions not in any group
    return suggestions.filter(s => !groupedPersonIds.has(s.person_id))
  }, [suggestions, groupedSuggestions])

  const refetch = () => {
    refetchIndividual()
    refetchGrouped()
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-md p-4">
        <p className="text-red-800">Failed to load suggestions. Please try again.</p>
        <button
          onClick={() => refetch()}
          className="mt-2 text-sm text-red-600 hover:text-red-700 underline"
        >
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="space-y-6">
          {/* Header */}
          <div>
            <Link
              to={`/programs/${programId}/stakeholders`}
              className="inline-flex items-center text-sm text-gray-500 hover:text-gray-700 mb-2"
            >
              <ArrowLeftIcon className="h-4 w-4 mr-1" />
              Back to Stakeholders
            </Link>

            <div className="flex items-center justify-between">
              <div>
                <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
                  <SparklesIcon className="h-7 w-7 text-blue-600" />
                  AI Detected People
                </h1>
                <p className="mt-1 text-sm text-gray-500">
                  These people were mentioned in your documents. Create stakeholder records or link to existing ones.
                </p>
              </div>
            </div>
          </div>

          {/* Stats */}
          {(groupedSuggestions && groupedSuggestions.length > 0) || (individualSuggestions && individualSuggestions.length > 0) ? (
            <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
              <p className="text-sm text-blue-800">
                {groupedSuggestions && groupedSuggestions.length > 0 && (
                  <>
                    <span className="font-semibold">{groupedSuggestions.length}</span> person group
                    {groupedSuggestions.length !== 1 ? 's' : ''}
                  </>
                )}
                {groupedSuggestions && groupedSuggestions.length > 0 && individualSuggestions && individualSuggestions.length > 0 && (
                  <span> and </span>
                )}
                {individualSuggestions && individualSuggestions.length > 0 && (
                  <>
                    <span className="font-semibold">{individualSuggestions.length}</span> individual suggestion
                    {individualSuggestions.length !== 1 ? 's' : ''}
                  </>
                )}
                {' '}pending review
              </p>
            </div>
          ) : null}

          {/* Suggestions List - Combined View */}
          {(groupedSuggestions && groupedSuggestions.length > 0) || (individualSuggestions && individualSuggestions.length > 0) ? (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Render grouped suggestions first */}
              {groupedSuggestions?.map((group) => (
                <GroupedSuggestionCard
                  key={group.group_id}
                  group={group}
                  programId={programId!}
                  onSuccess={() => refetch()}
                />
              ))}

              {/* Then render individual suggestions */}
              {individualSuggestions?.map((suggestion) => (
                <SuggestionCard
                  key={suggestion.person_id}
                  suggestion={suggestion}
                  programId={programId!}
                  onSuccess={() => refetch()}
                />
              ))}
            </div>
          ) : (
            /* Empty State */
            <div className="text-center py-12 bg-white rounded-lg shadow">
              <SparklesIcon className="mx-auto h-12 w-12 text-gray-400" />
              <h3 className="mt-2 text-sm font-medium text-gray-900">No new people detected</h3>
              <p className="mt-1 text-sm text-gray-500">
                We'll notify you when new people are mentioned in documents.
              </p>
              <div className="mt-6">
                <Link
                  to={`/programs/${programId}/stakeholders`}
                  className="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"
                >
                  View All Stakeholders
                </Link>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
