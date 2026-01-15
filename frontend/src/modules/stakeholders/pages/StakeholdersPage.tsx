import { useState, useMemo } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  ArrowLeftIcon,
  FunnelIcon,
  PlusIcon,
  Squares2X2Icon,
  TableCellsIcon,
  LightBulbIcon,
} from '@heroicons/react/24/outline'
import {
  useStakeholders,
  useStakeholderSuggestions,
  useDeleteStakeholder,
} from '../hooks/useStakeholders'
import { StakeholderCard } from '../components/StakeholderCard'
import { StakeholderTable } from '../components/StakeholderTable'
import { StakeholderFormModal } from '../components/StakeholderFormModal'
import { Stakeholder } from '@/services/api'

type ViewMode = 'card' | 'table'

export function StakeholdersPage() {
  const { programId } = useParams<{ programId: string }>()
  const [viewMode, setViewMode] = useState<ViewMode>('card')
  const [searchQuery, setSearchQuery] = useState('')
  const [typeFilter, setTypeFilter] = useState<string>('')
  const [isInternalFilter, setIsInternalFilter] = useState<boolean | undefined>(undefined)
  const [engagementFilter, setEngagementFilter] = useState<string>('')
  const [showFormModal, setShowFormModal] = useState(false)
  const [editingStakeholder, setEditingStakeholder] = useState<Stakeholder | undefined>(undefined)

  const { data: stakeholders, isLoading } = useStakeholders(programId || '', {
    type: typeFilter || undefined,
    is_internal: isInternalFilter,
    engagement_level: engagementFilter || undefined,
  })

  const { data: suggestions } = useStakeholderSuggestions(programId || '')
  const deleteStakeholderMutation = useDeleteStakeholder(programId || '')

  const safeStakeholders = stakeholders ?? []
  const safeSuggestions = suggestions ?? []

  // Client-side search filtering
  const filteredStakeholders = useMemo(() => {
    if (!searchQuery.trim()) return safeStakeholders

    const query = searchQuery.toLowerCase()
    return safeStakeholders.filter(
      (s) =>
        s.person_name.toLowerCase().includes(query) ||
        (s.email?.Valid && s.email.String.toLowerCase().includes(query)) ||
        (s.role?.Valid && s.role.String.toLowerCase().includes(query)) ||
        (s.organization?.Valid && s.organization.String.toLowerCase().includes(query)) ||
        (s.department?.Valid && s.department.String.toLowerCase().includes(query))
    )
  }, [safeStakeholders, searchQuery])

  // Stats calculations
  const stats = useMemo(() => {
    const byType = {
      internal: safeStakeholders.filter((s) => s.stakeholder_type === 'internal').length,
      external: safeStakeholders.filter((s) => s.stakeholder_type === 'external').length,
      vendor: safeStakeholders.filter((s) => s.stakeholder_type === 'vendor').length,
      partner: safeStakeholders.filter((s) => s.stakeholder_type === 'partner').length,
      customer: safeStakeholders.filter((s) => s.stakeholder_type === 'customer').length,
    }

    const byEngagement = {
      key: safeStakeholders.filter((s) => s.engagement_level?.Valid && s.engagement_level.String === 'key').length,
      primary: safeStakeholders.filter((s) => s.engagement_level?.Valid && s.engagement_level.String === 'primary')
        .length,
      secondary: safeStakeholders.filter((s) => s.engagement_level?.Valid && s.engagement_level.String === 'secondary')
        .length,
      observer: safeStakeholders.filter((s) => s.engagement_level?.Valid && s.engagement_level.String === 'observer')
        .length,
    }

    return { byType, byEngagement }
  }, [safeStakeholders])

  const handleEdit = (stakeholder: Stakeholder) => {
    setEditingStakeholder(stakeholder)
    setShowFormModal(true)
  }

  const handleDelete = async (stakeholderId: string) => {
    await deleteStakeholderMutation.mutateAsync(stakeholderId)
  }

  const handleCloseModal = () => {
    setShowFormModal(false)
    setEditingStakeholder(undefined)
  }

  const handleAddNew = () => {
    setEditingStakeholder(undefined)
    setShowFormModal(true)
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="mb-2">
            <Link
              to={`/programs/${programId}`}
              className="inline-flex items-center text-sm text-gray-600 hover:text-gray-900"
            >
              <ArrowLeftIcon className="h-4 w-4 mr-1" />
              Back to Program
            </Link>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Stakeholders</h1>
              <p className="mt-1 text-sm text-gray-500">Manage program stakeholders and engagement</p>
            </div>
            <button
              onClick={handleAddNew}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
            >
              <PlusIcon className="h-5 w-5 mr-2" />
              Add Stakeholder
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* AI Suggestions Banner */}
        {safeSuggestions.length > 0 && (
          <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
            <div className="flex items-start">
              <LightBulbIcon className="h-6 w-6 text-blue-600 mt-0.5 mr-3" />
              <div className="flex-1">
                <h3 className="text-sm font-medium text-blue-900">AI Detected Stakeholders</h3>
                <p className="text-sm text-blue-700 mt-1">
                  We found {safeSuggestions.length} potential {safeSuggestions.length === 1 ? 'stakeholder' : 'stakeholders'}{' '}
                  in your documents.
                </p>
                <Link
                  to={`/programs/${programId}/stakeholders/suggestions`}
                  className="text-sm text-blue-600 hover:text-blue-800 font-medium mt-2 inline-block"
                >
                  Review suggestions
                </Link>
              </div>
            </div>
          </div>
        )}

        {/* Stats Cards */}
        <div className="grid grid-cols-2 md:grid-cols-6 gap-4 mb-6">
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Total</p>
            <p className="text-xl font-semibold text-gray-900">{safeStakeholders.length}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Internal</p>
            <p className="text-xl font-semibold text-blue-600">{stats.byType.internal}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">External</p>
            <p className="text-xl font-semibold text-purple-600">{stats.byType.external}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Vendors</p>
            <p className="text-xl font-semibold text-orange-600">{stats.byType.vendor}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Key</p>
            <p className="text-xl font-semibold text-red-600">{stats.byEngagement.key}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-xs text-gray-500">Primary</p>
            <p className="text-xl font-semibold text-orange-600">{stats.byEngagement.primary}</p>
          </div>
        </div>

        {/* Filters and View Toggle */}
        <div className="bg-white rounded-lg shadow p-4 mb-6">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center space-x-2">
              <FunnelIcon className="h-5 w-5 text-gray-400" />
              <h3 className="text-sm font-medium text-gray-700">Filters</h3>
            </div>
            <div className="flex items-center space-x-2">
              <button
                onClick={() => setViewMode('card')}
                className={`p-2 rounded ${
                  viewMode === 'card' ? 'bg-blue-100 text-blue-600' : 'text-gray-400 hover:text-gray-600'
                }`}
                title="Card View"
              >
                <Squares2X2Icon className="h-5 w-5" />
              </button>
              <button
                onClick={() => setViewMode('table')}
                className={`p-2 rounded ${
                  viewMode === 'table' ? 'bg-blue-100 text-blue-600' : 'text-gray-400 hover:text-gray-600'
                }`}
                title="Table View"
              >
                <TableCellsIcon className="h-5 w-5" />
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <div>
              <label htmlFor="search" className="block text-sm font-medium text-gray-700 mb-1">
                Search
              </label>
              <input
                type="text"
                id="search"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="Name, email, role..."
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              />
            </div>

            <div>
              <label htmlFor="type" className="block text-sm font-medium text-gray-700 mb-1">
                Type
              </label>
              <select
                id="type"
                value={typeFilter}
                onChange={(e) => setTypeFilter(e.target.value)}
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="">All Types</option>
                <option value="internal">Internal</option>
                <option value="external">External</option>
                <option value="vendor">Vendor</option>
                <option value="partner">Partner</option>
                <option value="customer">Customer</option>
              </select>
            </div>

            <div>
              <label htmlFor="internal" className="block text-sm font-medium text-gray-700 mb-1">
                Internal/External
              </label>
              <select
                id="internal"
                value={isInternalFilter === undefined ? '' : isInternalFilter ? 'true' : 'false'}
                onChange={(e) =>
                  setIsInternalFilter(e.target.value === '' ? undefined : e.target.value === 'true')
                }
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="">All</option>
                <option value="true">Internal</option>
                <option value="false">External</option>
              </select>
            </div>

            <div>
              <label htmlFor="engagement" className="block text-sm font-medium text-gray-700 mb-1">
                Engagement
              </label>
              <select
                id="engagement"
                value={engagementFilter}
                onChange={(e) => setEngagementFilter(e.target.value)}
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="">All Levels</option>
                <option value="key">Key</option>
                <option value="primary">Primary</option>
                <option value="secondary">Secondary</option>
                <option value="observer">Observer</option>
              </select>
            </div>
          </div>
        </div>

        {/* Content */}
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          </div>
        ) : filteredStakeholders.length === 0 ? (
          <div className="bg-white rounded-lg shadow p-12 text-center">
            <p className="text-gray-500">No stakeholders found matching your criteria</p>
          </div>
        ) : viewMode === 'card' ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {filteredStakeholders.map((stakeholder) => (
              <StakeholderCard
                key={stakeholder.stakeholder_id}
                stakeholder={stakeholder}
                programId={programId || ''}
                onEdit={handleEdit}
                onDelete={handleDelete}
              />
            ))}
          </div>
        ) : (
          <StakeholderTable
            stakeholders={filteredStakeholders}
            programId={programId || ''}
            onEdit={handleEdit}
            onDelete={handleDelete}
          />
        )}
      </div>

      {/* Modal */}
      <StakeholderFormModal
        isOpen={showFormModal}
        mode={editingStakeholder ? 'edit' : 'create'}
        programId={programId || ''}
        stakeholder={editingStakeholder}
        onClose={handleCloseModal}
        onSuccess={handleCloseModal}
      />
    </div>
  )
}
