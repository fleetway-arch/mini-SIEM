import type {
  SiemEvent, Alert, AlertStatus,
  StatsSummary, TimelinePoint, TopIp, TimeRange,
  User, LoginResponse, Address,
  SignupPayload, TeamMember, Role, MemberStatus,
  InvitationDetails, BulkInvitationResult,
} from '../types'

const BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
const REQUEST_TIMEOUT_MS = 10000

const TOKEN_KEY = 'siem_token'

// Endpoints that authenticate by themselves: a 401 there is a business error
// (wrong password, invalid link), never a reason to refresh or redirect.
const SELF_AUTHENTICATED_PATHS = [
  '/api/auth/login', '/api/auth/signup', '/api/auth/refresh', '/api/auth/logout',
  '/api/auth/forgot-password', '/api/auth/reset-password', '/api/invitations/',
]
const PUBLIC_PAGES = ['/login', '/signup', '/invite/', '/forgot-password', '/reset-password/']

let onUnauthorized: (() => void) | null = null
let onSessionRefreshed: ((session: LoginResponse) => void) | null = null

export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

export function setSessionRefreshedHandler(fn: (session: LoginResponse) => void) {
  onSessionRefreshed = fn
}

export function getToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY)
}

export function storeToken(token: string | null) {
  if (token) sessionStorage.setItem(TOKEN_KEY, token)
  else sessionStorage.removeItem(TOKEN_KEY)
}

async function request(path: string, options?: RequestInit): Promise<Response> {
  const token = getToken()
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  if (token && !SELF_AUTHENTICATED_PATHS.some(p => path.startsWith(p))) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS)
  try {
    // credentials: 'include' lets the browser send the httpOnly refresh cookie.
    return await fetch(`${BASE_URL}${path}`, {
      ...options, headers, credentials: 'include', signal: controller.signal,
    })
  } catch (error) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new Error('La requête a expiré. Vérifiez que le serveur et MongoDB sont démarrés.')
    }
    throw error
  } finally {
    clearTimeout(timeout)
  }
}

async function parseBody<T>(res: Response): Promise<T> {
  if (res.status === 204) return undefined as T
  const text = await res.text()
  return (text ? JSON.parse(text) : undefined) as T
}

async function errorFrom(res: Response): Promise<Error> {
  const body = await res.json().catch(() => null)
  if (res.status === 429) {
    const retry = res.headers.get('Retry-After')
    return new Error(`Trop de tentatives. Réessayez${retry ? ` dans ${retry} s` : ' plus tard'}.`)
  }
  return new Error(body?.error || body?.message || `API error ${res.status}`)
}

let refreshInFlight: Promise<boolean> | null = null

// Rotates the refresh cookie and stores the new access token. Concurrent
// callers share one request so the cookie is only rotated once.
export function refreshSession(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = (async () => {
      try {
        const res = await request('/api/auth/refresh', { method: 'POST' })
        if (!res.ok) return false
        const session = await parseBody<LoginResponse>(res)
        storeToken(session.token)
        onSessionRefreshed?.(session)
        return true
      } catch {
        return false
      } finally {
        refreshInFlight = null
      }
    })()
  }
  return refreshInFlight
}

function handleSessionLost() {
  storeToken(null)
  onUnauthorized?.()
  if (!PUBLIC_PAGES.some(p => window.location.pathname.startsWith(p))) {
    window.location.assign('/login')
  }
}

async function http<T>(path: string, options?: RequestInit): Promise<T> {
  let res = await request(path, options)
  const selfAuthenticated = SELF_AUTHENTICATED_PATHS.some(p => path.startsWith(p))

  if (res.status === 401 && !selfAuthenticated) {
    if (await refreshSession()) {
      res = await request(path, options)
    }
    if (res.status === 401) {
      handleSessionLost()
      throw new Error('Session expirée')
    }
  }
  if (!res.ok) throw await errorFrom(res)
  return parseBody<T>(res)
}

function withAddress(query: string, address?: string): string {
  if (!address) return query
  const sep = query.includes('?') && query.length > query.indexOf('?') + 1 ? '&' : query.includes('?') ? '' : '?'
  return `${query}${sep}address=${encodeURIComponent(address)}`
}

