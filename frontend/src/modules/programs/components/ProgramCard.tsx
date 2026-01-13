import { Link } from 'react-router-dom'
import { ProgramWithStats } from '../../../services/api'
import { DocumentTextIcon, CurrencyDollarIcon, ExclamationTriangleIcon } from '@heroicons/react/24/outline'

interface ProgramCardProps {
  program: ProgramWithStats
}

const statusColors = {
  active: 'bg-green-100 text-green-800',
  planning: 'bg-blue-100 text-blue-800',
  'on-hold': 'bg-yellow-100 text-yellow-800',
  completed: 'bg-gray-100 text-gray-800',
  archived: 'bg-gray-100 text-gray-600',
}

export function ProgramCard({ program }: ProgramCardProps) {
  const description = program.description?.Valid ? program.description.String : ''
  const truncatedDescription = description.length > 120 ? description.substring(0, 120) + '...' : description

  return (
    <Link
      to={`/programs/${program.program_id}`}
      className="block bg-white rounded-lg shadow hover:shadow-md transition-shadow duration-200 p-6"
    >
      <div className="flex items-start justify-between mb-3">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">{program.program_name}</h3>
          <p className="text-sm text-gray-500 mt-1">{program.program_code}</p>
        </div>
        <span
          className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
            statusColors[program.status] || statusColors.active
          }`}
        >
          {program.status}
        </span>
      </div>

      {truncatedDescription && (
        <p className="text-sm text-gray-600 mb-4 line-clamp-3">{truncatedDescription}</p>
      )}

      <div className="grid grid-cols-3 gap-4 pt-4 border-t border-gray-200">
        <div className="flex items-center">
          <DocumentTextIcon className="h-5 w-5 text-blue-500 mr-2" />
          <div>
            <p className="text-xs text-gray-500">Artifacts</p>
            <p className="text-sm font-semibold text-gray-900">{program.artifact_count}</p>
          </div>
        </div>
        <div className="flex items-center">
          <CurrencyDollarIcon className="h-5 w-5 text-green-500 mr-2" />
          <div>
            <p className="text-xs text-gray-500">Invoices</p>
            <p className="text-sm font-semibold text-gray-900">{program.invoice_count}</p>
          </div>
        </div>
        <div className="flex items-center">
          <ExclamationTriangleIcon className="h-5 w-5 text-orange-500 mr-2" />
          <div>
            <p className="text-xs text-gray-500">Risks</p>
            <p className="text-sm font-semibold text-gray-900">{program.risk_count}</p>
          </div>
        </div>
      </div>
    </Link>
  )
}
