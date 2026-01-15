interface EngagementTrackerProps {
  level?: string
  className?: string
}

const engagementConfig = {
  key: {
    label: 'Key',
    color: 'bg-red-100 text-red-800 border-red-200',
    dotColor: 'bg-red-500',
  },
  primary: {
    label: 'Primary',
    color: 'bg-orange-100 text-orange-800 border-orange-200',
    dotColor: 'bg-orange-500',
  },
  secondary: {
    label: 'Secondary',
    color: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    dotColor: 'bg-yellow-500',
  },
  observer: {
    label: 'Observer',
    color: 'bg-gray-100 text-gray-800 border-gray-200',
    dotColor: 'bg-gray-500',
  },
}

export function EngagementTracker({ level, className = '' }: EngagementTrackerProps) {
  if (!level || !(level in engagementConfig)) {
    return (
      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-400 border border-gray-200 ${className}`}>
        <span className="w-2 h-2 rounded-full bg-gray-400 mr-1.5"></span>
        Not set
      </span>
    )
  }

  const config = engagementConfig[level as keyof typeof engagementConfig]

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${config.color} ${className}`}>
      <span className={`w-2 h-2 rounded-full ${config.dotColor} mr-1.5`}></span>
      {config.label}
    </span>
  )
}
