import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import {
  ArrowLeftIcon,
  PencilIcon,
  TrashIcon,
  EnvelopeIcon,
  UserIcon,
  BuildingOfficeIcon,
  DocumentTextIcon,
} from '@heroicons/react/24/outline'
import { useStakeholder, useDeleteStakeholder, useLinkedArtifacts } from '../hooks/useStakeholders'
import { StakeholderFormModal } from '../components/StakeholderFormModal'
import { EngagementTracker } from '../components/EngagementTracker'
import { LinkedArtifactsList } from '../components/LinkedArtifactsList'
import { format } from 'date-fns'

export function StakeholderDetailPage() {
  const { programId, stakeholderId } = useParams<{ programId: string; stakeholderId: string }>()
  const navigate = useNavigate()
  const [showEditModal, setShowEditModal] = useState(false)

  const { data: stakeholder, isLoading } = useStakeholder(programId || '', stakeholderId || '')
  const { data: linkedArtifacts } = useLinkedArtifacts(programId || '', stakeholderId || '')
  const deleteStakeholderMutation = useDeleteStakeholder(programId || '')

  const handleDelete = async () => {
    if (confirm('Are you sure you want to delete this stakeholder?')) {
      await deleteStakeholderMutation.mutateAsync(stakeholderId || '')
      navigate(`/programs/${programId}/stakeholders`)
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (!stakeholder) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Stakeholder Not Found</h2>
          <Link to={`/programs/${programId}/stakeholders`} className="text-blue-600 hover:text-blue-700">
            Back to Stakeholders
          </Link>
        </div>
      </div>
    )
  }

  const typeColors = {
    internal: 'bg-blue-100 text-blue-800 border-blue-300',
    external: 'bg-purple-100 text-purple-800 border-purple-300',
    vendor: 'bg-orange-100 text-orange-800 border-orange-300',
    partner: 'bg-green-100 text-green-800 border-green-300',
    customer: 'bg-teal-100 text-teal-800 border-teal-300',
  }

  const formatLabel = (value: string) => {
    return value.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
  }

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'MMM d, yyyy')
    } catch {
      return dateString
    }
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Link to={`/programs/${programId}/stakeholders`} className="text-gray-400 hover:text-gray-600">
                <ArrowLeftIcon className="h-6 w-6" />
              </Link>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">{stakeholder.person_name}</h1>
                <p className="mt-1 text-sm text-gray-500">Stakeholder ID: {stakeholder.stakeholder_id}</p>
              </div>
            </div>

            <div className="flex items-center space-x-3">
              <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium border-2 ${typeColors[stakeholder.stakeholder_type]}`}>
                {formatLabel(stakeholder.stakeholder_type)}
              </span>

              <EngagementTracker
                level={stakeholder.engagement_level?.Valid ? stakeholder.engagement_level.String : undefined}
                className="text-sm px-3 py-1"
              />

              <button
                onClick={() => setShowEditModal(true)}
                className="inline-flex items-center px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
              >
                <PencilIcon className="h-4 w-4 mr-1" />
                Edit
              </button>

              <button
                onClick={handleDelete}
                className="inline-flex items-center px-3 py-2 border border-red-300 rounded-md text-sm font-medium text-red-700 bg-white hover:bg-red-50"
              >
                <TrashIcon className="h-4 w-4 mr-1" />
                Delete
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {/* Main Content */}
          <div className="lg:col-span-2 space-y-8">
            {/* Contact Information */}
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Contact Information</h3>

              <div className="space-y-4">
                {stakeholder.email?.Valid && (
                  <div className="flex items-start space-x-3">
                    <EnvelopeIcon className="h-5 w-5 text-gray-400 mt-0.5" />
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Email</dt>
                      <dd className="text-sm text-gray-900 mt-1">
                        <a
                          href={`mailto:${stakeholder.email.String}`}
                          className="text-blue-600 hover:text-blue-800"
                        >
                          {stakeholder.email.String}
                        </a>
                      </dd>
                    </div>
                  </div>
                )}

                {stakeholder.role?.Valid && (
                  <div className="flex items-start space-x-3">
                    <UserIcon className="h-5 w-5 text-gray-400 mt-0.5" />
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Role</dt>
                      <dd className="text-sm text-gray-900 mt-1">{stakeholder.role.String}</dd>
                    </div>
                  </div>
                )}

                {(stakeholder.organization?.Valid || stakeholder.department?.Valid) && (
                  <div className="flex items-start space-x-3">
                    <BuildingOfficeIcon className="h-5 w-5 text-gray-400 mt-0.5" />
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Organization</dt>
                      <dd className="text-sm text-gray-900 mt-1">
                        {stakeholder.organization?.Valid && stakeholder.organization.String}
                        {stakeholder.department?.Valid && (
                          <>
                            <br />
                            <span className="text-gray-600">Department: {stakeholder.department.String}</span>
                          </>
                        )}
                      </dd>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Engagement Details */}
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Engagement Details</h3>

              <div className="space-y-4">
                <div>
                  <dt className="text-sm font-medium text-gray-500 mb-1">Engagement Level</dt>
                  <dd className="text-sm text-gray-900">
                    <EngagementTracker
                      level={stakeholder.engagement_level?.Valid ? stakeholder.engagement_level.String : undefined}
                    />
                  </dd>
                </div>

                <div>
                  <dt className="text-sm font-medium text-gray-500 mb-1">Type</dt>
                  <dd className="text-sm text-gray-900">
                    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${typeColors[stakeholder.stakeholder_type]}`}>
                      {formatLabel(stakeholder.stakeholder_type)}
                    </span>
                    <span className="ml-3 text-gray-600">
                      ({stakeholder.is_internal ? 'Internal' : 'External'})
                    </span>
                  </dd>
                </div>

                <div>
                  <dt className="text-sm font-medium text-gray-500 mb-1">Added</dt>
                  <dd className="text-sm text-gray-900">{formatDate(stakeholder.created_at)}</dd>
                </div>

                <div>
                  <dt className="text-sm font-medium text-gray-500 mb-1">Last Updated</dt>
                  <dd className="text-sm text-gray-900">{formatDate(stakeholder.updated_at)}</dd>
                </div>
              </div>
            </div>

            {/* Notes */}
            {stakeholder.notes?.Valid && (
              <div className="bg-white rounded-lg shadow p-6">
                <div className="flex items-center space-x-2 mb-4">
                  <DocumentTextIcon className="h-5 w-5 text-gray-400" />
                  <h3 className="text-lg font-semibold text-gray-900">Notes</h3>
                </div>
                <p className="text-sm text-gray-700 whitespace-pre-wrap">{stakeholder.notes.String}</p>
              </div>
            )}

            {/* Linked Artifacts */}
            <LinkedArtifactsList programId={programId || ''} stakeholderId={stakeholderId || ''} />
          </div>

          {/* Sidebar */}
          <div className="space-y-8">
            {/* Quick Stats */}
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Quick Stats</h3>
              <div className="space-y-3">
                <div className="flex justify-between items-center">
                  <span className="text-sm text-gray-600">Documents</span>
                  <span className="text-sm font-semibold text-gray-900">
                    {linkedArtifacts?.length || 0}
                  </span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm text-gray-600">Linked Risks</span>
                  <span className="text-sm font-semibold text-gray-900">-</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-sm text-gray-600">Last Activity</span>
                  <span className="text-sm font-semibold text-gray-900">
                    {stakeholder.updated_at ? formatDate(stakeholder.updated_at) : '-'}
                  </span>
                </div>
              </div>
            </div>

            {/* Activity Timeline Placeholder */}
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Activity Timeline</h3>
              <div className="text-center text-gray-500 py-4">
                <p className="text-sm">Coming soon</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Edit Modal */}
      <StakeholderFormModal
        isOpen={showEditModal}
        mode="edit"
        programId={programId || ''}
        stakeholder={stakeholder}
        onClose={() => setShowEditModal(false)}
        onSuccess={() => setShowEditModal(false)}
      />
    </div>
  )
}
