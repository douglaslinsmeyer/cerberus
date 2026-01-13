import { Fragment } from 'react'
import { Dialog, Transition } from '@headlessui/react'
import { XMarkIcon } from '@heroicons/react/24/outline'
import { ProgramForm } from './ProgramForm'
import { useCreateProgram, useUpdateProgram } from '../hooks/usePrograms'
import { Program, CreateProgramRequest } from '../../../services/api'

interface ProgramFormModalProps {
  isOpen: boolean
  onClose: () => void
  mode: 'create' | 'edit'
  program?: Program
  onSuccess: () => void
}

export function ProgramFormModal({ isOpen, onClose, mode, program, onSuccess }: ProgramFormModalProps) {
  const createMutation = useCreateProgram()
  const updateMutation = useUpdateProgram()

  const handleSubmit = async (data: CreateProgramRequest) => {
    try {
      if (mode === 'create') {
        await createMutation.mutateAsync(data)
      } else if (program) {
        await updateMutation.mutateAsync({
          programId: program.program_id,
          updates: data,
        })
      }
      onSuccess()
      onClose()
    } catch (error) {
      console.error('Failed to save program:', error)
      alert('Failed to save program. Please try again.')
    }
  }

  const initialData = mode === 'edit' && program ? {
    program_name: program.program_name,
    program_code: program.program_code,
    description: program.description?.Valid ? program.description.String : '',
    start_date: program.start_date?.Valid ? program.start_date.Time.split('T')[0] : '',
    end_date: program.end_date?.Valid ? program.end_date.Time.split('T')[0] : '',
    status: program.status,
  } : undefined

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
                    {mode === 'create' ? 'Create New Program' : 'Edit Program'}
                  </Dialog.Title>
                  <button
                    type="button"
                    className="rounded-md text-gray-400 hover:text-gray-500 focus:outline-none"
                    onClick={onClose}
                  >
                    <XMarkIcon className="h-6 w-6" />
                  </button>
                </div>

                <ProgramForm
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
