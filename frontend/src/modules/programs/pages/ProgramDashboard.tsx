import { Link, useParams } from 'react-router-dom'
import {
  DocumentTextIcon,
  CurrencyDollarIcon,
  ExclamationTriangleIcon,
  ChatBubbleLeftRightIcon,
  UserGroupIcon,
  ClipboardDocumentListIcon,
  ChartBarIcon,
  FlagIcon,
  ShieldCheckIcon,
  ArrowPathIcon,
} from '@heroicons/react/24/outline'
import { useArtifacts } from '../../artifacts/hooks/useArtifacts'

export function ProgramDashboard() {
  const { programId } = useParams<{ programId: string }>()
  const { data: artifacts = [] } = useArtifacts(programId || '', {})

  // Hardcoded program data for Phase 2 (no program API yet)
  const program = {
    program_id: programId,
    program_name: 'Default Program',
    program_code: 'DEFAULT',
    description: 'Default program for Phase 2 testing and development',
    status: 'active',
    start_date: '2026-01-01',
  }

  const modules = [
    {
      name: 'Artifacts',
      icon: DocumentTextIcon,
      description: 'AI-powered document analysis',
      link: `/programs/${programId}/artifacts`,
      status: 'active',
      color: 'bg-blue-500',
      stats: `${artifacts.length} documents`,
    },
    {
      name: 'Financial',
      icon: CurrencyDollarIcon,
      description: 'Invoice validation & spend tracking',
      link: '#',
      status: 'coming-soon',
      color: 'bg-green-500',
      stats: 'Phase 3',
    },
    {
      name: 'Risk & Issues',
      icon: ExclamationTriangleIcon,
      description: 'Risk identification & mitigation',
      link: '#',
      status: 'coming-soon',
      color: 'bg-red-500',
      stats: 'Phase 3',
    },
    {
      name: 'Communications',
      icon: ChatBubbleLeftRightIcon,
      description: 'Stakeholder communications',
      link: '#',
      status: 'coming-soon',
      color: 'bg-purple-500',
      stats: 'Phase 4',
    },
    {
      name: 'Stakeholders',
      icon: UserGroupIcon,
      description: 'Stakeholder management',
      link: '#',
      status: 'coming-soon',
      color: 'bg-indigo-500',
      stats: 'Phase 4',
    },
    {
      name: 'Decisions',
      icon: ClipboardDocumentListIcon,
      description: 'Decision log & tracking',
      link: '#',
      status: 'coming-soon',
      color: 'bg-yellow-500',
      stats: 'Phase 5',
    },
    {
      name: 'Dashboard',
      icon: ChartBarIcon,
      description: 'Executive health scoring',
      link: '#',
      status: 'coming-soon',
      color: 'bg-orange-500',
      stats: 'Phase 5',
    },
    {
      name: 'Milestones',
      icon: FlagIcon,
      description: 'Phase & milestone tracking',
      link: '#',
      status: 'coming-soon',
      color: 'bg-pink-500',
      stats: 'Phase 6',
    },
    {
      name: 'Governance',
      icon: ShieldCheckIcon,
      description: 'Compliance & audit',
      link: '#',
      status: 'coming-soon',
      color: 'bg-cyan-500',
      stats: 'Phase 6',
    },
    {
      name: 'Change Control',
      icon: ArrowPathIcon,
      description: 'Change impact analysis',
      link: '#',
      status: 'coming-soon',
      color: 'bg-teal-500',
      stats: 'Phase 7',
    },
  ]

  const completedArtifacts = artifacts.filter(a => a.processing_status === 'completed').length
  const pendingArtifacts = artifacts.filter(a => a.processing_status === 'pending').length

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center justify-between">
            <div>
              <div className="flex items-center space-x-3">
                <h1 className="text-3xl font-bold text-gray-900">{program.program_name}</h1>
                <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800">
                  {program.status}
                </span>
              </div>
              <p className="mt-1 text-sm text-gray-500">{program.program_code}</p>
              <p className="mt-2 text-sm text-gray-600">{program.description}</p>
            </div>

            <Link
              to="/"
              className="text-sm text-gray-600 hover:text-gray-900"
            >
              ← Back to Home
            </Link>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Quick Stats */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <DocumentTextIcon className="h-8 w-8 text-blue-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Total Artifacts</p>
                <p className="text-2xl font-semibold text-gray-900">{artifacts.length}</p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <DocumentTextIcon className="h-8 w-8 text-green-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Analyzed</p>
                <p className="text-2xl font-semibold text-green-600">{completedArtifacts}</p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <DocumentTextIcon className="h-8 w-8 text-yellow-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Pending Analysis</p>
                <p className="text-2xl font-semibold text-yellow-600">{pendingArtifacts}</p>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center">
              <ChartBarIcon className="h-8 w-8 text-purple-600" />
              <div className="ml-4">
                <p className="text-sm text-gray-500">Modules Active</p>
                <p className="text-2xl font-semibold text-purple-600">1/10</p>
              </div>
            </div>
          </div>
        </div>

        {/* Modules Grid */}
        <div>
          <h2 className="text-2xl font-bold text-gray-900 mb-6">Program Modules</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {modules.map((module) => {
              const Icon = module.icon
              const isActive = module.status === 'active'

              return (
                <Link
                  key={module.name}
                  to={isActive ? module.link : '#'}
                  className={`
                    bg-white rounded-lg shadow p-6 transition-all duration-200
                    ${isActive ? 'hover:shadow-lg cursor-pointer' : 'opacity-60 cursor-not-allowed'}
                  `}
                  onClick={(e) => {
                    if (!isActive) {
                      e.preventDefault()
                    }
                  }}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex items-start space-x-3">
                      <div className={`${module.color} p-2 rounded-lg`}>
                        <Icon className="h-6 w-6 text-white" />
                      </div>
                      <div>
                        <h3 className="text-lg font-semibold text-gray-900">{module.name}</h3>
                        <p className="mt-1 text-sm text-gray-600">{module.description}</p>
                        <p className="mt-2 text-xs text-gray-500">{module.stats}</p>
                      </div>
                    </div>

                    {module.status === 'coming-soon' && (
                      <span className="inline-flex items-center px-2 py-1 rounded text-xs font-medium bg-gray-100 text-gray-600">
                        Coming Soon
                      </span>
                    )}
                  </div>
                </Link>
              )
            })}
          </div>
        </div>

        {/* Info Box */}
        <div className="mt-8 bg-blue-50 border border-blue-200 rounded-lg p-6">
          <h3 className="text-lg font-semibold text-blue-900 mb-2">Phase 2: Artifacts Module Complete</h3>
          <p className="text-sm text-blue-700">
            The Artifacts module is fully operational with AI-powered document analysis.
            Upload PDFs or text files to see Claude extract topics, insights, and structured metadata.
          </p>
          <p className="mt-2 text-sm text-blue-700">
            Remaining modules (Financial, Risk, Communications, etc.) will be implemented in Phase 3-7.
          </p>
        </div>
      </div>
    </div>
  )
}
