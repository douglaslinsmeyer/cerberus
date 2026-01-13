import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts'
import { BudgetCategory } from '@/services/api'

interface BudgetChartProps {
  categories: BudgetCategory[]
}

const COLORS = ['#3B82F6', '#10B981', '#F59E0B', '#EF4444', '#8B5CF6', '#EC4899', '#14B8A6', '#F97316']

export function BudgetChart({ categories }: BudgetChartProps) {
  const data = categories.map((cat) => ({
    name: cat.category_name,
    value: cat.actual_spend,
    budgeted: cat.budgeted_amount,
    variance: cat.variance_amount,
  }))

  const totalBudget = categories.reduce((sum, cat) => sum + cat.budgeted_amount, 0)
  const totalSpend = categories.reduce((sum, cat) => sum + cat.actual_spend, 0)
  const totalVariance = totalBudget - totalSpend

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value)
  }

  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-white p-3 shadow-lg rounded-lg border border-gray-200">
          <p className="font-semibold text-gray-900">{payload[0].name}</p>
          <p className="text-sm text-gray-600">
            Spend: <span className="font-semibold">{formatCurrency(payload[0].value)}</span>
          </p>
          <p className="text-sm text-gray-600">
            Budget: <span className="font-semibold">{formatCurrency(payload[0].payload.budgeted)}</span>
          </p>
          <p className={`text-sm ${payload[0].payload.variance >= 0 ? 'text-green-600' : 'text-red-600'}`}>
            Variance: <span className="font-semibold">{formatCurrency(payload[0].payload.variance)}</span>
          </p>
        </div>
      )
    }
    return null
  }

  if (categories.length === 0) {
    return (
      <div className="bg-white rounded-lg shadow p-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">Budget Overview</h3>
        <div className="flex items-center justify-center h-64 text-gray-500">
          <p>No budget data available</p>
        </div>
      </div>
    )
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Budget Overview</h3>

      {/* Summary Stats */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div>
          <p className="text-sm text-gray-500">Total Budget</p>
          <p className="text-xl font-semibold text-gray-900">{formatCurrency(totalBudget)}</p>
        </div>
        <div>
          <p className="text-sm text-gray-500">Total Spend</p>
          <p className="text-xl font-semibold text-blue-600">{formatCurrency(totalSpend)}</p>
        </div>
        <div>
          <p className="text-sm text-gray-500">Remaining</p>
          <p className={`text-xl font-semibold ${totalVariance >= 0 ? 'text-green-600' : 'text-red-600'}`}>
            {formatCurrency(totalVariance)}
          </p>
        </div>
      </div>

      {/* Pie Chart */}
      <ResponsiveContainer width="100%" height={300}>
        <PieChart>
          <Pie
            data={data}
            cx="50%"
            cy="50%"
            labelLine={false}
            label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
            outerRadius={100}
            fill="#8884d8"
            dataKey="value"
          >
            {data.map((_entry, index) => (
              <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
            ))}
          </Pie>
          <Tooltip content={<CustomTooltip />} />
          <Legend />
        </PieChart>
      </ResponsiveContainer>
    </div>
  )
}
