import { Routes, Route, Link } from 'react-router-dom'
import { ArtifactsPage } from './modules/artifacts/pages/ArtifactsPage'
import { ArtifactDetailPage } from './modules/artifacts/pages/ArtifactDetailPage'
import { ProgramDashboard } from './modules/programs/pages/ProgramDashboard'
import { FinancialDashboard } from './modules/financial/pages/FinancialDashboard'
import { InvoiceListPage } from './modules/financial/pages/InvoiceListPage'
import { InvoiceDetailPage } from './modules/financial/pages/InvoiceDetailPage'
import { RiskDashboard } from './modules/risk/pages/RiskDashboard'
import { RiskListPage } from './modules/risk/pages/RiskListPage'
import { RiskDetailPage } from './modules/risk/pages/RiskDetailPage'

function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/programs/:programId" element={<ProgramDashboard />} />

        {/* Artifacts Module */}
        <Route path="/programs/:programId/artifacts" element={<ArtifactsPage />} />
        <Route path="/programs/:programId/artifacts/:artifactId" element={<ArtifactDetailPage />} />

        {/* Financial Module */}
        <Route path="/programs/:programId/financial" element={<FinancialDashboard />} />
        <Route path="/programs/:programId/financial/invoices" element={<InvoiceListPage />} />
        <Route path="/programs/:programId/financial/invoices/:invoiceId" element={<InvoiceDetailPage />} />

        {/* Risk Module */}
        <Route path="/programs/:programId/risks" element={<RiskDashboard />} />
        <Route path="/programs/:programId/risks/list" element={<RiskListPage />} />
        <Route path="/programs/:programId/risks/:riskId" element={<RiskDetailPage />} />

        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </div>
  )
}

function HomePage() {
  const DEFAULT_PROGRAM_ID = '00000000-0000-0000-0000-000000000001'

  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="text-center max-w-2xl px-4">
        <h1 className="text-4xl font-bold text-gray-900 mb-4">
          Cerberus
        </h1>
        <p className="text-xl text-gray-600 mb-8">
          Enterprise Program Governance System
        </p>
        <p className="text-sm text-gray-500 mb-8">
          Phase 3: Financial & Risk Modules - AI-Powered Invoice Validation & Risk Management
        </p>

        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <Link
            to={`/programs/${DEFAULT_PROGRAM_ID}`}
            className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 shadow-sm"
          >
            Go to Program Dashboard
          </Link>
          <Link
            to={`/programs/${DEFAULT_PROGRAM_ID}/artifacts`}
            className="inline-flex items-center px-6 py-3 border border-gray-300 text-base font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 shadow-sm"
          >
            Go to Artifacts
          </Link>
        </div>

        <div className="mt-12 text-left bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Features Available</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <h3 className="text-sm font-semibold text-gray-900 mb-2">Artifacts Module</h3>
              <ul className="space-y-1 text-sm text-gray-600">
                <li className="flex items-start">
                  <span className="text-green-600 mr-2">✓</span>
                  <span>AI-powered document analysis</span>
                </li>
                <li className="flex items-start">
                  <span className="text-green-600 mr-2">✓</span>
                  <span>Semantic search & metadata extraction</span>
                </li>
              </ul>
            </div>
            <div>
              <h3 className="text-sm font-semibold text-gray-900 mb-2">Financial Module</h3>
              <ul className="space-y-1 text-sm text-gray-600">
                <li className="flex items-start">
                  <span className="text-green-600 mr-2">✓</span>
                  <span>AI invoice validation & variance detection</span>
                </li>
                <li className="flex items-start">
                  <span className="text-green-600 mr-2">✓</span>
                  <span>Budget tracking & spend analysis</span>
                </li>
              </ul>
            </div>
            <div>
              <h3 className="text-sm font-semibold text-gray-900 mb-2">Risk Module</h3>
              <ul className="space-y-1 text-sm text-gray-600">
                <li className="flex items-start">
                  <span className="text-green-600 mr-2">✓</span>
                  <span>AI risk identification & suggestions</span>
                </li>
                <li className="flex items-start">
                  <span className="text-green-600 mr-2">✓</span>
                  <span>Mitigation planning & tracking</span>
                </li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

function LoginPage() {
  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="w-full max-w-md">
        <div className="bg-white shadow-md rounded-lg px-8 pt-6 pb-8 mb-4">
          <h2 className="text-2xl font-bold mb-6 text-center">Login</h2>
          <p className="text-center text-gray-600">
            Authentication coming soon...
          </p>
        </div>
      </div>
    </div>
  )
}

function NotFoundPage() {
  return (
    <div className="flex items-center justify-center min-h-screen">
      <div className="text-center">
        <h1 className="text-6xl font-bold text-gray-900 mb-4">404</h1>
        <p className="text-xl text-gray-600">Page not found</p>
      </div>
    </div>
  )
}

export default App
