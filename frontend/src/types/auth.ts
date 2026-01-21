export interface LoginRequest {
  email: string
  password: string
}

export interface UserInfo {
  user_id: string
  email: string
  full_name: string
  is_admin: boolean
}

export interface OrganizationInfo {
  organization_id: string
  organization_name: string
  organization_code: string
  org_role: 'owner' | 'admin' | 'member'
}

export interface ProgramAccess {
  program_id: string
  program_name: string
  program_code: string
  role: 'admin' | 'contributor' | 'viewer'
  granted_at: string
}

export interface TokenPair {
  access_token: string
  refresh_token?: string
  expires_in: number
}

export interface LoginResponse {
  user: UserInfo
  organization: OrganizationInfo
  current_program: ProgramAccess | null
  tokens: TokenPair | null
  programs: ProgramAccess[]
}

export interface RefreshResponse {
  user: UserInfo
  organization: OrganizationInfo
  current_program: ProgramAccess | null
  programs: ProgramAccess[]
  access_token: string
  expires_in: number
}

export interface SwitchProgramRequest {
  program_id: string
}

export interface SwitchProgramResponse {
  access_token: string
  expires_in: number
}
