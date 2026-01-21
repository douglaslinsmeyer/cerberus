import React, { createContext, useState, useEffect, useRef, useCallback } from 'react'
import { authService } from '../services/authApi'
import { setAuthContext } from '../services/api'
import type { UserInfo, OrganizationInfo, ProgramAccess, LoginResponse } from '../types/auth'

interface AuthState {
  user: UserInfo | null
  organization: OrganizationInfo | null
  currentProgram: ProgramAccess | null
  availablePrograms: ProgramAccess[]
  accessToken: string | null
  tokenExpiresAt: number | null
  isAuthenticated: boolean
  isLoading: boolean
}

interface AuthContextValue extends AuthState {
  login: (email: string, password: string) => Promise<void>
  logout: () => Promise<void>
  switchProgram: (programId: string) => Promise<void>
  refreshAccessToken: () => Promise<void>
  hasOrgRole: (role: string) => boolean
  hasProgramRole: (role: string) => boolean
}

export const AuthContext = createContext<AuthContextValue | undefined>(undefined)

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [state, setState] = useState<AuthState>({
    user: null,
    organization: null,
    currentProgram: null,
    availablePrograms: [],
    accessToken: null,
    tokenExpiresAt: null,
    isAuthenticated: false,
    isLoading: true,
  })

  const refreshTimerRef = useRef<NodeJS.Timeout | null>(null)

  // Define functions first before useEffects

  const login = useCallback(async (email: string, password: string) => {
    try {
      const data: LoginResponse = await authService.login({ email, password })

      setState({
        user: data.user,
        organization: data.organization,
        currentProgram: data.current_program,
        availablePrograms: data.programs,
        accessToken: data.tokens?.access_token || null,
        tokenExpiresAt: data.tokens ? Date.now() + (data.tokens.expires_in * 1000) : null,
        isAuthenticated: !!data.tokens,
        isLoading: false,
      })
    } catch (error) {
      setState(prev => ({ ...prev, isLoading: false }))
      throw error
    }
  }, [])

  const refreshAccessToken = useCallback(async () => {
    try {
      const data = await authService.refresh()
      console.log('Refresh response:', data)

      setState({
        user: data.user,
        organization: data.organization,
        currentProgram: data.current_program,
        availablePrograms: data.programs,
        accessToken: data.access_token,
        tokenExpiresAt: Date.now() + (data.expires_in * 1000),
        isAuthenticated: true,
        isLoading: false,
      })
    } catch (error) {
      // Refresh failed - logout
      console.error('Refresh failed:', error)
      setState({
        user: null,
        organization: null,
        currentProgram: null,
        availablePrograms: [],
        accessToken: null,
        tokenExpiresAt: null,
        isAuthenticated: false,
        isLoading: false,
      })
      throw error
    }
  }, [])

  const logout = useCallback(async () => {
    try {
      await authService.logout()
    } catch (error) {
      console.error('Logout API call:', error)
    } finally {
      // Clear all state
      setState({
        user: null,
        organization: null,
        currentProgram: null,
        availablePrograms: [],
        accessToken: null,
        tokenExpiresAt: null,
        isAuthenticated: false,
        isLoading: false,
      })

      // Clear auto-refresh timer
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current)
      }
    }
  }, [])

  const switchProgram = useCallback(async (programId: string) => {
    try {
      const data = await authService.switchProgram({ program_id: programId })

      // Find the program in available programs
      const program = state.availablePrograms.find(p => p.program_id === programId)

      setState(prev => ({
        ...prev,
        currentProgram: program || prev.currentProgram,
        accessToken: data.access_token,
        tokenExpiresAt: Date.now() + (data.expires_in * 1000),
      }))

      // Return programId so calling component can navigate
      return programId
    } catch (error) {
      console.error('Switch program failed:', error)
      throw error
    }
  }, [state.availablePrograms])

  // Auto-refresh timer
  useEffect(() => {
    if (!state.tokenExpiresAt) return

    // Clear existing timer
    if (refreshTimerRef.current) {
      clearTimeout(refreshTimerRef.current)
    }

    // Calculate time until 1 minute before expiry
    const refreshIn = state.tokenExpiresAt - Date.now() - 60000 // 1 min before expiry

    if (refreshIn > 0) {
      refreshTimerRef.current = setTimeout(() => {
        refreshAccessToken().catch(() => {
          // Refresh failed, will be logged out
        })
      }, refreshIn)
    } else {
      // Token already expired or about to - refresh immediately
      refreshAccessToken().catch(() => {
        // Refresh failed, will be logged out
      })
    }

    return () => {
      if (refreshTimerRef.current) {
        clearTimeout(refreshTimerRef.current)
      }
    }
  }, [state.tokenExpiresAt, refreshAccessToken])

  const hasOrgRole = useCallback((role: string): boolean => {
    if (!state.organization) return false
    const roleHierarchy: Record<string, number> = {
      member: 1,
      admin: 2,
      owner: 3,
    }
    const userLevel = roleHierarchy[state.organization.org_role] || 0
    const requiredLevel = roleHierarchy[role] || 0
    return userLevel >= requiredLevel
  }, [state.organization])

  const hasProgramRole = useCallback((role: string): boolean => {
    if (!state.currentProgram) return false
    const roleHierarchy: Record<string, number> = {
      viewer: 1,
      contributor: 2,
      admin: 3,
    }
    const userLevel = roleHierarchy[state.currentProgram.role] || 0
    const requiredLevel = roleHierarchy[role] || 0
    return userLevel >= requiredLevel
  }, [state.currentProgram])

  // Connect this context to the API client (after functions are defined)
  useEffect(() => {
    setAuthContext({
      accessToken: state.accessToken,
      refreshAccessToken,
      logout,
    })
  }, [state.accessToken, refreshAccessToken, logout])

  // Initial auth check on mount - only restore if no valid session exists
  useEffect(() => {
    const restoreSession = async () => {
      // Check if we already have a valid token
      if (state.accessToken && state.tokenExpiresAt && Date.now() < state.tokenExpiresAt) {
        // Already have valid token, don't need to refresh
        setState(prev => ({ ...prev, isLoading: false }))
        return
      }

      // No valid token, try to refresh from cookie
      try {
        await refreshAccessToken()
      } catch (error) {
        // No valid refresh token - user needs to login
        setState(prev => ({ ...prev, isLoading: false }))
      }
    }

    restoreSession()
  }, []) // Empty deps - only run once on mount

  const value: AuthContextValue = {
    ...state,
    login,
    logout,
    switchProgram,
    refreshAccessToken,
    hasOrgRole,
    hasProgramRole,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
