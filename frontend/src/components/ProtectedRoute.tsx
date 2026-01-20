import React from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'

interface ProtectedRouteProps {
  children: React.ReactNode
  requireAuth?: boolean
  requiredRole?: string
  requireProgramAccess?: boolean
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({
  children,
  requireAuth = true,
  requiredRole,
  requireProgramAccess = false,
}) => {
  const { isAuthenticated, isLoading, currentProgram, hasProgramRole } = useAuth()
  const location = useLocation()

  // Show loading spinner during initial auth check
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-center">
          <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900"></div>
          <p className="mt-2 text-sm text-gray-600">Loading...</p>
        </div>
      </div>
    )
  }

  // Check authentication
  if (requireAuth && !isAuthenticated) {
    // Redirect to login, preserving intended destination
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  // Check program access
  if (requireProgramAccess && !currentProgram) {
    return <Navigate to="/no-access" replace />
  }

  // Check role requirements
  if (requiredRole && !hasProgramRole(requiredRole)) {
    return <Navigate to="/unauthorized" replace />
  }

  return <>{children}</>
}
