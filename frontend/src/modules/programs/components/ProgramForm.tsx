import { useState, FormEvent } from 'react'
import { CreateProgramRequest } from '../../../services/api'

interface ProgramFormProps {
  mode: 'create' | 'edit'
  initialData?: Partial<CreateProgramRequest>
  onSubmit: (data: CreateProgramRequest) => Promise<void>
  onCancel: () => void
  isSubmitting: boolean
}

export function ProgramForm({ mode, initialData, onSubmit, onCancel, isSubmitting }: ProgramFormProps) {
  const [programName, setProgramName] = useState(initialData?.program_name || '')
  const [programCode, setProgramCode] = useState(initialData?.program_code || '')
  const [description, setDescription] = useState(initialData?.description || '')
  const [startDate, setStartDate] = useState(initialData?.start_date || '')
  const [endDate, setEndDate] = useState(initialData?.end_date || '')
  const [status, setStatus] = useState(initialData?.status || 'active')

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()

    // Validation
    if (!programName.trim() || !programCode.trim()) {
      alert('Program name and code are required')
      return
    }

    if (startDate && endDate && new Date(endDate) < new Date(startDate)) {
      alert('End date must be after start date')
      return
    }

    await onSubmit({
      program_name: programName.trim(),
      program_code: programCode.trim().toUpperCase(),
      description: description.trim() || undefined,
      start_date: startDate || undefined,
      end_date: endDate || undefined,
      status,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label htmlFor="programName" className="block text-sm font-medium text-gray-700 mb-1">
          Program Name *
        </label>
        <input
          type="text"
          id="programName"
          value={programName}
          onChange={(e) => setProgramName(e.target.value)}
          className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          required
          disabled={isSubmitting}
        />
      </div>

      <div>
        <label htmlFor="programCode" className="block text-sm font-medium text-gray-700 mb-1">
          Program Code *
        </label>
        <input
          type="text"
          id="programCode"
          value={programCode}
          onChange={(e) => setProgramCode(e.target.value.toUpperCase())}
          className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          required
          disabled={isSubmitting || mode === 'edit'}
          placeholder="e.g., PROJ-2024"
        />
        {mode === 'edit' && (
          <p className="mt-1 text-xs text-gray-500">Program code cannot be changed</p>
        )}
      </div>

      <div>
        <label htmlFor="description" className="block text-sm font-medium text-gray-700 mb-1">
          Description
        </label>
        <textarea
          id="description"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          rows={3}
          className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          disabled={isSubmitting}
        />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <div>
          <label htmlFor="startDate" className="block text-sm font-medium text-gray-700 mb-1">
            Start Date
          </label>
          <input
            type="date"
            id="startDate"
            value={startDate}
            onChange={(e) => setStartDate(e.target.value)}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            disabled={isSubmitting}
          />
        </div>

        <div>
          <label htmlFor="endDate" className="block text-sm font-medium text-gray-700 mb-1">
            End Date
          </label>
          <input
            type="date"
            id="endDate"
            value={endDate}
            onChange={(e) => setEndDate(e.target.value)}
            className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            disabled={isSubmitting}
          />
        </div>
      </div>

      <div>
        <label htmlFor="status" className="block text-sm font-medium text-gray-700 mb-1">
          Status
        </label>
        <select
          id="status"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          disabled={isSubmitting}
        >
          <option value="active">Active</option>
          <option value="planning">Planning</option>
          <option value="on-hold">On Hold</option>
          <option value="completed">Completed</option>
          <option value="archived">Archived</option>
        </select>
      </div>

      <div className="flex justify-end space-x-3 pt-4">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          disabled={isSubmitting}
        >
          Cancel
        </button>
        <button
          type="submit"
          className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          disabled={isSubmitting}
        >
          {isSubmitting ? 'Saving...' : mode === 'create' ? 'Create Program' : 'Update Program'}
        </button>
      </div>
    </form>
  )
}
