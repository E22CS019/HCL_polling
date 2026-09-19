import axios from 'axios'

const client = axios.create({
  baseURL: '/api/v1',
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
