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
  // Payment status (financial state) - Blue theme
  const paymentStatusColors = {
    unpaid: 'bg-blue-100 text-blue-800 border-blue-200',
    partial: 'bg-blue-100 text-blue-700 border-blue-200',
    paid: 'bg-green-100 text-green-800 border-green-200',
    overdue: 'bg-red-100 text-red-800 border-red-200',
  }

  const paymentStatusLabels = {
    unpaid: 'Unpaid',
    partial: 'Partially Paid',
    paid: 'Paid',
    overdue: 'Overdue',
  }

  const paymentStatusTooltips = {
    unpaid: 'Invoice has not been paid yet',
    partial: 'Invoice has been partially paid',
    paid: 'Invoice has been fully paid',
    overdue: 'Invoice is past due date and unpaid',
  }

  // Processing status (approval workflow) - Purple/Orange theme
  const processingStatusColors = {
    pending: 'bg-gray-100 text-gray-700 border-gray-300',
    processing: 'bg-purple-100 text-purple-800 border-purple-200',
    validated: 'bg-orange-100 text-orange-800 border-orange-200',
    approved: 'bg-green-100 text-green-800 border-green-200',
    rejected: 'bg-red-100 text-red-800 border-red-200',
  }

  const processingStatusLabels = {
    pending: 'Pending',
    processing: 'Processing',
    validated: 'Validated',
    approved: 'Approved',
    rejected: 'Rejected',
  }

  const processingStatusTooltips = {
    pending: 'Invoice uploaded, waiting for AI analysis',
    processing: 'AI is currently analyzing this invoice',
    validated: 'AI has validated this invoice - ready for your approval',
    approved: 'Invoice has been approved by a manager',
    rejected: 'Invoice has been rejected',
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
                  <span>Due: {formatDate(invoice.due_date.Time)}</span>
                </>
              )}
            </div>

            <div className="mt-3 flex items-center space-x-2">
              <span
                className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${processingStatusColors[invoice.processing_status as keyof typeof processingStatusColors]}`}
                title={processingStatusTooltips[invoice.processing_status as keyof typeof processingStatusTooltips]}
              >
                📋 {processingStatusLabels[invoice.processing_status as keyof typeof processingStatusLabels] || invoice.processing_status}
              </span>
              <span
                className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${paymentStatusColors[invoice.payment_status as keyof typeof paymentStatusColors]}`}
                title={paymentStatusTooltips[invoice.payment_status as keyof typeof paymentStatusTooltips]}
              >
                💰 {paymentStatusLabels[invoice.payment_status as keyof typeof paymentStatusLabels] || invoice.payment_status}
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
