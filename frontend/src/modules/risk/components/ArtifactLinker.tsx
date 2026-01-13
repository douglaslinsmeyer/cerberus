import { useState } from 'react'
import { DocumentTextIcon, LinkIcon, XMarkIcon } from '@heroicons/react/24/outline'
import { RiskArtifactLink } from '@/services/api'
import { format } from 'date-fns'

interface ArtifactLinkerProps {
  linkedArtifacts: RiskArtifactLink[]
  onLink: (artifactId: string, linkType: string, description?: string) => void
  onUnlink: (linkId: string) => void
  isLoading?: boolean
}

export function ArtifactLinker({ linkedArtifacts, onLink, onUnlink, isLoading = false }: ArtifactLinkerProps) {
  const [showAddForm, setShowAddForm] = useState(false)
  const [artifactId, setArtifactId] = useState('')
  const [linkType, setLinkType] = useState('evidence')
  const [description, setDescription] = useState('')

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (artifactId.trim()) {
      onLink(artifactId.trim(), linkType, description.trim() || undefined)
      setArtifactId('')
      setLinkType('evidence')
      setDescription('')
      setShowAddForm(false)
    }
  }

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'MMM d, yyyy')
    } catch {
      return dateString
    }
  }

  const linkTypeLabels: Record<string, string> = {
    evidence: 'Evidence',
    mitigation_plan: 'Mitigation Plan',
    analysis: 'Analysis',
    related: 'Related Document',
    historical: 'Historical Reference',
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-lg font-semibold text-gray-900">Linked Artifacts</h3>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="inline-flex items-center px-3 py-1.5 border border-transparent rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
        >
          <LinkIcon className="h-4 w-4 mr-1" />
          Link Artifact
        </button>
      </div>

      {/* Add Form */}
      {showAddForm && (
        <form onSubmit={handleSubmit} className="mb-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
          <div className="space-y-3">
            <div>
              <label htmlFor="artifactId" className="block text-sm font-medium text-gray-700">
                Artifact ID
              </label>
              <input
                type="text"
                id="artifactId"
                value={artifactId}
                onChange={(e) => setArtifactId(e.target.value)}
                placeholder="Enter artifact UUID"
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
                required
              />
            </div>

            <div>
              <label htmlFor="linkType" className="block text-sm font-medium text-gray-700">
                Link Type
              </label>
              <select
                id="linkType"
                value={linkType}
                onChange={(e) => setLinkType(e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="evidence">Evidence</option>
                <option value="mitigation_plan">Mitigation Plan</option>
                <option value="analysis">Analysis</option>
                <option value="related">Related Document</option>
                <option value="historical">Historical Reference</option>
              </select>
            </div>

            <div>
              <label htmlFor="description" className="block text-sm font-medium text-gray-700">
                Description (Optional)
              </label>
              <textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Why is this artifact linked?"
                rows={2}
                className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              />
            </div>

            <div className="flex items-center justify-end space-x-2">
              <button
                type="button"
                onClick={() => {
                  setShowAddForm(false)
                  setArtifactId('')
                  setDescription('')
                }}
                className="px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={isLoading || !artifactId.trim()}
                className="px-3 py-2 border border-transparent rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50"
              >
                {isLoading ? 'Linking...' : 'Link Artifact'}
              </button>
            </div>
          </div>
        </form>
      )}

      {/* Linked Artifacts List */}
      {linkedArtifacts.length === 0 ? (
        <div className="text-center text-gray-500 py-8">
          <DocumentTextIcon className="h-12 w-12 mx-auto text-gray-300 mb-2" />
          <p>No artifacts linked to this risk</p>
        </div>
      ) : (
        <div className="space-y-3">
          {linkedArtifacts.map((link) => (
            <div
              key={link.link_id}
              className="flex items-start justify-between p-3 bg-gray-50 rounded-lg border border-gray-200"
            >
              <div className="flex items-start space-x-3 flex-1">
                <DocumentTextIcon className="h-5 w-5 text-gray-400 flex-shrink-0 mt-0.5" />

                <div className="flex-1 min-w-0">
                  <div className="flex items-center space-x-2">
                    <span className="text-sm font-medium text-gray-900 font-mono">
                      {link.artifact_id}
                    </span>
                    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 text-blue-800">
                      {linkTypeLabels[link.link_type] || link.link_type}
                    </span>
                  </div>

                  {link.description?.Valid && (
                    <p className="text-sm text-gray-600 mt-1">{link.description.String}</p>
                  )}

                  <p className="text-xs text-gray-500 mt-1">
                    Linked {formatDate(link.created_at)}
                  </p>
                </div>
              </div>

              <button
                onClick={() => onUnlink(link.link_id)}
                disabled={isLoading}
                className="ml-4 text-gray-400 hover:text-red-600 disabled:opacity-50"
                title="Unlink artifact"
              >
                <XMarkIcon className="h-5 w-5" />
              </button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
