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

  it('uses localhost only for non-production builds without explicit API env', async () => {
    vi.stubEnv('DEV', true)
    vi.stubEnv('PROD', false)

    const client = await loadApiClient()

    expect(client.defaults.baseURL).toBe('http://localhost:8080')
  })

  it('fails closed in production builds without explicit API env', async () => {
    vi.stubEnv('DEV', false)
    vi.stubEnv('PROD', true)

    await expect(loadApiClient()).rejects.toThrow('VITE_TACHIGO_API_URL is required for production dashboard builds')
  })
})
