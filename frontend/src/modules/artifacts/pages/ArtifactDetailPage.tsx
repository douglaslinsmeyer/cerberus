import { useParams, Link } from 'react-router-dom'
import { ArrowLeftIcon, ArrowDownTrayIcon, ArrowPathIcon, TrashIcon } from '@heroicons/react/24/outline'
import { useArtifactMetadata, useDeleteArtifact, useReanalyzeArtifact } from '../hooks/useArtifacts'
import { MetadataView } from '../components/MetadataView'
import { artifactsApi } from '@/services/api'

const DEFAULT_PROGRAM_ID = '00000000-0000-0000-0000-000000000001'

export function ArtifactDetailPage() {
  const { artifactId } = useParams<{ artifactId: string }>()
  const { data: artifact, isLoading, error } = useArtifactMetadata(DEFAULT_PROGRAM_ID, artifactId!)
  const deleteMutation = useDeleteArtifact(DEFAULT_PROGRAM_ID)
  const reanalyzeMutation = useReanalyzeArtifact(DEFAULT_PROGRAM_ID)

  const handleDownload = async () => {
    if (!artifactId) return
    try {
      const blob = await artifactsApi.download(DEFAULT_PROGRAM_ID, artifactId)
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = artifact?.filename || 'download'
      a.click()
      window.URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Download failed:', err)
    }
  }

  const handleDelete = async () => {
    if (!artifactId) return
    if (confirm('Are you sure you want to delete this artifact?')) {
      await deleteMutation.mutateAsync(artifactId)
      window.location.href = `/programs/${DEFAULT_PROGRAM_ID}/artifacts`
    }
  }

  const handleReanalyze = async () => {
    if (!artifactId) return
    await reanalyzeMutation.mutateAsync(artifactId)
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (error || !artifact) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Artifact Not Found</h2>
          <Link to={`/programs/${DEFAULT_PROGRAM_ID}/artifacts`} className="text-blue-600 hover:text-blue-700">
            ← Back to Artifacts
          </Link>
        </div>
      </div>
    )
  }

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Link
                to={`/programs/${DEFAULT_PROGRAM_ID}/artifacts`}
                className="text-gray-400 hover:text-gray-600"
              >
                <ArrowLeftIcon className="h-6 w-6" />
              </Link>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">{artifact.filename}</h1>
                <p className="mt-1 text-sm text-gray-500">
                  {artifact.file_type} • {formatFileSize(artifact.file_size_bytes)}
                </p>
              </div>
            </div>

            <div className="flex items-center space-x-2">
              <button
                onClick={handleDownload}
                className="inline-flex items-center px-3 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
              >
                <ArrowDownTrayIcon className="h-4 w-4 mr-2" />
                Download
              </button>

              <button
                onClick={handleReanalyze}
                disabled={reanalyzeMutation.isPending}
                className="inline-flex items-center px-3 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 disabled:opacity-50"
              >
                <ArrowPathIcon className={`h-4 w-4 mr-2 ${reanalyzeMutation.isPending ? 'animate-spin' : ''}`} />
                Reanalyze
              </button>

              <button
                onClick={handleDelete}
                disabled={deleteMutation.isPending}
                className="inline-flex items-center px-3 py-2 border border-red-300 rounded-md shadow-sm text-sm font-medium text-red-700 bg-white hover:bg-red-50 disabled:opacity-50"
              >
                <TrashIcon className="h-4 w-4 mr-2" />
                Delete
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <MetadataView artifact={artifact} />
      </div>
    </div>
  )
}
