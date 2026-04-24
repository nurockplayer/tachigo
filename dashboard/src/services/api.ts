import axios, { type AxiosRequestConfig } from 'axios'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

const client = axios.create({
  baseURL: BASE_URL,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true,
})

let _accessToken: string | null = null

export function setAuthToken(token: string) {
  _accessToken = token
  client.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

export function clearAuthToken() {
  _accessToken = null
  delete client.defaults.headers.common['Authorization']
}

export function hasAuthToken(): boolean {
  return _accessToken !== null
}

interface RefreshResponse {
  data: { tokens: { access_token: string } }
}

// 正在進行中的 refresh promise，讓並發 401 只觸發一次刷新（token rotation 下必要）
let _refreshPromise: Promise<void> | null = null

client.interceptors.response.use(
  response => response,
  async (error) => {
    const originalRequest = error.config as AxiosRequestConfig & { _retry?: boolean }
    const isRefreshEndpoint = (originalRequest.url ?? '').includes('/api/v1/auth/refresh')

    if (error.response?.status !== 401 || originalRequest._retry || isRefreshEndpoint) {
      return Promise.reject(error)
    }

    originalRequest._retry = true

    if (!_refreshPromise) {
      _refreshPromise = client
        .post<RefreshResponse>('/api/v1/auth/refresh')
        .then(({ data }) => {
          setAuthToken(data.data.tokens.access_token)
        })
        .catch((refreshError) => {
          clearAuthToken()
          throw refreshError
        })
        .finally(() => {
          _refreshPromise = null
        })
    }

    try {
      await _refreshPromise
    } catch {
      return Promise.reject(error)
    }

    // axios 1.x 在 error.config 中已 flatten Authorization header，retry 前需明確覆寫
    // 否則 request-level header 優先於 defaults，新 token 不會生效
    if (originalRequest.headers) {
      originalRequest.headers['Authorization'] = client.defaults.headers.common['Authorization']
    }

    return client(originalRequest)
  },
)

export default client
