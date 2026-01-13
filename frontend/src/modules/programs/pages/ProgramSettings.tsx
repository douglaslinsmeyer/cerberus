import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeftIcon } from '@heroicons/react/24/outline'

export function ProgramSettings() {
  const { programId } = useParams<{ programId: string }>()
  const [internalOrganization, setInternalOrganization] = useState('PING')
  const [isSaving, setIsSaving] = useState(false)

  // Load current value from API
  useEffect(() => {
    fetch(`http://localhost:8080/api/v1/programs/${programId}`)
      .then(res => res.json())
      .then(data => {
        if (data.data?.internal_organization) {
          setInternalOrganization(data.data.internal_organization)
        }
      })
      .catch(err => console.error('Failed to load program:', err))
  }, [programId])

  const handleSave = async () => {
    setIsSaving(true)
    try {
      await fetch(`http://localhost:8080/api/v1/programs/${programId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ internal_organization: internalOrganization })
      })
      alert('Settings saved! AI will now use "' + internalOrganization + '" to classify internal vs external people.')
    } catch (err) {
      alert('Failed to save: ' + err)
    } finally {
      setIsSaving(false)
    }
  }

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
              disabled={isSaving}
              className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50"
            >
              {isSaving ? 'Saving...' : 'Save Settings'}
            </button>
          </div>
        </div>
      </div>

      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Simple Single-Field Configuration */}
        <div className="bg-white rounded-lg shadow p-8">
          <h2 className="text-2xl font-semibold text-gray-900 mb-2">AI Context Configuration</h2>
          <p className="text-sm text-gray-600 mb-8">
            Configure the minimal information AI needs to provide intelligent, context-aware analysis.
          </p>

          <div className="max-w-2xl">
            <label className="block text-sm font-medium text-gray-700 mb-3">
              Internal Organization Name(s)
              <span className="ml-2 text-xs text-gray-500">(Who are "we"?)</span>
            </label>
            <input
              type="text"
              value={internalOrganization}
              onChange={(e) => setInternalOrganization(e.target.value)}
              className="block w-full text-lg rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 px-4 py-3"
              placeholder="e.g., PING, Ping Identity, PING Inc"
            />
            <p className="mt-2 text-xs text-gray-500 font-medium">
              💡 Tip: Include common variations separated by commas (e.g., "PING, Ping Identity, PING Incorporated")
            </p>
            <p className="mt-3 text-sm text-gray-600">
              This is the name of YOUR organization. AI uses it to automatically classify people:
            </p>
            <ul className="mt-2 space-y-1 text-sm text-gray-600 ml-4">
              <li>• If someone's organization matches <strong>ANY of these names</strong> → <span className="font-medium text-blue-600">Internal</span></li>
              <li>• If someone's organization is different → <span className="font-medium text-gray-600">External</span></li>
            </ul>

            <div className="mt-6 p-4 bg-green-50 border border-green-200 rounded-lg">
              <h3 className="text-sm font-medium text-green-800 mb-2">✨ Everything Else is Automatic</h3>
              <p className="text-xs text-green-700">
                <strong>Stakeholders</strong> emerge from uploaded documents (emails, meeting notes, contracts).
                <br />
                <strong>Vendors</strong> emerge from invoices and financial documents.
                <br />
                <strong>Taxonomy</strong> is inferred by AI from your documents.
                <br /><br />
                No manual list maintenance required! Just set your organization name and upload documents.
              </p>
            </div>
          </div>
        </div>

        {/* How It Works */}
        <div className="mt-6 bg-blue-50 border border-blue-200 rounded-lg p-6">
          <h3 className="text-sm font-semibold text-blue-900 mb-3">How This Works</h3>
          <div className="space-y-3 text-sm text-blue-800">
            <div className="flex">
              <span className="font-bold mr-2">1.</span>
              <span>You upload an invoice from vendor "Infor (US), LLC"</span>
            </div>
            <div className="flex">
              <span className="font-bold mr-2">2.</span>
              <span>AI extracts person "Bob Teicher" working at "Infor (US), LLC"</span>
            </div>
            <div className="flex">
              <span className="font-bold mr-2">3.</span>
              <span>AI compares: "Infor (US), LLC" ≠ "{internalOrganization}" → Classifies as <strong>External</strong></span>
            </div>
            <div className="flex">
              <span className="font-bold mr-2">4.</span>
              <span>Invoice appears in Financial module with vendor properly identified</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
