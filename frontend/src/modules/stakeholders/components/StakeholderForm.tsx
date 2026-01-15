import { useState, FormEvent } from 'react'
import { CreateStakeholderRequest } from '@/services/api'

interface StakeholderFormProps {
  mode: 'create' | 'edit'
  initialData?: Partial<CreateStakeholderRequest> & { stakeholder_id?: string }
  onSubmit: (data: CreateStakeholderRequest) => Promise<void>
  onCancel: () => void
  isSubmitting: boolean
}

export function StakeholderForm({ mode, initialData, onSubmit, onCancel, isSubmitting }: StakeholderFormProps) {
  const [personName, setPersonName] = useState(initialData?.person_name || '')
  const [email, setEmail] = useState(initialData?.email || '')
  const [role, setRole] = useState(initialData?.role || '')
  const [organization, setOrganization] = useState(initialData?.organization || '')
  const [stakeholderType, setStakeholderType] = useState(initialData?.stakeholder_type || 'internal')
  const [engagementLevel, setEngagementLevel] = useState(initialData?.engagement_level || '')
  const [department, setDepartment] = useState(initialData?.department || '')
  const [notes, setNotes] = useState(initialData?.notes || '')

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()

    // Validation
    if (!personName.trim()) {
      alert('Person name is required')
      return
    }

    if (!stakeholderType) {
      alert('Stakeholder type is required')
      return
    }

    // Email validation
    if (email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      alert('Please enter a valid email address')
      return
    }

    await onSubmit({
      person_name: personName.trim(),
      stakeholder_type: stakeholderType,
      is_internal: stakeholderType === 'internal',
      email: email.trim() || undefined,
      role: role.trim() || undefined,
      organization: organization.trim() || undefined,
      engagement_level: engagementLevel || undefined,
      department: department.trim() || undefined,
      notes: notes.trim() || undefined,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="personName" className="block text-sm font-medium text-gray-700 mb-1">
          Person Name *
        </label>
        <input
          type="text"
          id="personName"
          value={personName}
          onChange={(e) => setPersonName(e.target.value)}
          className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          required
          disabled={isSubmitting}
          placeholder="John Doe"
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label htmlFor="email" className="block text-sm font-medium text-gray-700 mb-1">
            Email
          </label>
          <input
            type="email"
            id="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            disabled={isSubmitting}
            placeholder="john.doe@example.com"
          />
        </div>

        <div>
          <label htmlFor="role" className="block text-sm font-medium text-gray-700 mb-1">
            Role/Title
          </label>
          <input
            type="text"
            id="role"
            value={role}
            onChange={(e) => setRole(e.target.value)}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            disabled={isSubmitting}
            placeholder="Project Manager"
          />
        </div>
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label htmlFor="stakeholderType" className="block text-sm font-medium text-gray-700 mb-1">
            Stakeholder Type *
          </label>
          <select
            id="stakeholderType"
            value={stakeholderType}
            onChange={(e) => setStakeholderType(e.target.value)}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            required
            disabled={isSubmitting}
          >
            <option value="internal">Internal</option>
            <option value="external">External</option>
            <option value="vendor">Vendor</option>
            <option value="partner">Partner</option>
            <option value="customer">Customer</option>
          </select>
        </div>

        <div>
          <label htmlFor="engagementLevel" className="block text-sm font-medium text-gray-700 mb-1">
            Engagement Level
          </label>
          <select
            id="engagementLevel"
            value={engagementLevel}
            onChange={(e) => setEngagementLevel(e.target.value)}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            disabled={isSubmitting}
          >
            <option value="">Select level</option>
            <option value="key">Key</option>
            <option value="primary">Primary</option>
            <option value="secondary">Secondary</option>
            <option value="observer">Observer</option>
          </select>
        </div>
      </div>

      <div>
        <label htmlFor={stakeholderType === 'internal' ? "department" : "organization"} className="block text-sm font-medium text-gray-700 mb-1">
          {stakeholderType === 'internal' ? "Department" : "Organization"}
        </label>
        <input
          type="text"
          id={stakeholderType === 'internal' ? "department" : "organization"}
          value={stakeholderType === 'internal' ? department : organization}
          onChange={(e) => stakeholderType === 'internal' ? setDepartment(e.target.value) : setOrganization(e.target.value)}
          className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          disabled={isSubmitting}
          placeholder={stakeholderType === 'internal' ? "Engineering" : "Acme Corp"}
        />
      </div>

      <div>
        <label htmlFor="notes" className="block text-sm font-medium text-gray-700 mb-1">
          Notes
        </label>
        <textarea
          id="notes"
          value={notes}
          onChange={(e) => setNotes(e.target.value)}
          rows={3}
          className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          disabled={isSubmitting}
          placeholder="Additional context or information..."
        />
      </div>

      <div className="flex justify-end gap-3 pt-4">
        <button
          type="button"
          onClick={onCancel}
          disabled={isSubmitting}
          className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isSubmitting}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {isSubmitting ? 'Saving...' : mode === 'create' ? 'Create Stakeholder' : 'Update Stakeholder'}
        </button>
      </div>
    </form>
  )
}
