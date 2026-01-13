import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeftIcon, BuildingOfficeIcon, UserGroupIcon, TagIcon } from '@heroicons/react/24/outline'

export function ProgramSettings() {
  const { programId } = useParams<{ programId: string }>()
  const [activeTab, setActiveTab] = useState<'company' | 'stakeholders' | 'vendors' | 'taxonomy'>('company')

  const [companyName, setCompanyName] = useState('PING, Inc.')
  const [legalName, setLegalName] = useState('PING Incorporated')
  const [aliases, setAliases] = useState('PING, Ping Identity')

  const [stakeholders, setStakeholders] = useState([
    { name: 'John Smith', role: 'Program Director', type: 'internal', engagement: 'key' },
    { name: 'Sarah Chen', role: 'Finance Lead', type: 'internal', engagement: 'primary' },
  ])

  const [vendors, setVendors] = useState([
    { name: 'Infor (US), LLC', type: 'software_vendor' },
    { name: 'Microsoft', type: 'cloud_provider' },
  ])

  const [riskCategories, setRiskCategories] = useState('vendor, security, compliance, technical, financial')
  const [spendCategories, setSpendCategories] = useState('software, consulting, infrastructure, support, labor')

  const handleSave = () => {
    // TODO: Save to API
    alert('Configuration saved! (API integration pending)')
  }

  const tabs = [
    { id: 'company', name: 'Company Info', icon: BuildingOfficeIcon },
    { id: 'stakeholders', name: 'Stakeholders', icon: UserGroupIcon },
    { id: 'vendors', name: 'Vendors', icon: TagIcon },
    { id: 'taxonomy', name: 'Taxonomy', icon: TagIcon },
  ]

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <Link
                to={`/programs/${programId}`}
                className="text-gray-400 hover:text-gray-600"
              >
                <ArrowLeftIcon className="h-6 w-6" />
              </Link>
              <div>
                <h1 className="text-2xl font-bold text-gray-900">Program Settings</h1>
                <p className="mt-1 text-sm text-gray-500">Configure AI context and stakeholders</p>
              </div>
            </div>

            <button
              onClick={handleSave}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700"
            >
              Save Changes
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Tabs */}
        <div className="border-b border-gray-200 mb-6">
          <nav className="-mb-px flex space-x-8">
            {tabs.map((tab) => {
              const Icon = tab.icon
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as any)}
                  className={`
                    flex items-center py-4 px-1 border-b-2 font-medium text-sm
                    ${activeTab === tab.id
                      ? 'border-blue-500 text-blue-600'
                      : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                    }
                  `}
                >
                  <Icon className="h-5 w-5 mr-2" />
                  {tab.name}
                </button>
              )
            })}
          </nav>
        </div>

        {/* Company Info Tab */}
        {activeTab === 'company' && (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Company Information</h2>
            <p className="text-sm text-gray-600 mb-6">
              Configure your organization name so AI can distinguish internal vs external people.
            </p>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Company Name
                </label>
                <input
                  type="text"
                  value={companyName}
                  onChange={(e) => setCompanyName(e.target.value)}
                  className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                  placeholder="e.g., PING, Inc."
                />
                <p className="mt-1 text-xs text-gray-500">
                  Primary company name used in documents
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Full Legal Name
                </label>
                <input
                  type="text"
                  value={legalName}
                  onChange={(e) => setLegalName(e.target.value)}
                  className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                  placeholder="e.g., PING Incorporated"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Aliases (comma-separated)
                </label>
                <input
                  type="text"
                  value={aliases}
                  onChange={(e) => setAliases(e.target.value)}
                  className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                  placeholder="e.g., PING, Ping Identity, PING Inc"
                />
                <p className="mt-1 text-xs text-gray-500">
                  Other names your company might be referred to as
                </p>
              </div>
            </div>
          </div>
        )}

        {/* Stakeholders Tab */}
        {activeTab === 'stakeholders' && (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Internal Stakeholders</h2>
            <p className="text-sm text-gray-600 mb-6">
              Define key internal people so AI can recognize them in documents.
            </p>

            <div className="overflow-hidden">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Role</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Type</th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Engagement</th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {stakeholders.map((stakeholder, idx) => (
                    <tr key={idx}>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        {stakeholder.name}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {stakeholder.role}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <span className="px-2 py-1 text-xs rounded-full bg-blue-100 text-blue-800">
                          {stakeholder.type}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                        {stakeholder.engagement}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>

            <button className="mt-4 text-sm text-blue-600 hover:text-blue-700">
              + Add Stakeholder
            </button>
          </div>
        )}

        {/* Vendors Tab */}
        {activeTab === 'vendors' && (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Known Vendors</h2>
            <p className="text-sm text-gray-600 mb-6">
              Define external vendors so AI can recognize them in invoices and contracts.
            </p>

            <div className="space-y-2">
              {vendors.map((vendor, idx) => (
                <div key={idx} className="flex items-center justify-between p-3 border rounded-lg">
                  <div>
                    <p className="text-sm font-medium text-gray-900">{vendor.name}</p>
                    <p className="text-xs text-gray-500">{vendor.type}</p>
                  </div>
                </div>
              ))}
            </div>

            <button className="mt-4 text-sm text-blue-600 hover:text-blue-700">
              + Add Vendor
            </button>
          </div>
        )}

        {/* Taxonomy Tab */}
        {activeTab === 'taxonomy' && (
          <div className="bg-white rounded-lg shadow p-6">
            <h2 className="text-lg font-semibold text-gray-900 mb-4">Custom Taxonomy</h2>
            <p className="text-sm text-gray-600 mb-6">
              Define program-specific categories for AI classification.
            </p>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Risk Categories
                </label>
                <input
                  type="text"
                  value={riskCategories}
                  onChange={(e) => setRiskCategories(e.target.value)}
                  className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                  placeholder="Comma-separated categories"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Spend Categories
                </label>
                <input
                  type="text"
                  value={spendCategories}
                  onChange={(e) => setSpendCategories(e.target.value)}
                  className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                  placeholder="Comma-separated categories"
                />
              </div>
            </div>
          </div>
        )}

        {/* Info Box */}
        <div className="mt-6 bg-blue-50 border border-blue-200 rounded-lg p-4">
          <p className="text-sm text-blue-700">
            <strong>AI Context:</strong> This configuration is used by Claude AI to provide program-specific analysis.
            Company info helps distinguish internal vs external people. Taxonomy ensures consistent categorization.
          </p>
        </div>
      </div>
    </div>
  )
}
