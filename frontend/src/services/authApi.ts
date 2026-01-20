import axios from 'axios'
import type { LoginRequest, LoginResponse, RefreshResponse, SwitchProgramRequest, SwitchProgramResponse } from '../types/auth'

const API_URL = (import.meta as any).env?.VITE_API_URL || 'http://localhost:8080/api/v1'

// Separate axios instance for auth endpoints (no interceptors to avoid circular dependencies)
const authApi = axios.create({
  baseURL: API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // Important: Send cookies with requests
})

export const authService = {
  login: async (data: LoginRequest): Promise<LoginResponse> => {
    const response = await authApi.post('/auth/login', data)
    return response.data.data
  },

  refresh: async (): Promise<RefreshResponse> => {
    const response = await authApi.post('/auth/refresh')
    return response.data.data
  },

  logout: async (): Promise<void> => {
    await authApi.post('/auth/logout')
  },

  switchProgram: async (data: SwitchProgramRequest): Promise<SwitchProgramResponse> => {
    const response = await authApi.post('/auth/switch-program', data)
    return response.data.data
  },
}

export default authApi