// --- Auth ---
export async function login(email: string, password: string): Promise<LoginResponse> {
  return http<LoginResponse>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export async function logout(): Promise<void> {
  await http<void>('/api/auth/logout', { method: 'POST' })
}

export async function getCurrentUser(): Promise<User> {
  return http<User>('/api/auth/me')
}

export async function signup(payload: SignupPayload): Promise<LoginResponse> {
  return http<LoginResponse>('/api/auth/signup', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

// --- Team ---
export async function getTeam(): Promise<TeamMember[]> {
  return http<TeamMember[]>('/api/team')
}

export async function createInvitations(emails: string[], role: Role): Promise<BulkInvitationResult> {
  return http<BulkInvitationResult>('/api/team/invitations', {
    method: 'POST',
    body: JSON.stringify({ emails, role }),
  })
}

export async function updateMember(id: number, data: { role?: Role; status?: MemberStatus }): Promise<TeamMember> {
  return http<TeamMember>(`/api/team/${id}`, {
    method: 'PATCH',
    body: JSON.stringify(data),
  })
}

export async function resendInvitation(id: number): Promise<TeamMember> {
  return http<TeamMember>(`/api/team/${id}/resend`, { method: 'POST' })
}

export async function deleteMember(id: number): Promise<void> {
  await http<void>(`/api/team/${id}`, { method: 'DELETE' })
}

// --- Invitations ---
export async function getInvitation(token: string): Promise<InvitationDetails> {
  return http<InvitationDetails>(`/api/invitations/${token}`)
}

export async function acceptInvitation(token: string, fullName: string, password: string): Promise<LoginResponse> {
  return http<LoginResponse>(`/api/invitations/${token}/accept`, {
    method: 'POST',
    body: JSON.stringify({ fullName, password }),
  })
}

// --- Stats ---
export async function getStatsSummary(address?: string): Promise<StatsSummary> {
  return http<StatsSummary>(withAddress('/api/stats/summary', address))
}

export async function getTimeline(range: TimeRange, address?: string): Promise<TimelinePoint[]> {
  return http<TimelinePoint[]>(withAddress(`/api/stats/timeline?range=${range}`, address))
}

export async function getTopIps(limit = 10, address?: string): Promise<TopIp[]> {
  return http<TopIp[]>(withAddress(`/api/stats/top-ips?limit=${limit}`, address))
}

// --- Events ---
export async function getEvents(params: {
  page: number; limit: number; source?: string; severity?: string;
  ip?: string; q?: string; from?: string; to?: string; address?: string;
}): Promise<{ items: SiemEvent[]; total: number }> {
  const query = new URLSearchParams()
  query.set('page', String(params.page))
  query.set('limit', String(params.limit))
  if (params.source) query.set('source', params.source)
  if (params.severity) query.set('severity', params.severity)
  if (params.ip) query.set('ip', params.ip)
  if (params.q) query.set('q', params.q)
  if (params.from) query.set('from', params.from)
  if (params.to) query.set('to', params.to)
  if (params.address) query.set('address', params.address)

  return http<{ items: SiemEvent[]; total: number }>(`/api/events?${query.toString()}`)
}

// --- Alerts ---
export async function getAlerts(status?: AlertStatus | '', address?: string): Promise<Alert[]> {
  let query = ''
  if (status) query = `?status=${status}`
  query = withAddress(query, address)
  return http<Alert[]>(`/api/alerts${query}`)
}

export async function patchAlert(id: number, status: AlertStatus): Promise<Alert> {
  return http<Alert>(`/api/alerts/${id}`, {
    method: 'PATCH',
    body: JSON.stringify({ status }),
  })
}

// --- Addresses ---
export async function getAddresses(): Promise<Address[]> {
  return http<Address[]>('/api/addresses')
}

export async function createAddress(value: string, label?: string): Promise<Address> {
  return http<Address>('/api/addresses', {
    method: 'POST',
    body: JSON.stringify({ value, label }),
  })
}

export async function deleteAddress(id: number): Promise<void> {
  await http<void>(`/api/addresses/${id}`, { method: 'DELETE' })
}
