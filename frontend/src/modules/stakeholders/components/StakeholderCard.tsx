import { Link } from 'react-router-dom'
import { UserIcon, EnvelopeIcon, BuildingOfficeIcon, PencilIcon, TrashIcon, EyeIcon } from '@heroicons/react/24/outline'
import { Stakeholder } from '@/services/api'
import { EngagementTracker } from './EngagementTracker'

interface StakeholderCardProps {
  stakeholder: Stakeholder
  programId: string
  onEdit?: (stakeholder: Stakeholder) => void
  onDelete?: (stakeholderId: string) => void
}

export function StakeholderCard({ stakeholder, programId, onEdit, onDelete }: StakeholderCardProps) {
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

  const handleEdit = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (onEdit) {
      onEdit(stakeholder)
    }
  }

  const handleDelete = (e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    if (onDelete && confirm('Are you sure you want to delete this stakeholder?')) {
      onDelete(stakeholder.stakeholder_id)
    }
  }

  return (
    <div className="bg-white rounded-lg shadow hover:shadow-md transition-shadow duration-200 p-6">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-start space-x-3 flex-1">
          <div className={`p-2 rounded-lg border-2 ${typeColors[stakeholder.stakeholder_type]}`}>
            <UserIcon className="h-5 w-5" />
          </div>

          <div className="flex-1 min-w-0">
            <h3 className="text-sm font-semibold text-gray-900">{stakeholder.person_name}</h3>

            <div className="mt-2 flex flex-wrap items-center gap-2">
              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${typeColors[stakeholder.stakeholder_type]}`}>
                {formatLabel(stakeholder.stakeholder_type)}
              </span>

              <EngagementTracker level={stakeholder.engagement_level?.Valid ? stakeholder.engagement_level.String : undefined} />
            </div>
          </div>
        </div>

        <div className="flex items-center space-x-1 ml-2">
          <button
            onClick={handleEdit}
            className="p-1 text-gray-400 hover:text-blue-600 transition-colors"
            title="Edit"
          >
            <PencilIcon className="h-4 w-4" />
          </button>
          <button
            onClick={handleDelete}
            className="p-1 text-gray-400 hover:text-red-600 transition-colors"
            title="Delete"
          >
            <TrashIcon className="h-4 w-4" />
          </button>
          <Link
            to={`/programs/${programId}/stakeholders/${stakeholder.stakeholder_id}`}
            className="p-1 text-gray-400 hover:text-blue-600 transition-colors"
            title="View Details"
          >
            <EyeIcon className="h-4 w-4" />
          </Link>
        </div>
      </div>

      <div className="space-y-2 text-xs text-gray-600">
        {stakeholder.email?.Valid && (
          <div className="flex items-center space-x-2">
            <EnvelopeIcon className="h-4 w-4 text-gray-400" />
            <span className="truncate">{stakeholder.email.String}</span>
          </div>
        )}

        {stakeholder.role?.Valid && (
          <div className="flex items-start space-x-2">
            <UserIcon className="h-4 w-4 text-gray-400 mt-0.5" />
            <span>{stakeholder.role.String}</span>
          </div>
        )}

        {(stakeholder.organization?.Valid || stakeholder.department?.Valid) && (
          <div className="flex items-start space-x-2">
            <BuildingOfficeIcon className="h-4 w-4 text-gray-400 mt-0.5" />
            <span>
              {stakeholder.organization?.Valid && stakeholder.organization.String}
              {stakeholder.organization?.Valid && stakeholder.department?.Valid && ' - '}
              {stakeholder.department?.Valid && stakeholder.department.String}
            </span>
          </div>
        )}
      </div>
    </div>
  )
}
