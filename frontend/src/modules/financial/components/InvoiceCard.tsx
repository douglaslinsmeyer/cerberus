import { Link } from 'react-router-dom'
import { DocumentTextIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline'
import { Invoice } from '@/services/api'
import { format } from 'date-fns'

interface InvoiceCardProps {
  invoice: Invoice
  programId: string
  showVarianceAlert?: boolean
}

export function InvoiceCard({ invoice, programId, showVarianceAlert = false }: InvoiceCardProps) {
  const paymentStatusColors = {
    pending: 'bg-yellow-100 text-yellow-800',
    approved: 'bg-green-100 text-green-800',
    rejected: 'bg-red-100 text-red-800',
    paid: 'bg-blue-100 text-blue-800',
  }

  const processingStatusColors = {
    pending: 'bg-gray-100 text-gray-800',
    processing: 'bg-blue-100 text-blue-800',
    completed: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: invoice.currency || 'USD',
      minimumFractionDigits: 2,
    }).format(value)
  }

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'MMM d, yyyy')
    } catch {
      return dateString
    }
  }

  return (
    <Link
      to={`/programs/${programId}/financial/invoices/${invoice.invoice_id}`}
      className="block bg-white rounded-lg shadow hover:shadow-md transition-shadow duration-200 p-6"
    >
      <div className="flex items-start justify-between">
        <div className="flex items-start space-x-3 flex-1">
          <DocumentTextIcon className="h-6 w-6 text-gray-400 flex-shrink-0 mt-1" />

          <div className="flex-1 min-w-0">
            <div className="flex items-center space-x-2">
              <h3 className="text-sm font-medium text-gray-900">
                {invoice.invoice_number?.Valid ? invoice.invoice_number.String : 'Invoice'}
              </h3>
              {showVarianceAlert && (
                <ExclamationTriangleIcon className="h-5 w-5 text-red-500" />
              )}
            </div>

            <p className="text-sm text-gray-600 mt-1">{invoice.vendor_name}</p>

            <div className="mt-2 flex items-center space-x-4 text-xs text-gray-500">
              <span>Date: {formatDate(invoice.invoice_date)}</span>
              {invoice.due_date?.Valid && (
                <>
                  <span>•</span>
                  <span>Due: {formatDate(invoice.due_date.String)}</span>
                </>
              )}
            </div>

            <div className="mt-3 flex items-center space-x-2">
              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${paymentStatusColors[invoice.payment_status]}`}>
                {invoice.payment_status}
              </span>
              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${processingStatusColors[invoice.processing_status]}`}>
                {invoice.processing_status}
              </span>
            </div>
          </div>
        </div>

        <div className="text-right ml-4">
          <p className="text-lg font-semibold text-gray-900">{formatCurrency(invoice.total_amount)}</p>
          {invoice.ai_confidence_score?.Valid && (
            <p className="text-xs text-gray-500 mt-1">
              Confidence: {(invoice.ai_confidence_score.Float64 * 100).toFixed(0)}%
            </p>
          )}
        </div>
      </div>
    </Link>
  )
}
