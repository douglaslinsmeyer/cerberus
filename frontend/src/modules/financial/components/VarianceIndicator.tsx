import { ExclamationTriangleIcon, ExclamationCircleIcon, CheckCircleIcon } from '@heroicons/react/24/outline'
import { FinancialVariance } from '@/services/api'

interface VarianceIndicatorProps {
  variance?: FinancialVariance
  severity?: 'low' | 'medium' | 'high' | 'critical'
  showDetails?: boolean
  className?: string
}

export function VarianceIndicator({ variance, severity, showDetails = false, className = '' }: VarianceIndicatorProps) {
  const effectiveSeverity = variance?.severity || severity || 'low'

  const severityConfig = {
    low: {
      icon: CheckCircleIcon,
      bgColor: 'bg-green-100',
      textColor: 'text-green-800',
      iconColor: 'text-green-600',
      label: 'Low',
    },
    medium: {
      icon: ExclamationCircleIcon,
      bgColor: 'bg-yellow-100',
      textColor: 'text-yellow-800',
      iconColor: 'text-yellow-600',
      label: 'Medium',
    },
    high: {
      icon: ExclamationTriangleIcon,
      bgColor: 'bg-orange-100',
      textColor: 'text-orange-800',
      iconColor: 'text-orange-600',
      label: 'High',
    },
    critical: {
      icon: ExclamationTriangleIcon,
      bgColor: 'bg-red-100',
      textColor: 'text-red-800',
      iconColor: 'text-red-600',
      label: 'Critical',
    },
  }

  const config = severityConfig[effectiveSeverity]
  const Icon = config.icon

  if (!showDetails) {
    return (
      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.bgColor} ${config.textColor} ${className}`}>
        <Icon className={`h-4 w-4 mr-1 ${config.iconColor}`} />
        {config.label}
      </span>
    )
  }

  if (!variance) {
    return (
      <div className={`flex items-center space-x-2 ${className}`}>
        <Icon className={`h-5 w-5 ${config.iconColor}`} />
        <span className={`text-sm font-medium ${config.textColor}`}>{config.label}</span>
      </div>
    )
  }

  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(value)
  }

  return (
    <div className={`${config.bgColor} rounded-lg p-4 ${className}`}>
      <div className="flex items-start space-x-3">
        <Icon className={`h-6 w-6 ${config.iconColor} flex-shrink-0 mt-0.5`} />

        <div className="flex-1">
          <div className="flex items-center justify-between">
            <h4 className={`text-sm font-semibold ${config.textColor}`}>{variance.title}</h4>
            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${config.bgColor} ${config.textColor} border border-current`}>
              {config.label}
            </span>
          </div>

          <p className={`text-sm mt-1 ${config.textColor}`}>{variance.description}</p>

          {(variance.variance_amount?.Valid || variance.variance_percentage?.Valid) && (
            <div className="mt-2 flex items-center space-x-4 text-xs">
              {variance.variance_amount?.Valid && (
                <span className={config.textColor}>
                  Amount: <strong>{formatCurrency(variance.variance_amount.Float64)}</strong>
                </span>
              )}
              {variance.variance_percentage?.Valid && (
                <span className={config.textColor}>
                  Variance: <strong>{variance.variance_percentage.Float64.toFixed(1)}%</strong>
                </span>
              )}
            </div>
          )}

          {variance.ai_confidence_score?.Valid && (
            <p className={`text-xs mt-2 ${config.textColor} opacity-75`}>
              AI Confidence: {(variance.ai_confidence_score.Float64 * 100).toFixed(0)}%
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
