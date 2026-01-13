import { useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeftIcon, FunnelIcon } from '@heroicons/react/24/outline'
import { useInvoices } from '../hooks/useFinancial'
import { InvoiceCard } from '../components/InvoiceCard'

export function InvoiceListPage() {
  const { programId } = useParams<{ programId: string }>()
  const [processingStatus, setProcessingStatus] = useState<string>('')
  const [paymentStatus, setPaymentStatus] = useState<string>('')
  const [vendorName, setVendorName] = useState<string>('')

  const { data: invoices = [], isLoading } = useInvoices(programId || '', {
    processing_status: processingStatus || undefined,
    payment_status: paymentStatus || undefined,
    vendor_name: vendorName || undefined,
  })

  const pendingApproval = invoices.filter((i) => i.payment_status === 'pending').length
  const approved = invoices.filter((i) => i.payment_status === 'approved').length
  const rejected = invoices.filter((i) => i.payment_status === 'rejected').length

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="mb-2">
            <Link
              to={`/programs/${programId}/financial`}
              className="inline-flex items-center text-sm text-gray-600 hover:text-gray-900"
            >
              <ArrowLeftIcon className="h-4 w-4 mr-1" />
              Back to Financial Dashboard
            </Link>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Invoices</h1>
              <p className="mt-1 text-sm text-gray-500">
                Manage and review all invoices for this program
              </p>
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-500">Total Invoices</p>
            <p className="text-2xl font-semibold text-gray-900">{invoices.length}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-500">Pending Approval</p>
            <p className="text-2xl font-semibold text-yellow-600">{pendingApproval}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-500">Approved</p>
            <p className="text-2xl font-semibold text-green-600">{approved}</p>
          </div>
          <div className="bg-white rounded-lg shadow p-4">
            <p className="text-sm text-gray-500">Rejected</p>
            <p className="text-2xl font-semibold text-red-600">{rejected}</p>
          </div>
        </div>

        {/* Filters */}
        <div className="bg-white rounded-lg shadow p-4 mb-6">
          <div className="flex items-center space-x-2 mb-3">
            <FunnelIcon className="h-5 w-5 text-gray-400" />
            <h3 className="text-sm font-medium text-gray-700">Filters</h3>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div>
              <label htmlFor="processingStatus" className="block text-sm font-medium text-gray-700 mb-1">
                Processing Status
              </label>
              <select
                id="processingStatus"
                value={processingStatus}
                onChange={(e) => setProcessingStatus(e.target.value)}
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="">All</option>
                <option value="pending">Pending</option>
                <option value="processing">Processing</option>
                <option value="completed">Completed</option>
                <option value="failed">Failed</option>
              </select>
            </div>

            <div>
              <label htmlFor="paymentStatus" className="block text-sm font-medium text-gray-700 mb-1">
                Payment Status
              </label>
              <select
                id="paymentStatus"
                value={paymentStatus}
                onChange={(e) => setPaymentStatus(e.target.value)}
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              >
                <option value="">All</option>
                <option value="pending">Pending</option>
                <option value="approved">Approved</option>
                <option value="rejected">Rejected</option>
                <option value="paid">Paid</option>
              </select>
            </div>

            <div>
              <label htmlFor="vendorName" className="block text-sm font-medium text-gray-700 mb-1">
                Vendor Name
              </label>
              <input
                type="text"
                id="vendorName"
                value={vendorName}
                onChange={(e) => setVendorName(e.target.value)}
                placeholder="Search by vendor..."
                className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              />
            </div>
          </div>
        </div>

        {/* Invoices Grid */}
        {isLoading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
          </div>
        ) : invoices.length === 0 ? (
          <div className="bg-white rounded-lg shadow p-12 text-center">
            <p className="text-gray-500">No invoices found matching your criteria</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {invoices.map((invoice) => (
              <InvoiceCard key={invoice.invoice_id} invoice={invoice} programId={programId || ''} />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
