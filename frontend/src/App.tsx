import { Routes, Route, Link } from 'react-router-dom'
import { ArtifactsPage } from './modules/artifacts/pages/ArtifactsPage'
import { ArtifactDetailPage } from './modules/artifacts/pages/ArtifactDetailPage'

function App() {
  return (
    <div className="min-h-screen bg-gray-50">
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/programs/:programId/artifacts" element={<ArtifactsPage />} />
        <Route path="/programs/:programId/artifacts/:artifactId" element={<ArtifactDetailPage />} />
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
          Phase 2: Artifacts Module - AI-Powered Document Analysis
        </p>

        <div className="flex flex-col sm:flex-row gap-4 justify-center">
          <Link
            to={`/programs/${DEFAULT_PROGRAM_ID}/artifacts`}
            className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 shadow-sm"
          >
            Go to Artifacts
          </Link>
        </div>

        <div className="mt-12 text-left bg-white rounded-lg shadow p-6">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">Features Available</h2>
          <ul className="space-y-2 text-sm text-gray-600">
            <li className="flex items-start">
              <span className="text-green-600 mr-2">✓</span>
              <span>Upload PDFs and text documents</span>
            </li>
            <li className="flex items-start">
              <span className="text-green-600 mr-2">✓</span>
              <span>Automatic AI-powered content extraction</span>
            </li>
            <li className="flex items-start">
              <span className="text-green-600 mr-2">✓</span>
              <span>Metadata analysis (topics, persons, facts, insights)</span>
            </li>
            <li className="flex items-start">
              <span className="text-green-600 mr-2">✓</span>
              <span>Vector embeddings for semantic search</span>
            </li>
            <li className="flex items-start">
              <span className="text-green-600 mr-2">✓</span>
              <span>Real-time processing status updates</span>
            </li>
          </ul>
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
