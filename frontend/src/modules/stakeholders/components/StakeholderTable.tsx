import { PencilIcon, TrashIcon, EyeIcon } from '@heroicons/react/24/outline'
import { useNavigate } from 'react-router-dom'
import { Stakeholder } from '@/services/api'
import { EngagementTracker } from './EngagementTracker'

interface StakeholderTableProps {
  stakeholders: Stakeholder[]
  programId: string
  onEdit?: (stakeholder: Stakeholder) => void
  onDelete?: (stakeholderId: string) => void
}

export function StakeholderTable({ stakeholders, programId, onEdit, onDelete }: StakeholderTableProps) {
  const navigate = useNavigate()

  const typeColors = {
    internal: 'bg-blue-100 text-blue-800',
    external: 'bg-purple-100 text-purple-800',
    vendor: 'bg-orange-100 text-orange-800',
    partner: 'bg-green-100 text-green-800',
    customer: 'bg-teal-100 text-teal-800',
  }

  const formatLabel = (value: string) => {
    return value.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
  }

  const handleEdit = (e: React.MouseEvent, stakeholder: Stakeholder) => {
    e.stopPropagation()
    if (onEdit) {
      onEdit(stakeholder)
    }
  }

  const handleDelete = (e: React.MouseEvent, stakeholderId: string) => {
    e.stopPropagation()
    if (onDelete && confirm('Are you sure you want to delete this stakeholder?')) {
      onDelete(stakeholderId)
    }
  }

  const handleView = (stakeholderId: string) => {
    navigate(`/programs/${programId}/stakeholders/${stakeholderId}`)
  }

  if (stakeholders.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow p-12 text-center">
        <p className="text-gray-500">No stakeholders found</p>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow overflow-hidden">
      <table className="min-w-full divide-y divide-gray-200">
        <thead className="bg-gray-50">
          <tr>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Name
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Type
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Role
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Organization
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Engagement
            </th>
            <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {stakeholders.map((stakeholder) => (
            <tr
              key={stakeholder.stakeholder_id}
              className="hover:bg-gray-50 cursor-pointer"
              onClick={() => handleView(stakeholder.stakeholder_id)}
            >
              <td className="px-6 py-4 whitespace-nowrap">
                <div className="text-sm font-medium text-gray-900">{stakeholder.person_name}</div>
                {stakeholder.email?.Valid && (
                  <div className="text-sm text-gray-500 truncate max-w-xs">{stakeholder.email.String}</div>
                )}
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${typeColors[stakeholder.stakeholder_type]}`}>
                  {formatLabel(stakeholder.stakeholder_type)}
                </span>
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <div className="text-sm text-gray-900">
                  {stakeholder.role?.Valid ? stakeholder.role.String : '-'}
                </div>
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <div className="text-sm text-gray-900">
                  {stakeholder.organization?.Valid ? stakeholder.organization.String : '-'}
                </div>
                {stakeholder.department?.Valid && (
                  <div className="text-sm text-gray-500">{stakeholder.department.String}</div>
                )}
              </td>
              <td className="px-6 py-4 whitespace-nowrap">
                <EngagementTracker level={stakeholder.engagement_level?.Valid ? stakeholder.engagement_level.String : undefined} />
              </td>
              <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div className="flex items-center justify-end space-x-2">
                  <button
                    onClick={(e) => handleEdit(e, stakeholder)}
                    className="text-blue-600 hover:text-blue-900"
                    title="Edit"
                  >
                    <PencilIcon className="h-5 w-5" />
                  </button>
                  <button
                    onClick={(e) => handleDelete(e, stakeholder.stakeholder_id)}
                    className="text-red-600 hover:text-red-900"
                    title="Delete"
                  >
                    <TrashIcon className="h-5 w-5" />
                  </button>
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleView(stakeholder.stakeholder_id)
                    }}
                    className="text-gray-600 hover:text-gray-900"
                    title="View Details"
                  >
                    <EyeIcon className="h-5 w-5" />
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
