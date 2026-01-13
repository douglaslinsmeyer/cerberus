import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeftIcon, CheckCircleIcon, XCircleIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline'
import { useInvoice, useApproveInvoice, useRejectInvoice } from '../hooks/useFinancial'
import { VarianceIndicator } from '../components/VarianceIndicator'
import { format } from 'date-fns'

export function InvoiceDetailPage() {
  const { programId, invoiceId } = useParams<{ programId: string; invoiceId: string }>()
  const { data: invoice, isLoading } = useInvoice(programId || '', invoiceId || '')
  const approveMutation = useApproveInvoice(programId || '')
  const rejectMutation = useRejectInvoice(programId || '')

  const [showRejectDialog, setShowRejectDialog] = useState(false)
  const [rejectReason, setRejectReason] = useState('')

  const handleApprove = async () => {
    if (confirm('Are you sure you want to approve this invoice?')) {
      await approveMutation.mutateAsync(invoiceId || '')
    }
  }

  const handleReject = async () => {
    if (rejectReason.trim()) {
      await rejectMutation.mutateAsync({ invoiceId: invoiceId || '', reason: rejectReason })
      setShowRejectDialog(false)
      setRejectReason('')
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  if (!invoice) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-2">Invoice Not Found</h2>
          <Link to={`/programs/${programId}/financial/invoices`} className="text-blue-600 hover:text-blue-700">
            Back to Invoices
          </Link>
        </div>
      </div>
    )
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

  const lineItemsWithVariances = invoice.line_items?.filter((item) => item.has_variance) || []
  const totalVarianceAmount = lineItemsWithVariances.reduce(
    (sum, item) => sum + (item.rate_variance_amount?.Valid ? item.rate_variance_amount.Float64 : 0),
    0
  )

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Link
                to={`/programs/${programId}/financial/invoices`}
                className="text-gray-400 hover:text-gray-600"
              >
                <ArrowLeftIcon className="h-6 w-6" />
              </Link>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">
                  {invoice.invoice_number?.Valid ? `Invoice ${invoice.invoice_number.String}` : 'Invoice Details'}
                </h1>
                <p className="mt-1 text-sm text-gray-500">{invoice.vendor_name}</p>
              </div>
            </div>

            <div className="flex items-center space-x-2">
              {/* Show approve/reject buttons when validated (AI approved, waiting for human approval) */}
              {invoice.processing_status === 'validated' && (
                <>
                  <button
                    onClick={handleApprove}
                    disabled={approveMutation.isPending}
                    className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-green-600 hover:bg-green-700 disabled:opacity-50"
                    title="Approve this invoice for payment"
                  >
                    <CheckCircleIcon className="h-4 w-4 mr-2" />
                    {approveMutation.isPending ? 'Approving...' : 'Approve Invoice'}
                  </button>
                  <button
                    onClick={() => setShowRejectDialog(true)}
                    disabled={rejectMutation.isPending}
                    className="inline-flex items-center px-4 py-2 border border-red-300 rounded-md shadow-sm text-sm font-medium text-red-700 bg-white hover:bg-red-50 disabled:opacity-50"
                    title="Reject this invoice"
                  >
                    <XCircleIcon className="h-4 w-4 mr-2" />
                    Reject Invoice
                  </button>
                </>
              )}

              {/* Show status for approved/rejected invoices */}
              {invoice.processing_status === 'approved' && (
                <div className="inline-flex items-center px-4 py-2 rounded-md bg-green-50 border border-green-200">
                  <CheckCircleIcon className="h-5 w-5 text-green-600 mr-2" />
                  <span className="text-sm font-medium text-green-800">Invoice Approved</span>
                </div>
              )}

              {invoice.processing_status === 'rejected' && (
                <div className="inline-flex items-center px-4 py-2 rounded-md bg-red-50 border border-red-200">
                  <XCircleIcon className="h-5 w-5 text-red-600 mr-2" />
                  <span className="text-sm font-medium text-red-800">Invoice Rejected</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Help Text for Validated Status */}
        {invoice.processing_status === 'validated' && (
          <div className="mb-6 bg-orange-50 border border-orange-200 rounded-lg p-4">
            <div className="flex">
              <div className="flex-shrink-0">
                <ExclamationTriangleIcon className="h-5 w-5 text-orange-600" />
              </div>
              <div className="ml-3">
                <h3 className="text-sm font-medium text-orange-800">Action Required: Approve or Reject</h3>
                <p className="mt-1 text-sm text-orange-700">
                  AI has successfully validated this invoice and extracted all line items.
                  Review the details below and approve for payment or reject if there are issues.
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Workflow Status Indicator */}
        <div className="mb-6 bg-white rounded-lg shadow p-4">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">Workflow Progress</h3>
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="flex items-center">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-bold ${
                  invoice.processing_status === 'pending' || invoice.processing_status === 'processing' || invoice.processing_status === 'validated' || invoice.processing_status === 'approved' ? 'bg-orange-500' : 'bg-gray-300'
                }`}>
                  1
                </div>
                <div className="ml-3">
                  <p className="text-xs font-medium text-gray-900">Validated</p>
                  <p className="text-xs text-gray-500">AI analyzed</p>
                </div>
              </div>

              <div className={`flex-1 h-1 mx-2 ${
                invoice.processing_status === 'approved' ? 'bg-green-500' : 'bg-gray-200'
              }`}></div>

              <div className="flex items-center">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-bold ${
                  invoice.processing_status === 'approved' ? 'bg-green-500' : 'bg-gray-300'
                }`}>
                  2
                </div>
                <div className="ml-3">
                  <p className="text-xs font-medium text-gray-900">Approved</p>
                  <p className="text-xs text-gray-500">Manager review</p>
                </div>
              </div>

              <div className={`flex-1 h-1 mx-2 ${
                invoice.payment_status === 'paid' ? 'bg-green-500' : 'bg-gray-200'
              }`}></div>

              <div className="flex items-center">
                <div className={`w-8 h-8 rounded-full flex items-center justify-center text-white text-sm font-bold ${
                  invoice.payment_status === 'paid' ? 'bg-green-500' : 'bg-gray-300'
                }`}>
                  3
                </div>
                <div className="ml-3">
                  <p className="text-xs font-medium text-gray-900">Paid</p>
                  <p className="text-xs text-gray-500">Payment sent</p>
                </div>
              </div>
            </div>
          </div>
        </div>

        {/* Invoice Summary */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8 mb-8">
          <div className="lg:col-span-2">
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Invoice Information</h3>

              <dl className="grid grid-cols-2 gap-4">
                <div>
                  <dt className="text-sm font-medium text-gray-500">Invoice Date</dt>
                  <dd className="mt-1 text-sm text-gray-900">{formatDate(invoice.invoice_date)}</dd>
                </div>
                {invoice.due_date?.Valid && (
                  <div>
                    <dt className="text-sm font-medium text-gray-500">Due Date</dt>
                    <dd className="mt-1 text-sm text-gray-900">{formatDate(invoice.due_date.Time)}</dd>
                  </div>
                )}
                {invoice.period_start_date?.Valid && invoice.period_end_date?.Valid && (
                  <>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Period Start</dt>
                      <dd className="mt-1 text-sm text-gray-900">{formatDate(invoice.period_start_date.Time)}</dd>
                    </div>
                    <div>
                      <dt className="text-sm font-medium text-gray-500">Period End</dt>
                      <dd className="mt-1 text-sm text-gray-900">{formatDate(invoice.period_end_date.Time)}</dd>
                    </div>
                  </>
                )}
                {invoice.subtotal_amount?.Valid && (
                  <div>
                    <dt className="text-sm font-medium text-gray-500">Subtotal</dt>
                    <dd className="mt-1 text-sm text-gray-900">{formatCurrency(invoice.subtotal_amount.Float64)}</dd>
                  </div>
                )}
                {invoice.tax_amount?.Valid && (
                  <div>
                    <dt className="text-sm font-medium text-gray-500">Tax</dt>
                    <dd className="mt-1 text-sm text-gray-900">{formatCurrency(invoice.tax_amount.Float64)}</dd>
                  </div>
                )}
              </dl>
            </div>
          </div>

          <div>
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-lg font-semibold text-gray-900 mb-4">Total Amount</h3>
              <p className="text-3xl font-bold text-gray-900">{formatCurrency(invoice.total_amount)}</p>

              {/* Status Badges with Tooltips */}
              <div className="mt-4 space-y-2">
                <div>
                  <p className="text-xs text-gray-500 mb-1">Approval Status:</p>
                  <span
                    className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border ${
                      invoice.processing_status === 'validated' ? 'bg-orange-100 text-orange-800 border-orange-200' :
                      invoice.processing_status === 'approved' ? 'bg-green-100 text-green-800 border-green-200' :
                      invoice.processing_status === 'rejected' ? 'bg-red-100 text-red-800 border-red-200' :
                      'bg-gray-100 text-gray-700 border-gray-300'
                    }`}
                    title={
                      invoice.processing_status === 'validated' ? 'AI has validated - awaiting your approval' :
                      invoice.processing_status === 'approved' ? 'Manager has approved this invoice' :
                      invoice.processing_status === 'rejected' ? 'Invoice has been rejected' :
                      'Processing'
                    }
                  >
                    📋 {invoice.processing_status === 'validated' ? 'Validated (Ready to Approve)' : invoice.processing_status}
                  </span>
                </div>

                <div>
                  <p className="text-xs text-gray-500 mb-1">Payment Status:</p>
                  <span
                    className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium border ${
                      invoice.payment_status === 'unpaid' ? 'bg-blue-100 text-blue-800 border-blue-200' :
                      invoice.payment_status === 'paid' ? 'bg-green-100 text-green-800 border-green-200' :
                      invoice.payment_status === 'overdue' ? 'bg-red-100 text-red-800 border-red-200' :
                      'bg-gray-100 text-gray-700 border-gray-300'
                    }`}
                    title={
                      invoice.payment_status === 'unpaid' ? 'Invoice has not been paid yet' :
                      invoice.payment_status === 'paid' ? 'Invoice has been fully paid' :
                      invoice.payment_status === 'overdue' ? 'Invoice is past due and unpaid' :
                      'Payment tracking'
                    }
                  >
                    💰 {invoice.payment_status === 'unpaid' ? 'Unpaid' : invoice.payment_status}
                  </span>
                </div>
              </div>

              {lineItemsWithVariances.length > 0 && totalVarianceAmount !== 0 && (
                <div className="mt-4 p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
                  <p className="text-sm text-yellow-800">
                    <ExclamationTriangleIcon className="h-4 w-4 inline mr-1" />
                    Variance: <strong>{formatCurrency(totalVarianceAmount)}</strong>
                  </p>
                </div>
              )}

              {invoice.ai_confidence_score?.Valid && (
                <div className="mt-4">
                  <p className="text-xs text-gray-500">AI Confidence</p>
                  <p className="text-sm font-medium text-gray-700">
                    {(invoice.ai_confidence_score.Float64 * 100).toFixed(0)}%
                  </p>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Variances */}
        {invoice.variances && invoice.variances.length > 0 && (
          <div className="mb-8">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Detected Variances</h3>
            <div className="space-y-4">
              {invoice.variances.map((variance) => (
                <VarianceIndicator key={variance.variance_id} variance={variance} showDetails />
              ))}
            </div>
          </div>
        )}

        {/* Line Items */}
        <div className="bg-white rounded-lg shadow overflow-hidden">
          <div className="px-6 py-4 border-b border-gray-200">
            <h3 className="text-lg font-semibold text-gray-900">Line Items</h3>
          </div>

          {!invoice.line_items || invoice.line_items.length === 0 ? (
            <div className="px-6 py-8 text-center text-gray-500">
              <p>No line items available</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Description
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Quantity
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Rate
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Amount
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                      Variance
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {invoice.line_items.map((item) => (
                    <tr key={item.line_item_id} className={item.has_variance ? 'bg-yellow-50' : ''}>
                      <td className="px-6 py-4 text-sm text-gray-900">
                        <div>
                          {item.description}
                          {item.person_name?.Valid && (
                            <p className="text-xs text-gray-500 mt-1">{item.person_name.String}</p>
                          )}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {item.quantity?.Valid ? item.quantity.Float64.toFixed(2) : '-'}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                        {item.unit_rate?.Valid ? formatCurrency(item.unit_rate.Float64) : '-'}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        {formatCurrency(item.line_amount)}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm">
                        {item.has_variance ? (
                          <VarianceIndicator severity={item.variance_severity?.Valid ? item.variance_severity.String as any : 'medium'} />
                        ) : (
                          <span className="text-gray-400">-</span>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* Reject Dialog */}
      {showRejectDialog && (
        <div className="fixed inset-0 bg-gray-500 bg-opacity-75 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-md w-full p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Reject Invoice</h3>
            <p className="text-sm text-gray-600 mb-4">
              Please provide a reason for rejecting this invoice:
            </p>
            <textarea
              value={rejectReason}
              onChange={(e) => setRejectReason(e.target.value)}
              rows={4}
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
              placeholder="Enter rejection reason..."
            />
            <div className="mt-4 flex items-center justify-end space-x-2">
              <button
                onClick={() => {
                  setShowRejectDialog(false)
                  setRejectReason('')
                }}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                onClick={handleReject}
                disabled={!rejectReason.trim() || rejectMutation.isPending}
                className="px-4 py-2 border border-transparent rounded-md text-sm font-medium text-white bg-red-600 hover:bg-red-700 disabled:opacity-50"
              >
                {rejectMutation.isPending ? 'Rejecting...' : 'Reject Invoice'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
