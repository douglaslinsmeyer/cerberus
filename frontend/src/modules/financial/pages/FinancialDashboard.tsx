import { Link, useParams } from 'react-router-dom'
import { ArrowLeftIcon, CurrencyDollarIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline'
import { useBudgetStatus, useInvoices, useVariances } from '../hooks/useFinancial'
import { BudgetChart } from '../components/BudgetChart'
import { InvoiceCard } from '../components/InvoiceCard'
import { VarianceIndicator } from '../components/VarianceIndicator'

export function FinancialDashboard() {
  const { programId } = useParams<{ programId: string }>()
  const currentYear = new Date().getFullYear()

  const { data: budgetStatus, isLoading: budgetLoading } = useBudgetStatus(programId || '', currentYear)
  const { data: invoices = [], isLoading: invoicesLoading } = useInvoices(programId || '', { limit: 5 })
  const { data: variances = [], isLoading: variancesLoading } = useVariances(programId || '')

  const activeVariances = variances.filter((v) => !v.is_dismissed && !v.resolved_at)
  const criticalVariances = activeVariances.filter((v) => v.severity === 'critical')
  const highVariances = activeVariances.filter((v) => v.severity === 'high')

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value)
  }

  if (budgetLoading || invoicesLoading || variancesLoading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="mb-2">
            <Link
              to={`/programs/${programId}`}
              className="inline-flex items-center text-sm text-gray-600 hover:text-gray-900"
            >
              <ArrowLeftIcon className="h-4 w-4 mr-1" />
              Back to Program Dashboard
            </Link>
          </div>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Financial Overview</h1>
              <p className="mt-1 text-sm text-gray-500">
                Invoice validation, budget tracking, and spend analysis
              </p>
            </div>

            <Link
              to={`/programs/${programId}/financial/invoices`}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
            >
              View All Invoices
            </Link>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Summary Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <CurrencyDollarIcon className="h-8 w-8 text-blue-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Total Budget</p>
                <p className="text-2xl font-semibold text-gray-900">
                  {formatCurrency(budgetStatus?.total_budgeted || 0)}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <CurrencyDollarIcon className="h-8 w-8 text-green-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Actual Spend</p>
                <p className="text-2xl font-semibold text-green-600">
                  {formatCurrency(budgetStatus?.total_actual || 0)}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <CurrencyDollarIcon className="h-8 w-8 text-purple-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Committed</p>
                <p className="text-2xl font-semibold text-purple-600">
                  {formatCurrency(budgetStatus?.total_committed || 0)}
                </p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <ExclamationTriangleIcon className="h-8 w-8 text-red-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Variances</p>
                <p className="text-2xl font-semibold text-red-600">{activeVariances.length}</p>
              </div>
            </div>
          </div>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 mb-8">
          {/* Budget Chart */}
          {budgetStatus && <BudgetChart categories={budgetStatus.categories || []} />}

          {/* Recent Variances */}
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold text-gray-900">Recent Variances</h3>
              <Link
                to={`/programs/${programId}/financial/variances`}
                className="text-sm text-blue-600 hover:text-blue-700"
              >
                View All
              </Link>
            </div>

            {activeVariances.length === 0 ? (
              <div className="text-center text-gray-500 py-8">
                <p>No active variances</p>
              </div>
            ) : (
              <div className="space-y-3">
                {activeVariances.slice(0, 5).map((variance) => (
                  <VarianceIndicator key={variance.variance_id} variance={variance} showDetails />
                ))}
              </div>
            )}

            {criticalVariances.length > 0 && (
              <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
                <p className="text-sm text-red-800">
                  <strong>{criticalVariances.length}</strong> critical variances require immediate attention
                </p>
              </div>
            )}
          </div>
        </div>

        {/* Recent Invoices */}
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-lg font-semibold text-gray-900">Recent Invoices</h3>
            <Link
              to={`/programs/${programId}/financial/invoices`}
              className="text-sm text-blue-600 hover:text-blue-700"
            >
              View All
            </Link>
          </div>

          {invoices.length === 0 ? (
            <div className="text-center text-gray-500 py-8">
              <p>No invoices yet</p>
            </div>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {invoices.map((invoice) => (
                <InvoiceCard key={invoice.invoice_id} invoice={invoice} programId={programId || ''} />
              ))}
            </div>
          )}
        </div>

        {/* Info Box */}
        {(criticalVariances.length > 0 || highVariances.length > 0) && (
          <div className="mt-8 bg-yellow-50 border border-yellow-200 rounded-lg p-6">
            <h3 className="text-lg font-semibold text-yellow-900 mb-2">Action Required</h3>
            <p className="text-sm text-yellow-700">
              There are {criticalVariances.length + highVariances.length} high-priority variances that need
              review. These may indicate billing discrepancies or budget overruns.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
