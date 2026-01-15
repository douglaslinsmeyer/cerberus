import { Fragment } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import { XMarkIcon } from '@heroicons/react/24/outline'
import { StakeholderForm } from './StakeholderForm'
import { useCreateStakeholder, useUpdateStakeholder } from '../hooks/useStakeholders'
import { Stakeholder, CreateStakeholderRequest } from '@/services/api'

interface StakeholderFormModalProps {
  isOpen: boolean
  onClose: () => void
  mode: 'create' | 'edit'
  programId: string
  stakeholder?: Stakeholder
  initialData?: Partial<CreateStakeholderRequest>
  onSuccess: (createdStakeholder?: Stakeholder) => void
}

export function StakeholderFormModal({ isOpen, onClose, mode, programId, stakeholder, initialData: propInitialData, onSuccess }: StakeholderFormModalProps) {
  const createMutation = useCreateStakeholder(programId)
  const updateMutation = useUpdateStakeholder(programId)

  const handleSubmit = async (data: CreateStakeholderRequest) => {
    try {
      if (mode === 'create') {
        const createdStakeholder = await createMutation.mutateAsync(data)
        onSuccess(createdStakeholder)
      } else if (stakeholder) {
        await updateMutation.mutateAsync({
          stakeholderId: stakeholder.stakeholder_id,
          updates: data,
        })
        onSuccess()
      }
      onClose()
    } catch (error) {
      console.error('Failed to save stakeholder:', error)
      alert('Failed to save stakeholder. Please try again.')
    }
  }

  const initialData = mode === 'edit' && stakeholder ? {
    person_name: stakeholder.person_name,
    email: stakeholder.email?.Valid ? stakeholder.email.String : '',
    role: stakeholder.role?.Valid ? stakeholder.role.String : '',
    organization: stakeholder.organization?.Valid ? stakeholder.organization.String : '',
    stakeholder_type: stakeholder.stakeholder_type,
    is_internal: stakeholder.is_internal,
    engagement_level: stakeholder.engagement_level?.Valid ? stakeholder.engagement_level.String : '',
    department: stakeholder.department?.Valid ? stakeholder.department.String : '',
    notes: stakeholder.notes?.Valid ? stakeholder.notes.String : '',
  } : propInitialData

  const isSubmitting = createMutation.isPending || updateMutation.isPending

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
              <Dialog.Panel className="w-full max-w-2xl transform overflow-hidden rounded-2xl bg-white p-6 text-left align-middle shadow-xl transition-all">
                <div className="flex items-center justify-between mb-4">
                  <Dialog.Title as="h3" className="text-lg font-medium leading-6 text-gray-900">
                    {mode === 'create' ? 'Create New Stakeholder' : 'Edit Stakeholder'}
                  </Dialog.Title>
                  <button
                    type="button"
                    className="rounded-md text-gray-400 hover:text-gray-500 focus:outline-none"
                    onClick={onClose}
                  >
                    <XMarkIcon className="h-6 w-6" />
                  </button>
                </div>

                <StakeholderForm
                  mode={mode}
                  initialData={initialData}
                  onSubmit={handleSubmit}
                  onCancel={onClose}
                  isSubmitting={isSubmitting}
                />
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  )
}
