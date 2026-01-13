import { DocumentTextIcon, ClockIcon } from '@heroicons/react/24/outline'
import { Artifact } from '@/services/api'
import { Link } from 'react-router-dom'

interface ArtifactCardProps {
  artifact: Artifact
  programId: string
}

export function ArtifactCard({ artifact, programId }: ArtifactCardProps) {
  const statusColors = {
    pending: 'bg-yellow-100 text-yellow-800',
    processing: 'bg-blue-100 text-blue-800',
    completed: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
    ocr_required: 'bg-orange-100 text-orange-800',
  }

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  const formatDate = (dateString: string) => {
    const date = new Date(dateString)
    return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }

  return (
    <Link
      to={`/programs/${programId}/artifacts/${artifact.artifact_id}`}
      className="block bg-white rounded-lg shadow hover:shadow-md transition-shadow duration-200 p-6"
    >
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start space-x-3 flex-1 min-w-0">
          <DocumentTextIcon className="h-6 w-6 text-gray-400 flex-shrink-0 mt-1" />

          <div className="flex-1 min-w-0">
            <h3 className="text-sm font-medium text-gray-900 truncate">
              {artifact.filename}
            </h3>

            <div className="mt-1 flex items-center space-x-2 text-xs text-gray-500">
              <span>{artifact.file_type || 'Unknown type'}</span>
              <span>•</span>
              <span>{formatFileSize(artifact.file_size_bytes)}</span>
            </div>

            <div className="mt-2 flex items-center space-x-2 text-xs text-gray-500">
              <ClockIcon className="h-4 w-4" />
              <span>{formatDate(artifact.uploaded_at)}</span>
            </div>
          </div>
        </div>

        <div>
          <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColors[artifact.processing_status]}`}>
            {artifact.processing_status}
          </span>
        </div>
      </div>

      {artifact.processing_status === 'processing' && (
        <div className="mt-4">
          <div className="w-full bg-gray-200 rounded-full h-1.5">
            <div className="bg-blue-600 h-1.5 rounded-full animate-pulse" style={{ width: '60%' }}></div>
          </div>
          <p className="mt-1 text-xs text-gray-500">Analyzing with AI...</p>
        </div>
      )}
    </Link>
  )
}
