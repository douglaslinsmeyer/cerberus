import React from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../../hooks/useAuth'

export const NoAccessPage: React.FC = () => {
  const { user, organization, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="max-w-md w-full space-y-8 p-8 bg-white rounded-lg shadow-md text-center">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">No Program Access</h2>
          <p className="mt-4 text-sm text-gray-600">
            Welcome, {user?.full_name}!
          </p>
          <p className="mt-2 text-sm text-gray-600">
            You are a member of <strong>{organization?.organization_name}</strong>, but you don't have access to any programs yet.
          </p>
          <p className="mt-4 text-sm text-gray-500">
            Contact your organization administrator to request access to a program.
          </p>
        </div>

        <button
          onClick={handleLogout}
          className="w-full flex justify-center py-2 px-4 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          Sign out
        </button>
      </div>
    </div>
  )
}
