import { ContextMention } from '@/services/api'
import { DocumentTextIcon } from '@heroicons/react/24/outline'

interface ContextTimelineViewProps {
  contexts: ContextMention[]
}

export function ContextTimelineView({ contexts }: ContextTimelineViewProps) {
  if (contexts.length === 0) {
    return (
      <div className="text-center py-6 text-gray-500">
        <p className="text-sm">No context snippets available</p>
      </div>
    )
  }

  // Group contexts by artifact for better organization
  const contextsByArtifact = contexts.reduce((acc, context) => {
    if (!acc[context.artifact_id]) {
      acc[context.artifact_id] = []
    }
    acc[context.artifact_id].push(context)
    return acc
  }, {} as Record<string, ContextMention[]>)

  return (
    <div className="space-y-4">
      <h4 className="text-sm font-medium text-gray-700 mb-3">Timeline of Mentions</h4>

      <div className="space-y-4">
        {Object.values(contextsByArtifact).map((artifactContexts) => {
          const firstContext = artifactContexts[0]
          const totalMentions = artifactContexts.length

          return (
            <div key={firstContext.artifact_id} className="border-l-2 border-blue-200 pl-4">
              <div className="flex items-start gap-2 mb-2">
                <DocumentTextIcon className="h-5 w-5 text-blue-600 mt-0.5" />
                <div className="flex-1">
                  <p className="text-sm font-medium text-gray-900">
                    {firstContext.artifact_name}
                  </p>
                  <p className="text-xs text-gray-500">
                    {new Date(firstContext.uploaded_at).toLocaleDateString('en-US', {
                      year: 'numeric',
                      month: 'short',
                      day: 'numeric',
                    })}
                  </p>
                </div>
                <span className="text-xs text-gray-500">
                  {totalMentions} mention{totalMentions !== 1 ? 's' : ''}
                </span>
              </div>

              {/* Show snippets */}
              <div className="space-y-2">
                {artifactContexts.map((context, idx) => (
                  <div key={idx} className="bg-gray-50 rounded p-3">
                    {context.snippet ? (
                      <>
                        <p className="text-sm text-gray-700 italic">
                          "{context.snippet}"
                        </p>
                        {context.person_name && (
                          <p className="text-xs text-gray-500 mt-1">
                            Mentioned as: <span className="font-medium">{context.person_name}</span>
                          </p>
                        )}
                      </>
                    ) : (
                      <p className="text-sm text-gray-500 italic">
                        (No context available)
                      </p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
