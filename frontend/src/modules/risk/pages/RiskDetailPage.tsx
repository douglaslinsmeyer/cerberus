import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeftIcon, ShieldCheckIcon, ChatBubbleLeftRightIcon } from '@heroicons/react/24/outline'
import {
  useRisk,
  useMitigations,
  useLinkedArtifacts,
  useConversations,
  useThreadMessages,
  useLinkArtifact,
  useUnlinkArtifact,
  useAddMessage,
  useCreateThread,
  useEnrichments,
} from '../hooks/useRisks'
import { ConversationThread } from '../components/ConversationThread'
import { ArtifactLinker } from '../components/ArtifactLinker'
import { RiskEnrichmentTimeline } from '../components/RiskEnrichmentTimeline'
import { format } from 'date-fns'

export function RiskDetailPage() {
  const { programId, riskId } = useParams<{ programId: string; riskId: string }>()

  const { data: risk, isLoading: riskLoading } = useRisk(programId || '', riskId || '')
  const { data: mitigations = [], isLoading: mitigationsLoading } = useMitigations(programId || '', riskId || '')
  const { data: linkedArtifacts = [], isLoading: artifactsLoading } = useLinkedArtifacts(programId || '', riskId || '')
  const { data: threads = [], isLoading: threadsLoading } = useConversations(programId || '', riskId || '')
  const { data: enrichments = [] } = useEnrichments(programId || '', riskId || '')

  const [selectedThreadId, setSelectedThreadId] = useState<string | null>(null)
  const [showNewThreadDialog, setShowNewThreadDialog] = useState(false)
  const [newThreadTitle, setNewThreadTitle] = useState('')

  const { data: selectedThread } = useThreadMessages(
    programId || '',
    riskId || '',
    selectedThreadId || ''
  )

  const linkArtifactMutation = useLinkArtifact(programId || '', riskId || '')
  const unlinkArtifactMutation = useUnlinkArtifact(programId || '', riskId || '')
  const addMessageMutation = useAddMessage(programId || '', riskId || '', selectedThreadId || '')
  const createThreadMutation = useCreateThread(programId || '', riskId || '')

  const handleLinkArtifact = async (artifactId: string, linkType: string, description?: string) => {
    await linkArtifactMutation.mutateAsync({ artifactId, linkType, description })
  }

  const handleUnlinkArtifact = async (linkId: string) => {
    if (confirm('Are you sure you want to unlink this artifact?')) {
      await unlinkArtifactMutation.mutateAsync(linkId)
    }
  }

  const handleSendMessage = async (message: string) => {
    await addMessageMutation.mutateAsync({ messageText: message, messageFormat: 'markdown' })
  }

  const handleCreateThread = async () => {
    if (newThreadTitle.trim()) {
      const result = await createThreadMutation.mutateAsync({
        title: newThreadTitle.trim(),
        threadType: 'discussion',
      })
      setShowNewThreadDialog(false)
      setNewThreadTitle('')
      if (result.thread_id) {
        setSelectedThreadId(result.thread_id)
      }
    }
  }

  if (riskLoading || mitigationsLoading || artifactsLoading || threadsLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (!risk) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Risk Not Found</h2>
          <Link to={`/programs/${programId}/risks`} className="text-blue-600 hover:text-blue-700">
            Back to Risks
          </Link>
        </div>
      </div>
    )
  }

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'MMM d, yyyy')
    } catch {
      return dateString
    }
  }

  const formatLabel = (value: string) => {
    return value.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
  }

  const severityColors = {
    low: 'bg-green-100 text-green-800 border-green-300',
    medium: 'bg-yellow-100 text-yellow-800 border-yellow-300',
    high: 'bg-orange-100 text-orange-800 border-orange-300',
    critical: 'bg-red-100 text-red-800 border-red-300',
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Link
                to={`/programs/${programId}/risks`}
                className="text-gray-400 hover:text-gray-600"
              >
                <ArrowLeftIcon className="h-6 w-6" />
              </Link>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">{risk.title}</h1>
                <p className="mt-1 text-sm text-gray-500">Risk ID: {risk.risk_id}</p>
              </div>
            </div>

            <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium border-2 ${severityColors[risk.severity]}`}>
              {formatLabel(risk.severity)} Severity
            </span>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Main Content */}
          <div className="lg:col-span-2 space-y-8">
            {/* Risk Details */}
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Risk Details</h3>

              <div className="space-y-4">
                <div>
                  <dt className="text-sm font-medium text-gray-500 mb-1">Description</dt>
                  <dd className="text-sm text-gray-900">{risk.description}</dd>
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <dt className="text-sm font-medium text-gray-500 mb-1">Probability</dt>
                    <dd className="text-sm text-gray-900">{formatLabel(risk.probability)}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-gray-500 mb-1">Impact</dt>
                    <dd className="text-sm text-gray-900">{formatLabel(risk.impact)}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-gray-500 mb-1">Category</dt>
                    <dd className="text-sm text-gray-900">{formatLabel(risk.category)}</dd>
                  </div>
                  <div>
                    <dt className="text-sm font-medium text-gray-500 mb-1">Status</dt>
                    <dd className="text-sm text-gray-900">{formatLabel(risk.status)}</dd>
                  </div>
                  {risk.owner_name?.Valid && (
                    <div>
                      <dt className="text-sm font-medium text-gray-500 mb-1">Owner</dt>
                      <dd className="text-sm text-gray-900">{risk.owner_name.String}</dd>
                    </div>
                  )}
                  <div>
                    <dt className="text-sm font-medium text-gray-500 mb-1">Identified Date</dt>
                    <dd className="text-sm text-gray-900">{formatDate(risk.identified_date)}</dd>
                  </div>
                  {risk.target_resolution_date?.Valid && (
                    <div>
                      <dt className="text-sm font-medium text-gray-500 mb-1">Target Resolution</dt>
                      <dd className="text-sm text-gray-900">{formatDate(risk.target_resolution_date.String)}</dd>
                    </div>
                  )}
                </div>

                {risk.ai_confidence_score?.Valid && (
                  <div>
                    <dt className="text-sm font-medium text-gray-500 mb-1">AI Confidence</dt>
                    <dd className="text-sm text-gray-900">
                      {(risk.ai_confidence_score.Float64 * 100).toFixed(0)}%
                    </dd>
                  </div>
                )}
              </div>
            </div>

            {/* Enrichment Timeline */}
            {enrichments.length > 0 && (
              <RiskEnrichmentTimeline
                riskId={riskId || ''}
                programId={programId || ''}
                enrichments={enrichments}
              />
            )}

            {/* Mitigations */}
            <div className="bg-white rounded-lg shadow p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900">Mitigation Actions</h3>
                <ShieldCheckIcon className="h-6 w-6 text-green-600" />
              </div>

              {mitigations.length === 0 ? (
                <div className="text-center text-gray-500 py-8">
                  <p>No mitigation actions defined yet</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {mitigations.map((mitigation) => (
                    <div key={mitigation.mitigation_id} className="border border-gray-200 rounded-lg p-4">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center space-x-2 mb-2">
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                              {formatLabel(mitigation.strategy)}
                            </span>
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                              {formatLabel(mitigation.status)}
                            </span>
                          </div>
                          <p className="text-sm text-gray-900">{mitigation.action_description}</p>

                          <div className="mt-2 flex items-center space-x-4 text-xs text-gray-500">
                            {mitigation.target_completion_date?.Valid && (
                              <span>Target: {formatDate(mitigation.target_completion_date.String)}</span>
                            )}
                            {mitigation.estimated_cost?.Valid && (
                              <span>Est. Cost: ${mitigation.estimated_cost.Float64.toFixed(0)}</span>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Linked Artifacts */}
            <ArtifactLinker
              linkedArtifacts={linkedArtifacts}
              onLink={handleLinkArtifact}
              onUnlink={handleUnlinkArtifact}
              isLoading={linkArtifactMutation.isPending || unlinkArtifactMutation.isPending}
            />
          </div>

          {/* Sidebar */}
          <div className="space-y-8">
            {/* Conversations */}
            <div className="bg-white rounded-lg shadow p-6">
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900">Discussions</h3>
                <button
                  onClick={() => setShowNewThreadDialog(true)}
                  className="text-sm text-blue-600 hover:text-blue-700"
                >
                  New Thread
                </button>
              </div>

              {threads.length === 0 ? (
                <div className="text-center text-gray-500 py-4">
                  <ChatBubbleLeftRightIcon className="h-8 w-8 mx-auto text-gray-300 mb-2" />
                  <p className="text-sm">No discussions yet</p>
                </div>
              ) : (
                <div className="space-y-2 mb-4">
                  {threads.map((thread) => (
                    <button
                      key={thread.thread_id}
                      onClick={() => setSelectedThreadId(thread.thread_id)}
                      className={`w-full text-left p-3 rounded-lg border transition-colors ${
                        selectedThreadId === thread.thread_id
                          ? 'border-blue-500 bg-blue-50'
                          : 'border-gray-200 hover:bg-gray-50'
                      }`}
                    >
                      <p className="text-sm font-medium text-gray-900">{thread.title}</p>
                      <p className="text-xs text-gray-500 mt-1">
                        {thread.message_count} message{thread.message_count !== 1 ? 's' : ''}
                      </p>
                    </button>
                  ))}
                </div>
              )}

              {selectedThreadId && selectedThread && (
                <ConversationThread
                  messages={selectedThread.messages || []}
                  onSendMessage={handleSendMessage}
                  isLoading={addMessageMutation.isPending}
                />
              )}
            </div>
          </div>
        </div>
      </div>

      {/* New Thread Dialog */}
      {showNewThreadDialog && (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">New Discussion Thread</h3>
            <input
              type="text"
              value={newThreadTitle}
              onChange={(e) => setNewThreadTitle(e.target.value)}
              placeholder="Thread title..."
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            />
            <div className="mt-4 flex items-center justify-end space-x-2">
              <button
                onClick={() => {
                  setShowNewThreadDialog(false)
                  setNewThreadTitle('')
                }}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleCreateThread}
                disabled={!newThreadTitle.trim() || createThreadMutation.isPending}
                className="px-4 py-2 border border-transparent rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50"
              >
                {createThreadMutation.isPending ? 'Creating...' : 'Create Thread'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
