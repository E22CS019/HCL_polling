import { useState, type FormEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { toast } from 'react-hot-toast'
import { register } from '../api/auth'
import { useAuth } from '../context/AuthContext'
import Input from '../components/ui/Input'
import Button from '../components/ui/Button'
import { BarChart2 } from 'lucide-react'

export default function RegisterPage() {
  const navigate    = useNavigate()
  const { setAuth } = useAuth()

  const [username, setUsername] = useState('')
  const [email,    setEmail]    = useState('')
  const [password, setPassword] = useState('')
  const [loading,  setLoading]  = useState(false)
  const [error,    setError]    = useState('')

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError('')

    if (password.length < 8) {
      setError('Password must be at least 8 characters.')
      return
    }

    setLoading(true)
    try {
      const res = await register({ username, email, password })
      setAuth(res.token, res.user)
      toast.success(`Account created! Welcome, ${res.user.username}!`)
      navigate('/')
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: string } } })
        ?.response?.data?.error ?? 'Registration failed. Please try again.'
      setError(msg)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-[calc(100vh-8rem)] items-center justify-center px-4">
      <div className="w-full max-w-md">
        <div className="text-center mb-8">
          <BarChart2 size={36} className="mx-auto text-brand-400 mb-3" />
          <h1 className="text-2xl font-bold">Create your account</h1>
          <p className="text-slate-400 text-sm mt-1">Start creating polls in seconds</p>
        </div>

        <form onSubmit={handleSubmit} className="card flex flex-col gap-4">
          {error && (
            <div className="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-4 py-2.5">
              {error}
            </div>
          )}

          <Input
            label="Username"
            type="text"
            value={username}
            onChange={e => setUsername(e.target.value)}
            placeholder="cooluser"
            minLength={3}
            maxLength={30}
            autoComplete="username"
            required
          />
          <Input
            label="Email"
            type="email"
            value={email}
            onChange={e => setEmail(e.target.value)}
            placeholder="you@example.com"
            autoComplete="email"
            required
          />
          <Input
            label="Password"
            type="password"
            value={password}
            onChange={e => setPassword(e.target.value)}
            placeholder="Min. 8 characters"
            autoComplete="new-password"
            required
          />

          <Button type="submit" loading={loading} size="lg" className="w-full mt-1">
            Create account
          </Button>

          <p className="text-center text-sm text-slate-500">
            Already have an account?{' '}
            <Link to="/login" className="text-brand-400 hover:text-brand-300 transition-colors">
              Log in
            </Link>
          </p>
        </form>
      </div>
    </div>
  )
}
