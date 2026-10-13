import { describe, it, expect, beforeEach, vi } from 'vitest'

type Client = typeof import('./client')

const user = { id: 1, email: 'owner@acme.test', companyId: 10, companyName: 'Acme', role: 'owner' as const }

function json(status: number, body?: unknown, headers: Record<string, string> = {}): Response {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json', ...headers },
  })
}

function header(init: RequestInit | undefined, name: string): string | undefined {
  return (init?.headers as Record<string, string> | undefined)?.[name]
}

let client: Client
let fetchMock: ReturnType<typeof vi.fn>

// The client keeps module state (refresh in flight, handlers): load a fresh
// copy for every test.
beforeEach(async () => {
  vi.resetModules()
  sessionStorage.clear()
  fetchMock = vi.fn()
  vi.stubGlobal('fetch', fetchMock)
  client = await import('./client')
  client.navigation.currentPath = () => '/events'
  client.navigation.toLogin = vi.fn()
})

describe('API client', () => {
  it('sends the access token and cookies with authenticated requests', async () => {
    client.storeToken('access-1')
    fetchMock.mockResolvedValueOnce(json(200, user))

    await expect(client.getCurrentUser()).resolves.toEqual(user)

    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe('http://localhost:8080/api/auth/me')
    expect(header(init, 'Authorization')).toBe('Bearer access-1')
    expect(init.credentials).toBe('include')
  })

  it('never sends the access token to self-authenticated auth endpoints', async () => {
    client.storeToken('access-1')
    fetchMock.mockResolvedValueOnce(json(200, { token: 't', user }))

    await client.login('owner@acme.test', 'secret')

    expect(header(fetchMock.mock.calls[0][1], 'Authorization')).toBeUndefined()
  })

  it('treats 204 No Content as success', async () => {
    client.storeToken('access-1')
    fetchMock.mockResolvedValueOnce(new Response(null, { status: 204 }))

    await expect(client.deleteAddress(3)).resolves.toBeUndefined()
  })

  it('refreshes silently once on 401 and retries with the new token', async () => {
    client.storeToken('expired')
    const refreshed = vi.fn()
    client.setSessionRefreshedHandler(refreshed)
    fetchMock
      .mockResolvedValueOnce(json(401, { error: 'missing or invalid token' }))
      .mockResolvedValueOnce(json(200, { token: 'access-2', user }))
      .mockResolvedValueOnce(json(200, [{ id: 1, value: '10.0.0.1', createdAt: '' }]))

    const addresses = await client.getAddresses()

    expect(addresses).toHaveLength(1)
    expect(fetchMock.mock.calls[1][0]).toBe('http://localhost:8080/api/auth/refresh')
    expect(fetchMock.mock.calls[1][1].method).toBe('POST')
    expect(header(fetchMock.mock.calls[2][1], 'Authorization')).toBe('Bearer access-2')
    expect(client.getToken()).toBe('access-2')
    expect(refreshed).toHaveBeenCalledWith({ token: 'access-2', user })
  })

  it('clears the session and redirects to /login when the refresh fails', async () => {
    client.storeToken('expired')
    const unauthorized = vi.fn()
    client.setUnauthorizedHandler(unauthorized)
    fetchMock
      .mockResolvedValueOnce(json(401, { error: 'missing or invalid token' }))
      .mockResolvedValueOnce(json(401, { error: 'invalid or expired session' }))

    await expect(client.getAddresses()).rejects.toThrow('Session expirée')

    expect(fetchMock).toHaveBeenCalledTimes(2)
    expect(unauthorized).toHaveBeenCalledOnce()
    expect(client.getToken()).toBeNull()
    expect(client.navigation.toLogin).toHaveBeenCalledOnce()
  })

  it('does not redirect when the user is already on a public page', async () => {
    client.navigation.currentPath = () => '/reset-password/abc'
    fetchMock.mockImplementation(() => Promise.resolve(json(401, {})))

    await expect(client.getAddresses()).rejects.toThrow()
    expect(client.navigation.toLogin).not.toHaveBeenCalled()
  })

  it('shares one refresh between concurrent 401 responses', async () => {
    client.storeToken('expired')
    let releaseRefresh: (r: Response) => void = () => {}
    fetchMock.mockImplementation((url: string, init: RequestInit) => {
      if (url.endsWith('/api/auth/refresh')) {
        return new Promise<Response>(resolve => { releaseRefresh = resolve })
      }
      return Promise.resolve(header(init, 'Authorization') === 'Bearer fresh' ? json(200, []) : json(401, {}))
    })

    const pending = Promise.all([client.getAddresses(), client.getAlerts(), client.getTeam()])
    await vi.waitFor(() => expect(fetchMock.mock.calls.filter(c => c[0].endsWith('/refresh'))).toHaveLength(1))
    releaseRefresh(json(200, { token: 'fresh', user }))

    await expect(pending).resolves.toEqual([[], [], []])
    expect(fetchMock.mock.calls.filter(c => c[0].endsWith('/refresh'))).toHaveLength(1)
  })

  it('returns the server message for a failed login without refreshing', async () => {
    fetchMock.mockResolvedValueOnce(json(401, { error: 'invalid credentials' }))

    await expect(client.login('a@b.co', 'wrong')).rejects.toThrow('invalid credentials')
    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(client.navigation.toLogin).not.toHaveBeenCalled()
  })

  it('explains rate limiting with the Retry-After delay', async () => {
    fetchMock.mockResolvedValueOnce(json(429, { error: 'too many requests' }, { 'Retry-After': '42' }))

    await expect(client.forgotPassword('a@b.co')).rejects.toThrow('42 s')
  })
})
