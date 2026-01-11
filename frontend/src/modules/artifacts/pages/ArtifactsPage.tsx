import { useState } from 'react'
import { MagnifyingGlassIcon } from '@heroicons/react/24/outline'
import { useArtifacts } from '../hooks/useArtifacts'
import { ArtifactUploader } from '../components/ArtifactUploader'
import { ArtifactList } from '../components/ArtifactList'

// Hardcoded for Phase 2 (no auth yet)
const DEFAULT_PROGRAM_ID = '00000000-0000-0000-0000-000000000001'

export function ArtifactsPage() {
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [showUploader, setShowUploader] = useState(false)

  const { data: artifacts = [], isLoading, refetch } = useArtifacts(DEFAULT_PROGRAM_ID, {
    status: statusFilter || undefined,
  })

  const handleUploadSuccess = () => {
    setShowUploader(false)
    refetch()
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Artifacts</h1>
              <p className="mt-1 text-sm text-gray-500">
                AI-powered document analysis and knowledge extraction
              </p>
            </div>

            <button
              onClick={() => setShowUploader(!showUploader)}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              {showUploader ? 'Hide Uploader' : 'Upload Document'}
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Upload Section */}
        {showUploader && (
          <div className="mb-8">
            <ArtifactUploader programId={DEFAULT_PROGRAM_ID} onUploadSuccess={handleUploadSuccess} />
          </div>
        )}

        {/* Filters */}
        <div className="mb-6 flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <label className="text-sm font-medium text-gray-700">Status:</label>
            <select
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
              className="block rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            >
              <option value="">All</option>
              <option value="pending">Pending</option>
              <option value="processing">Processing</option>
              <option value="completed">Completed</option>
              <option value="failed">Failed</option>
            </select>
          </div>

          <div className="flex items-center space-x-2">
            <div className="relative">
              <MagnifyingGlassIcon className="absolute left-3 top-1/2 transform -translate-y-1/2 h-5 w-5 text-gray-400" />
              <input
                type="text"
                placeholder="Search artifacts..."
                className="pl-10 pr-4 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
              />
            </div>
          </div>
        </div>

        {/* Stats */}
        <div className="mb-6 grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-500">Total Artifacts</p>
            <p className="text-2xl font-semibold text-gray-900">{artifacts.length}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-500">Pending</p>
            <p className="text-2xl font-semibold text-yellow-600">
              {artifacts.filter(a => a.processing_status === 'pending').length}
            </p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-500">Processing</p>
            <p className="text-2xl font-semibold text-blue-600">
              {artifacts.filter(a => a.processing_status === 'processing').length}
            </p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-500">Completed</p>
            <p className="text-2xl font-semibold text-green-600">
              {artifacts.filter(a => a.processing_status === 'completed').length}
            </p>
          </div>
        </div>

        {/* Artifacts List */}
        <ArtifactList artifacts={artifacts} programId={DEFAULT_PROGRAM_ID} isLoading={isLoading} />
      </div>
    </div>
  )
}
