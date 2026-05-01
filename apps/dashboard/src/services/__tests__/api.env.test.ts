import { afterEach, describe, expect, it, vi } from 'vitest'

async function loadApiClient() {
  vi.resetModules()
  const module = await import('@/services/api')
  return module.default
}

afterEach(() => {
  vi.unstubAllEnvs()
})

describe('api client base URL env', () => {
  it('uses VITE_TACHIGO_API_URL as the canonical API base URL', async () => {
    vi.stubEnv('VITE_TACHIGO_API_URL', 'https://canonical.example.test')
    vi.stubEnv('VITE_API_URL', 'https://legacy.example.test')

    const client = await loadApiClient()

    expect(client.defaults.baseURL).toBe('https://canonical.example.test')
  })

  it('falls back to VITE_API_URL during the env key migration', async () => {
    vi.stubEnv('VITE_API_URL', 'https://legacy.example.test')

    const client = await loadApiClient()

    expect(client.defaults.baseURL).toBe('https://legacy.example.test')
  })
})
