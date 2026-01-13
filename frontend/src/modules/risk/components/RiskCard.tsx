import { Link } from 'react-router-dom'
import { ExclamationTriangleIcon, UserIcon } from '@heroicons/react/24/outline'
import { Risk } from '@/services/api'
import { format } from 'date-fns'

interface RiskCardProps {
  risk: Risk
  programId: string
}

export function RiskCard({ risk, programId }: RiskCardProps) {
  const severityColors = {
    low: 'bg-green-100 text-green-800 border-green-300',
    medium: 'bg-yellow-100 text-yellow-800 border-yellow-300',
    high: 'bg-orange-100 text-orange-800 border-orange-300',
    critical: 'bg-red-100 text-red-800 border-red-300',
  }

  const statusColors = {
    identified: 'bg-gray-100 text-gray-800',
    assessing: 'bg-blue-100 text-blue-800',
    mitigating: 'bg-yellow-100 text-yellow-800',
    monitoring: 'bg-purple-100 text-purple-800',
    closed: 'bg-green-100 text-green-800',
    realized: 'bg-red-100 text-red-800',
  }

  const formatDate = (dateString: string) => {
    try {
      return format(new Date(dateString), 'MMM d, yyyy')
    } catch {
      return dateString
    }
  }

  const formatLabel = (value: string) => {
    return value.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
  }

  return (
    <Link
      to={`/programs/${programId}/risks/${risk.risk_id}`}
      className="block bg-white rounded-lg shadow hover:shadow-md transition-shadow duration-200 p-6"
    >
      <div className="flex items-start justify-between">
        <div className="flex items-start space-x-3 flex-1">
          <div className={`p-2 rounded-lg border-2 ${severityColors[risk.severity]}`}>
            <ExclamationTriangleIcon className="h-5 w-5" />
          </div>

          <div className="flex-1 min-w-0">
            <h3 className="text-sm font-semibold text-gray-900">{risk.title}</h3>

            <p className="text-sm text-gray-600 mt-1 line-clamp-2">{risk.description}</p>

            <div className="mt-3 flex flex-wrap items-center gap-2">
              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${severityColors[risk.severity]}`}>
                {formatLabel(risk.severity)}
              </span>
              <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColors[risk.status]}`}>
                {formatLabel(risk.status)}
              </span>
              <span className="inline-flex items-center text-xs text-gray-500">
                <span className="font-medium text-gray-700">P:</span> {formatLabel(risk.probability)}
              </span>
              <span className="inline-flex items-center text-xs text-gray-500">
                <span className="font-medium text-gray-700">I:</span> {formatLabel(risk.impact)}
              </span>
            </div>

            <div className="mt-3 flex items-center space-x-4 text-xs text-gray-500">
              {risk.owner_name?.Valid ? (
                <div className="flex items-center space-x-1">
                  <UserIcon className="h-4 w-4" />
                  <span>{risk.owner_name.String}</span>
                </div>
              ) : (
                <div className="flex items-center space-x-1 text-gray-400">
                  <UserIcon className="h-4 w-4" />
                  <span>Unassigned</span>
                </div>
              )}

              <span>•</span>
              <span>ID: {formatDate(risk.identified_date)}</span>

              {risk.target_resolution_date?.Valid && (
                <>
                  <span>•</span>
                  <span>Target: {formatDate(risk.target_resolution_date.String)}</span>
                </>
              )}
            </div>

            {risk.ai_confidence_score?.Valid && (
              <div className="mt-2 text-xs text-gray-500">
                AI Confidence: {(risk.ai_confidence_score.Float64 * 100).toFixed(0)}%
              </div>
            )}
          </div>
        </div>
      </div>
    </Link>
  )
}
