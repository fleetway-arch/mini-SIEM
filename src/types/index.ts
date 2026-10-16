export type Source = 'sshd' | 'apache' | 'dns' | 'vsftpd'
export type Severity = 'info' | 'warning' | 'critical'
export type AlertStatus = 'open' | 'acknowledged' | 'resolved'
export type TimeRange = '1h' | '24h' | '7d'
export type Role = 'owner' | 'admin' | 'viewer'
export type MemberStatus = 'invited' | 'active' | 'revoked'

export interface SiemEvent {
  id: number
  timestamp: string
  source: Source
  severity: Severity
  ip: string
  message: string
  rawLog: string
}

export interface Alert {
  id: number
  createdAt: string
  rule: string
  severity: Severity
  ip: string
  description: string
  eventCount: number
  status: AlertStatus
  lastSeenAt?: string
}

export interface StatsSummary {
  totalEvents: number
  failedLogins: number
  uniqueIps: number
  openAlerts: number
}

export interface TimelinePoint {
  time: string
  total: number
  failed: number
}

export interface TopIp {
  ip: string
  count: number
  failedCount: number
}

export interface User {
  id: number
  email: string
  fullName?: string
  companyId: number
  companyName: string
  role: Role
}

export interface LoginResponse {
  token: string
  expiresIn?: number
  user: User
}

export interface IngestKey {
  id: number
  name: string
  prefix: string
  createdAt: string
  lastUsedAt?: string
}

// Retournée une seule fois à la création : seule l'empreinte est stockée
export interface CreatedIngestKey extends IngestKey {
  key: string
}

export interface Address {
  id: number
  value: string
  label?: string
  createdAt: string
}

export interface EventsFilter {
  page: number
  limit: number
  source?: Source | ''
  severity?: Severity | ''
  ip?: string
  q?: string
  from?: string
  to?: string
  address?: string
}

export interface SignupPayload {
  company: {
    name: string
    industry: string
    size: string
    country: string
    website?: string
    securityContact: string
  }
  admin: {
    fullName: string
    email: string
    password: string
  }
}

export interface TeamMember {
  id: number
  email: string
  fullName?: string
  role: Role
  status: MemberStatus
  invitedAt: string
  lastLoginAt?: string
  inviteToken?: string
}

export interface InvitationDetails {
  companyName: string
  email: string
  role: Role
}

export interface BulkInvitationResult {
  created: TeamMember[]
  skipped: { email: string; reason: string }[]
}
