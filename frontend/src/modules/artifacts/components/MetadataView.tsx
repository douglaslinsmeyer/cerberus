import { ArtifactWithMetadata } from '@/services/api'
import {
  DocumentTextIcon,
  UserGroupIcon,
  LightBulbIcon,
  ChartBarIcon,
  ExclamationTriangleIcon,
} from '@heroicons/react/24/outline'

interface MetadataViewProps {
  artifact: ArtifactWithMetadata
}

export function MetadataView({ artifact }: MetadataViewProps) {
  if (artifact.processing_status === 'ocr_required') {
    return (
      <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-8 text-center">
        <ExclamationTriangleIcon className="h-12 w-12 text-yellow-600 mx-auto" />
        <p className="mt-4 text-yellow-700 font-medium">OCR Required</p>
        <p className="mt-2 text-sm text-yellow-600">
          This appears to be a scanned PDF or image-based document.
        </p>
        <p className="mt-1 text-sm text-yellow-600">
          OCR (Optical Character Recognition) will be added in a future release.
        </p>
        <p className="mt-3 text-xs text-yellow-500">
          For now, please use text-based PDFs or manually enter the document content.
        </p>
      </div>
    )
  }

  if (artifact.processing_status === 'pending' || artifact.processing_status === 'processing') {
    return (
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-8 text-center">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
        <p className="mt-4 text-blue-700 font-medium">AI Analysis in Progress...</p>
        <p className="mt-2 text-sm text-blue-600">
          Extracting metadata, topics, insights, and generating embeddings
        </p>
      </div>
    )
  }

  if (artifact.processing_status === 'failed') {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-8 text-center">
        <ExclamationTriangleIcon className="h-12 w-12 text-red-600 mx-auto" />
        <p className="mt-4 text-red-700 font-medium">Analysis Failed</p>
        <p className="mt-2 text-sm text-red-600">
          The AI analysis could not be completed. Try reanalyzing the artifact.
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {/* Summary */}
      {artifact.summary && (
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-start">
            <DocumentTextIcon className="h-6 w-6 text-blue-600 mt-1" />
            <div className="ml-4 flex-1">
              <h2 className="text-lg font-semibold text-gray-900">Executive Summary</h2>
              <p className="mt-2 text-gray-700">{artifact.summary.executive_summary}</p>

              <div className="mt-4 flex items-center space-x-4">
                {artifact.summary.sentiment?.Valid && (
                  <span className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${
                    artifact.summary.sentiment.String === 'positive' ? 'bg-green-100 text-green-800' :
                    artifact.summary.sentiment.String === 'concern' ? 'bg-yellow-100 text-yellow-800' :
                    artifact.summary.sentiment.String === 'negative' ? 'bg-red-100 text-red-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    Sentiment: {artifact.summary.sentiment.String}
                  </span>
                )}

                {artifact.summary.priority?.Valid && (
                  <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-purple-100 text-purple-800">
                    Priority: {artifact.summary.priority.Int32}/5
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Topics */}
      {artifact.topics && artifact.topics.length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-start">
            <ChartBarIcon className="h-6 w-6 text-green-600 mt-1" />
            <div className="ml-4 flex-1">
              <h2 className="text-lg font-semibold text-gray-900">Topics</h2>
              <div className="mt-4 flex flex-wrap gap-2">
                {artifact.topics.map((topic) => (
                  <span
                    key={topic.topic_id}
                    className="inline-flex items-center px-3 py-1 rounded-full text-sm bg-gray-100 text-gray-800"
                  >
                    {topic.topic_name}
                    <span className="ml-2 text-xs text-gray-500">
                      {(topic.confidence_score * 100).toFixed(0)}%
                    </span>
                  </span>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Persons */}
      {artifact.persons && artifact.persons.length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-start">
            <UserGroupIcon className="h-6 w-6 text-indigo-600 mt-1" />
            <div className="ml-4 flex-1">
              <h2 className="text-lg font-semibold text-gray-900">People Mentioned</h2>
              <div className="mt-4 space-y-3">
                {artifact.persons.map((person) => (
                  <div key={person.person_id} className="flex items-start justify-between">
                    <div>
                      <p className="font-medium text-gray-900">{person.person_name}</p>
                      {person.person_role?.Valid && (
                        <p className="text-sm text-gray-600">{person.person_role.String}</p>
                      )}
                      {person.person_organization?.Valid && (
                        <p className="text-sm text-gray-500">{person.person_organization.String}</p>
                      )}
                    </div>
                    {person.confidence_score?.Valid && (
                      <span className="text-xs text-gray-500">
                        {(person.confidence_score.Float64 * 100).toFixed(0)}% confidence
                      </span>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Facts */}
      {artifact.facts && artifact.facts.length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-start">
            <ChartBarIcon className="h-6 w-6 text-purple-600 mt-1" />
            <div className="ml-4 flex-1">
              <h2 className="text-lg font-semibold text-gray-900">Key Facts</h2>
              <div className="mt-4 grid grid-cols-1 md:grid-cols-2 gap-4">
                {artifact.facts.map((fact) => (
                  <div key={fact.fact_id} className="border border-gray-200 rounded-lg p-3">
                    <p className="text-xs text-gray-500 uppercase">{fact.fact_type}</p>
                    <p className="mt-1 font-medium text-gray-900">{fact.fact_key}</p>
                    <p className="mt-1 text-sm text-gray-700">{fact.fact_value}</p>
                    {fact.confidence_score?.Valid && (
                      <p className="mt-1 text-xs text-gray-500">
                        {(fact.confidence_score.Float64 * 100).toFixed(0)}% confidence
                      </p>
                    )}
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Insights */}
      {artifact.insights && artifact.insights.length > 0 && (
        <div className="bg-white rounded-lg shadow p-6">
          <div className="flex items-start">
            <LightBulbIcon className="h-6 w-6 text-yellow-600 mt-1" />
            <div className="ml-4 flex-1">
              <h2 className="text-lg font-semibold text-gray-900">AI Insights</h2>
              <div className="mt-4 space-y-4">
                {artifact.insights.map((insight) => {
                  const severityValue = insight.severity?.Valid ? insight.severity.String : 'low'
                  return (
                    <div
                      key={insight.insight_id}
                      className={`border-l-4 rounded-r-lg p-4 ${
                        severityValue === 'critical' ? 'border-red-500 bg-red-50' :
                        severityValue === 'high' ? 'border-orange-500 bg-orange-50' :
                        severityValue === 'medium' ? 'border-yellow-500 bg-yellow-50' :
                        'border-blue-500 bg-blue-50'
                      }`}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center space-x-2">
                            <span className="text-xs font-medium text-gray-500 uppercase">
                              {insight.insight_type}
                            </span>
                            {insight.severity?.Valid && (
                              <span className={`text-xs px-2 py-0.5 rounded ${
                                severityValue === 'critical' ? 'bg-red-200 text-red-800' :
                                severityValue === 'high' ? 'bg-orange-200 text-orange-800' :
                                severityValue === 'medium' ? 'bg-yellow-200 text-yellow-800' :
                                'bg-blue-200 text-blue-800'
                              }`}>
                                {severityValue}
                              </span>
                            )}
                          </div>
                          <h3 className="mt-1 font-semibold text-gray-900">{insight.title}</h3>
                          <p className="mt-2 text-sm text-gray-700">{insight.description}</p>
                          {insight.suggested_action?.Valid && (
                            <p className="mt-2 text-sm text-gray-600">
                              <span className="font-medium">Suggested Action:</span> {insight.suggested_action.String}
                            </p>
                          )}
                        </div>
                        {insight.confidence_score?.Valid && (
                          <span className="text-xs text-gray-500 ml-4">
                            {(insight.confidence_score.Float64 * 100).toFixed(0)}%
                          </span>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* No metadata yet */}
      {!artifact.summary && (!artifact.topics || artifact.topics.length === 0) && (
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
          <p className="text-gray-500">No metadata extracted yet.</p>
          <p className="mt-2 text-sm text-gray-400">
            {artifact.processing_status === 'completed'
              ? 'AI analysis completed but no metadata was extracted.'
              : 'Upload is complete. AI analysis will begin shortly.'}
          </p>
        </div>
      )}
    </div>
  )
}
