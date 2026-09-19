import { createContext, useContext, useState, useCallback, type ReactNode } from 'react'
import type { User } from '../api/types'

interface AuthContextValue {
  token: string | null
  user: User | null
  setAuth: (token: string, user: User) => void
  clearAuth: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(
    () => localStorage.getItem('pollster_token'),
  )
  const [user, setUser] = useState<User | null>(() => {
    try {
      const raw = localStorage.getItem('pollster_user')
      return raw ? (JSON.parse(raw) as User) : null
    } catch {
      return null
    }
  })

  const setAuth = useCallback((t: string, u: User) => {
    localStorage.setItem('pollster_token', t)
    localStorage.setItem('pollster_user', JSON.stringify(u))
    setToken(t)
    setUser(u)
  }, [])

  const clearAuth = useCallback(() => {
    localStorage.removeItem('pollster_token')
    localStorage.removeItem('pollster_user')
    setToken(null)
    setUser(null)
  }, [])

  return (
    <AuthContext.Provider value={{ token, user, setAuth, clearAuth }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used inside AuthProvider')
  return ctx
}
