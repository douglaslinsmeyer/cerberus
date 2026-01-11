import { Artifact } from '@/services/api'
import { ArtifactCard } from './ArtifactCard'

interface ArtifactListProps {
  artifacts: Artifact[]
  programId: string
  isLoading?: boolean
}

export function ArtifactList({ artifacts, programId, isLoading }: ArtifactListProps) {
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="bg-white rounded-lg shadow p-6 animate-pulse">
            <div className="h-4 bg-gray-200 rounded w-3/4 mb-4"></div>
            <div className="h-3 bg-gray-200 rounded w-1/2 mb-2"></div>
            <div className="h-3 bg-gray-200 rounded w-2/3"></div>
          </div>
        ))}
      </div>
    )
  }

  if (artifacts.length === 0) {
    return (
      <div className="text-center py-12">
        <p className="text-gray-500">No artifacts uploaded yet.</p>
        <p className="text-sm text-gray-400 mt-2">Upload a document to get started.</p>
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {artifacts.map((artifact) => (
        <ArtifactCard key={artifact.artifact_id} artifact={artifact} programId={programId} />
      ))}
    </div>
  )
}
