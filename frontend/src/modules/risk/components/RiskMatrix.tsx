import React from 'react'
import { Risk } from '@/services/api'

interface RiskMatrixProps {
  risks: Risk[]
  onRiskClick?: (risk: Risk) => void
}

const PROBABILITY_LEVELS = ['very_low', 'low', 'medium', 'high', 'very_high']
const IMPACT_LEVELS = ['very_low', 'low', 'medium', 'high', 'very_high']

export function RiskMatrix({ risks, onRiskClick }: RiskMatrixProps) {
  // Group risks by probability and impact
  const risksByCell: Record<string, Risk[]> = {}

  risks.forEach((risk) => {
    const key = `${risk.probability}-${risk.impact}`
    if (!risksByCell[key]) {
      risksByCell[key] = []
    }
    risksByCell[key].push(risk)
  })

  const getCellColor = (probability: string, impact: string) => {
    const probIndex = PROBABILITY_LEVELS.indexOf(probability)
    const impactIndex = IMPACT_LEVELS.indexOf(impact)
    const score = probIndex + impactIndex

    if (score <= 2) return 'bg-green-200 hover:bg-green-300'
    if (score <= 4) return 'bg-yellow-200 hover:bg-yellow-300'
    if (score <= 6) return 'bg-orange-200 hover:bg-orange-300'
    return 'bg-red-200 hover:bg-red-300'
  }

  const formatLabel = (value: string) => {
    return value.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ')
  }

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4">Risk Matrix</h3>

      <div className="overflow-x-auto">
        <div className="inline-block min-w-full">
          {/* Header */}
          <div className="flex">
            <div className="w-24"></div>
            <div className="flex-1 text-center font-semibold text-sm text-gray-700 mb-2">
              Impact
            </div>
          </div>

          {/* Matrix Grid */}
          <div className="grid" style={{ gridTemplateColumns: 'auto repeat(5, 1fr)' }}>
            {/* Y-axis label */}
            <div className="flex items-center justify-center">
              <div className="transform -rotate-90 whitespace-nowrap font-semibold text-sm text-gray-700">
                Probability
              </div>
            </div>

            {/* Column headers */}
            {IMPACT_LEVELS.map((impact) => (
              <div key={impact} className="text-center text-xs font-medium text-gray-600 py-2">
                {formatLabel(impact)}
              </div>
            ))}

            {/* Matrix cells */}
            {[...PROBABILITY_LEVELS].reverse().map((probability) => (
              <React.Fragment key={`row-${probability}`}>
                {/* Row header */}
                <div key={`header-${probability}`} className="flex items-center justify-end pr-3 text-xs font-medium text-gray-600">
                  {formatLabel(probability)}
                </div>

                {/* Cells for this probability level */}
                {IMPACT_LEVELS.map((impact) => {
                  const key = `${probability}-${impact}`
                  const cellRisks = risksByCell[key] || []

                  return (
                    <div
                      key={key}
                      className={`border border-gray-300 min-h-[80px] p-2 transition-colors ${getCellColor(probability, impact)}`}
                    >
                      {cellRisks.length > 0 && (
                        <div className="space-y-1">
                          {cellRisks.slice(0, 3).map((risk) => (
                            <button
                              key={risk.risk_id}
                              onClick={() => onRiskClick?.(risk)}
                              className="w-full text-left text-xs bg-white bg-opacity-80 hover:bg-opacity-100 rounded px-2 py-1 truncate transition-all"
                              title={risk.title}
                            >
                              {risk.title}
                            </button>
                          ))}
                          {cellRisks.length > 3 && (
                            <div className="text-xs text-gray-600 text-center">
                              +{cellRisks.length - 3} more
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  )
                })}
              </React.Fragment>
            ))}
          </div>

          {/* Legend */}
          <div className="mt-4 flex items-center justify-center space-x-6 text-xs">
            <div className="flex items-center space-x-2">
              <div className="w-4 h-4 bg-green-200 border border-gray-300"></div>
              <span className="text-gray-600">Low</span>
            </div>
            <div className="flex items-center space-x-2">
              <div className="w-4 h-4 bg-yellow-200 border border-gray-300"></div>
              <span className="text-gray-600">Medium</span>
            </div>
            <div className="flex items-center space-x-2">
              <div className="w-4 h-4 bg-orange-200 border border-gray-300"></div>
              <span className="text-gray-600">High</span>
            </div>
            <div className="flex items-center space-x-2">
              <div className="w-4 h-4 bg-red-200 border border-gray-300"></div>
              <span className="text-gray-600">Critical</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
