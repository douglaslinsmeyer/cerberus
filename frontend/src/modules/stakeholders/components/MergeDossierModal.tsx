import { Fragment, useState } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import { XMarkIcon, ExclamationTriangleIcon, CheckCircleIcon, ChevronDownIcon, ChevronUpIcon } from '@heroicons/react/24/outline'
import { GroupedSuggestion } from '@/services/api'
import { useConfirmMergeGroup } from '../hooks/useStakeholders'
import { ContextTimelineView } from './ContextTimelineView'

interface MergeDossierModalProps {
  isOpen: boolean
  onClose: () => void
  group: GroupedSuggestion
  programId: string
  onSuccess: () => void
}

export function MergeDossierModal({ isOpen, onClose, group, programId, onSuccess }: MergeDossierModalProps) {
  const confirmMutation = useConfirmMergeGroup(programId)

  // Get all unique name variants
  const nameVariants = Array.from(new Set(group.members.map((m) => m.person_name)))

  // State for selections
  const [selectedName, setSelectedName] = useState(group.suggested_name)
  const [selectedRole, setSelectedRole] = useState<string | undefined>(
    group.role_options && group.role_options.length > 0 ? group.role_options[0].value : undefined
  )
  const [selectedOrg, setSelectedOrg] = useState<string | undefined>(
    group.org_options && group.org_options.length > 0 ? group.org_options[0].value : undefined
  )
  const [createStakeholder, setCreateStakeholder] = useState(true)
  const [showTimeline, setShowTimeline] = useState(false)

  const handleConfirm = async () => {
    try {
      await confirmMutation.mutateAsync({
        groupId: group.group_id,
        request: {
          selected_name: selectedName,
          selected_role: selectedRole,
          selected_organization: selectedOrg,
          create_stakeholder: createStakeholder,
        },
      })
      onSuccess()
    } catch (error) {
      console.error('Failed to confirm merge:', error)
      alert('Failed to confirm merge. Please try again.')
    }
  }

  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black bg-opacity-25" />
        </Transition.Child>

        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel className="w-full max-w-3xl transform overflow-hidden rounded-2xl bg-white p-6 text-left align-middle shadow-xl transition-all">
                <div className="flex items-center justify-between mb-4">
                  <Dialog.Title as="h3" className="text-lg font-medium leading-6 text-gray-900">
                    Confirm Person Identity
                  </Dialog.Title>
                  <button
                    type="button"
                    className="rounded-md text-gray-400 hover:text-gray-500 focus:outline-none"
                    onClick={onClose}
                  >
                    <XMarkIcon className="h-6 w-6" />
                  </button>
                </div>

                <p className="text-sm text-gray-600 mb-6">
                  You're about to merge {group.total_persons} person mentions into one stakeholder record.
                </p>

                <div className="space-y-6">
                  {/* Name Selection */}
                  <div>
                    <label className="block text-sm font-medium text-gray-700 mb-2">
                      Name {nameVariants.length === 1 ? <CheckCircleIcon className="inline h-4 w-4 text-green-600" /> : ''}
                    </label>
                    <div className="space-y-2">
                      {nameVariants.map((name) => (
                        <label
                          key={name}
                          className="flex items-start p-3 border rounded-lg hover:bg-gray-50 cursor-pointer"
                        >
                          <input
                            type="radio"
                            name="name"
                            value={name}
                            checked={selectedName === name}
                            onChange={(e) => setSelectedName(e.target.value)}
                            className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
                          />
                          <div className="ml-3 flex-1">
                            <p className="text-sm font-medium text-gray-900">{name}</p>
                            {name === group.suggested_name && (
                              <span className="text-xs text-gray-500">(most common)</span>
                            )}
                          </div>
                        </label>
                      ))}
                    </div>
                  </div>

                  {/* Role Selection */}
                  {group.role_options && group.role_options.length > 0 && (
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Role {group.has_role_conflicts && <ExclamationTriangleIcon className="inline h-4 w-4 text-yellow-600" />} Conflict detected
                      </label>
                      <div className="space-y-2">
                        {group.role_options.map((option) => (
                          <label
                            key={option.value}
                            className="flex items-start p-3 border rounded-lg hover:bg-gray-50 cursor-pointer"
                          >
                            <input
                              type="radio"
                              name="role"
                              value={option.value}
                              checked={selectedRole === option.value}
                              onChange={(e) => setSelectedRole(e.target.value)}
                              className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
                            />
                            <div className="ml-3 flex-1">
                              <p className="text-sm font-medium text-gray-900">{option.value}</p>
                              <p className="text-xs text-gray-500">
                                {option.count} mention{option.count !== 1 ? 's' : ''} •{' '}
                                {Math.round(option.confidence * 100)}% confidence
                              </p>
                            </div>
                          </label>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Organization Selection */}
                  {group.org_options && group.org_options.length > 0 && (
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Organization {group.has_org_conflicts && <ExclamationTriangleIcon className="inline h-4 w-4 text-yellow-600" />} Conflict detected
                      </label>
                      <div className="space-y-2">
                        {group.org_options.map((option) => (
                          <label
                            key={option.value}
                            className="flex items-start p-3 border rounded-lg hover:bg-gray-50 cursor-pointer"
                          >
                            <input
                              type="radio"
                              name="organization"
                              value={option.value}
                              checked={selectedOrg === option.value}
                              onChange={(e) => setSelectedOrg(e.target.value)}
                              className="mt-1 h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300"
                            />
                            <div className="ml-3 flex-1">
                              <p className="text-sm font-medium text-gray-900">{option.value}</p>
                              <p className="text-xs text-gray-500">
                                {option.count} mention{option.count !== 1 ? 's' : ''} •{' '}
                                {Math.round(option.confidence * 100)}% confidence
                              </p>
                            </div>
                          </label>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Timeline Preview */}
                  {group.all_contexts.length > 0 && (
                    <div className="border-t pt-4">
                      <button
                        onClick={() => setShowTimeline(!showTimeline)}
                        className="flex items-center text-sm font-medium text-gray-700 hover:text-gray-900 mb-3"
                      >
                        {showTimeline ? (
                          <ChevronUpIcon className="h-4 w-4 mr-1" />
                        ) : (
                          <ChevronDownIcon className="h-4 w-4 mr-1" />
                        )}
                        {showTimeline ? 'Hide' : 'Show'} Context Timeline ({group.all_contexts.length} mentions)
                      </button>

                      {showTimeline && (
                        <div className="max-h-60 overflow-y-auto">
                          <ContextTimelineView contexts={group.all_contexts} />
                        </div>
                      )}
                    </div>
                  )}

                  {/* Create Stakeholder Checkbox */}
                  <div className="flex items-center pt-4 border-t">
                    <input
                      type="checkbox"
                      id="create-stakeholder"
                      checked={createStakeholder}
                      onChange={(e) => setCreateStakeholder(e.target.checked)}
                      className="h-4 w-4 text-blue-600 focus:ring-blue-500 border-gray-300 rounded"
                    />
                    <label htmlFor="create-stakeholder" className="ml-2 block text-sm text-gray-700">
                      Create stakeholder record after merging
                    </label>
                  </div>
                </div>

                {/* Action Buttons */}
                <div className="flex justify-end gap-3 mt-6 pt-4 border-t">
                  <button
                    type="button"
                    onClick={onClose}
                    disabled={confirmMutation.isPending}
                    className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md shadow-sm hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    onClick={handleConfirm}
                    disabled={confirmMutation.isPending}
                    className="px-4 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md shadow-sm hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    {confirmMutation.isPending ? 'Confirming...' : 'Confirm Merge'}
                  </button>
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  )
}
