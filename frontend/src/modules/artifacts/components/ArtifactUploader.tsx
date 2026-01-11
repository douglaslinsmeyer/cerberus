import { useCallback } from 'react'
import { useDropzone } from 'react-dropzone'
import { ArrowUpTrayIcon, DocumentIcon } from '@heroicons/react/24/outline'
import { useUploadArtifact } from '../hooks/useArtifacts'

interface ArtifactUploaderProps {
  programId: string
  onUploadSuccess?: () => void
}

export function ArtifactUploader({ programId, onUploadSuccess }: ArtifactUploaderProps) {
  const uploadMutation = useUploadArtifact(programId)

  const onDrop = useCallback((acceptedFiles: File[]) => {
    acceptedFiles.forEach((file) => {
      uploadMutation.mutate(file, {
        onSuccess: () => {
          onUploadSuccess?.()
        },
      })
    })
  }, [uploadMutation, onUploadSuccess])

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    accept: {
      'application/pdf': ['.pdf'],
      'text/plain': ['.txt'],
      'text/markdown': ['.md'],
      'text/csv': ['.csv'],
      'application/json': ['.json'],
    },
    maxSize: 50 * 1024 * 1024, // 50MB
  })

  return (
    <div>
      <div
        {...getRootProps()}
        className={`
          border-2 border-dashed rounded-lg p-12 text-center cursor-pointer
          transition-colors duration-200
          ${isDragActive ? 'border-blue-500 bg-blue-50' : 'border-gray-300 hover:border-gray-400'}
          ${uploadMutation.isPending ? 'opacity-50 cursor-not-allowed' : ''}
        `}
      >
        <input {...getInputProps()} disabled={uploadMutation.isPending} />

        <ArrowUpTrayIcon className="mx-auto h-12 w-12 text-gray-400" />

        <p className="mt-4 text-sm text-gray-600">
          {isDragActive ? (
            <span className="text-blue-600 font-medium">Drop files here...</span>
          ) : (
            <>
              <span className="font-medium text-blue-600 hover:text-blue-500">
                Click to upload
              </span>{' '}
              or drag and drop
            </>
          )}
        </p>

        <p className="mt-1 text-xs text-gray-500">
          PDF or text files up to 50MB
        </p>
      </div>

      {uploadMutation.isPending && (
        <div className="mt-4 bg-blue-50 border border-blue-200 rounded-lg p-4">
          <div className="flex items-center">
            <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-blue-600"></div>
            <span className="ml-3 text-sm text-blue-700">Uploading and extracting content...</span>
          </div>
        </div>
      )}

      {uploadMutation.isError && (
        <div className="mt-4 bg-red-50 border border-red-200 rounded-lg p-4">
          <p className="text-sm text-red-700">
            Error: {uploadMutation.error instanceof Error ? uploadMutation.error.message : 'Upload failed'}
          </p>
        </div>
      )}

      {uploadMutation.isSuccess && (
        <div className="mt-4 bg-green-50 border border-green-200 rounded-lg p-4">
          <div className="flex items-center">
            <DocumentIcon className="h-5 w-5 text-green-600" />
            <span className="ml-3 text-sm text-green-700">
              Upload successful! AI analysis starting...
            </span>
          </div>
        </div>
      )}
    </div>
  )
}
