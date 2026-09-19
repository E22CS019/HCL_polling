import axios from 'axios'

// In production (Vercel) VITE_API_URL is set to the Render backend URL.
// In local dev it is empty and Vite's proxy forwards /api to localhost:8080.
const BASE = import.meta.env.VITE_API_URL
  ? `${import.meta.env.VITE_API_URL}/api/v1`
  : '/api/v1'

const client = axios.create({
  baseURL: BASE,
  headers: { 'Content-Type': 'application/json' },
  timeout: 15_000,
})

// Attach JWT from localStorage on every request.
client.interceptors.request.use(config => {
  const token = localStorage.getItem('pollster_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// On 401, clear the stale token and let the UI handle the redirect.
client.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('pollster_token')
      localStorage.removeItem('pollster_user')
    }
    return Promise.reject(err)
  },
)

export default client
