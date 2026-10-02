// All backend calls go through /api. In dev, Vite proxies it to the Go server;
// in the cluster, the Ingress routes it to the Go service. Paths are unchanged
// in both cases.
const BASE = '/api'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, { credentials: 'include', ...init })
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export function getHello(): Promise<{ message: string }> {
  return request('/hello')
}
