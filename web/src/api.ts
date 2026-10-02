// All backend calls go through /api. In dev, Vite proxies it to the Go server;
// in the cluster, the Ingress routes it to the Go service. Paths are unchanged
// in both cases.
const BASE = '/api'

// Full-page navigation target; the server redirects to Spotify.
export const LOGIN_URL = `${BASE}/auth/login`

export class HttpError extends Error {
  readonly status: number

  constructor(status: number, statusText: string) {
    super(`${status} ${statusText}`)
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, { credentials: 'include', ...init })
  if (!res.ok) {
    throw new HttpError(res.status, res.statusText)
  }
  return res.json() as Promise<T>
}

export interface Me {
  displayName: string
}

// Returns null when the user isn't logged in.
export async function getMe(): Promise<Me | null> {
  try {
    return await request<Me>('/me')
  } catch (e) {
    if (e instanceof HttpError && e.status === 401) return null
    throw e
  }
}
