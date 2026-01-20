import { Routes, Route, Link } from 'react-router-dom'
import { AuthProvider } from './contexts/AuthContext'
import { ProtectedRoute } from './components/ProtectedRoute'
import { LoginPage } from './modules/auth/pages/LoginPage'
import { NoAccessPage } from './modules/auth/pages/NoAccessPage'
import { ArtifactsPage } from './modules/artifacts/pages/ArtifactsPage'
import { ArtifactDetailPage } from './modules/artifacts/pages/ArtifactDetailPage'
import { ProgramDashboard } from './modules/programs/pages/ProgramDashboard'
import { ProgramListPage } from './modules/programs/pages/ProgramListPage'
import { ProgramSettings } from './modules/programs/pages/ProgramSettings'
import { FinancialDashboard } from './modules/financial/pages/FinancialDashboard'
import { InvoiceListPage } from './modules/financial/pages/InvoiceListPage'
import { InvoiceDetailPage } from './modules/financial/pages/InvoiceDetailPage'
import { RiskDashboard } from './modules/risk/pages/RiskDashboard'
import { RiskListPage } from './modules/risk/pages/RiskListPage'
import { RiskDetailPage } from './modules/risk/pages/RiskDetailPage'
import { StakeholdersPage } from './modules/stakeholders/pages/StakeholdersPage'
import { StakeholderDetailPage } from './modules/stakeholders/pages/StakeholderDetailPage'
import { StakeholderSuggestionsPage } from './modules/stakeholders/pages/StakeholderSuggestionsPage'

function App() {
  return (
    <AuthProvider>
      <div className="min-h-screen bg-gray-50">
        <Routes>
          {/* Public routes */}
          <Route path="/" element={<HomePage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route path="/no-access" element={<NoAccessPage />} />

          {/* Protected routes - require authentication */}
          <Route path="/programs" element={
            <ProtectedRoute>
              <ProgramListPage />
            </ProtectedRoute>
          } />

          {/* Program-scoped routes - require program access */}
          <Route path="/programs/:programId" element={
            <ProtectedRoute requireProgramAccess>
              <ProgramDashboard />
            </ProtectedRoute>
          } />
          <Route path="/programs/:programId/settings" element={
            <ProtectedRoute requireProgramAccess>
              <ProgramSettings />
            </ProtectedRoute>
          } />

          {/* Artifacts Module */}
          <Route path="/programs/:programId/artifacts" element={
            <ProtectedRoute requireProgramAccess>
              <ArtifactsPage />
            </ProtectedRoute>
          } />
          <Route path="/programs/:programId/artifacts/:artifactId" element={
            <ProtectedRoute requireProgramAccess>
              <ArtifactDetailPage />
            </ProtectedRoute>
          } />

          {/* Financial Module */}
          <Route path="/programs/:programId/financial" element={
            <ProtectedRoute requireProgramAccess>
              <FinancialDashboard />
            </ProtectedRoute>
          } />
          <Route path="/programs/:programId/financial/invoices" element={
            <ProtectedRoute requireProgramAccess>
              <InvoiceListPage />
            </ProtectedRoute>
          } />
          <Route path="/programs/:programId/financial/invoices/:invoiceId" element={
            <ProtectedRoute requireProgramAccess>
              <InvoiceDetailPage />
            </ProtectedRoute>
          } />

          {/* Risk Module */}
          <Route path="/programs/:programId/risks" element={
            <ProtectedRoute requireProgramAccess>
              <RiskDashboard />
            </ProtectedRoute>
          } />
          <Route path="/programs/:programId/risks/list" element={
            <ProtectedRoute requireProgramAccess>
              <RiskListPage />
            </ProtectedRoute>
          } />
          <Route path="/programs/:programId/risks/:riskId" element={
            <ProtectedRoute requireProgramAccess>
              <RiskDetailPage />
            </ProtectedRoute>
          } />

          {/* Stakeholders Module */}
          <Route path="/programs/:programId/stakeholders" element={
            <ProtectedRoute requireProgramAccess>
              <StakeholdersPage />
            </ProtectedRoute>
          } />
          <Route path="/programs/:programId/stakeholders/suggestions" element={
            <ProtectedRoute requireProgramAccess>
              <StakeholderSuggestionsPage />
            </ProtectedRoute>
          } />
          <Route path="/programs/:programId/stakeholders/:stakeholderId" element={
            <ProtectedRoute requireProgramAccess>
              <StakeholderDetailPage />
            </ProtectedRoute>
          } />

          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </div>
    </AuthProvider>
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
            to="/programs"
            className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 shadow-sm"
          >
            Browse All Programs
          </Link>
          <Link
            to={`/programs/${DEFAULT_PROGRAM_ID}`}
            className="inline-flex items-center px-6 py-3 border border-gray-300 text-base font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 shadow-sm"
          >
            Go to Demo Program
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
