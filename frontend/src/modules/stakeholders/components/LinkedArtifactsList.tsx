import { Link } from 'react-router-dom'
import { DocumentTextIcon } from '@heroicons/react/24/outline'
import { useLinkedArtifacts } from '../hooks/useStakeholders'

interface LinkedArtifactsListProps {
  programId: string
  stakeholderId: string
}

export function LinkedArtifactsList({ programId, stakeholderId }: LinkedArtifactsListProps) {
  const { data: artifacts, isLoading } = useLinkedArtifacts(programId, stakeholderId)

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Linked Artifacts</h3>
        <div className="flex items-center justify-center h-32">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        </div>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Linked Artifacts</h3>

      {artifacts && artifacts.length > 0 ? (
        <div className="space-y-3">
          {artifacts.map((artifact: any) => (
            <Link
              key={artifact.artifact_id}
              to={`/programs/${programId}/artifacts/${artifact.artifact_id}`}
              className="block p-4 border border-gray-200 rounded-lg hover:border-blue-300 hover:shadow-sm transition-all"
            >
              <div className="flex items-start">
                <DocumentTextIcon className="h-5 w-5 text-gray-400 mt-0.5 mr-3" />
                <div className="flex-1">
                  <p className="font-medium text-gray-900">{artifact.filename}</p>
                  <p className="text-sm text-gray-500 mt-1">
                    Uploaded {new Date(artifact.uploaded_at).toLocaleDateString()}
                  </p>
                  {artifact.mention_count && (
                    <p className="text-xs text-blue-600 mt-1">
                      Mentioned {artifact.mention_count} time{artifact.mention_count !== 1 ? 's' : ''}
                    </p>
                  )}
                </div>
              </div>
            </Link>
          ))}
        </div>
      ) : (
        <div className="text-center py-8 text-gray-500">
          <DocumentTextIcon className="mx-auto h-12 w-12 text-gray-300" />
          <p className="mt-2 text-sm">No artifacts mention this stakeholder yet</p>
          <p className="mt-1 text-xs text-gray-400">
            Documents will appear here when AI detects this person's name
          </p>
        </div>
      )}
    </div>
  )
}
