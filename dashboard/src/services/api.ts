import axios from 'axios'

const BASE_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

const client = axios.create({
  baseURL: BASE_URL,
  headers: { 'Content-Type': 'application/json' },
})

export function setAuthToken(token: string) {
  client.defaults.headers.common['Authorization'] = `Bearer ${token}`
}

export function clearAuthToken() {
  delete client.defaults.headers.common['Authorization']
}

export default client
