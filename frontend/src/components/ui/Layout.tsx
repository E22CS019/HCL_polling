import { Outlet, Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { BarChart2, PlusCircle, LayoutDashboard, LogOut, LogIn } from 'lucide-react'

export default function Layout() {
  const { user, clearAuth } = useAuth()
  const navigate = useNavigate()

  function handleLogout() {
    clearAuth()
    navigate('/')
  }

  return (
    <div className="min-h-screen flex flex-col">
      {/* ── Nav ── */}
      <header className="sticky top-0 z-50 glass border-b border-white/10">
        <nav className="max-w-6xl mx-auto px-4 h-16 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2 font-bold text-lg tracking-tight">
            <BarChart2 className="text-brand-400" size={22} />
            <span className="text-gradient">Pollster</span>
          </Link>

          <div className="flex items-center gap-2">
            {user ? (
              <>
                <Link
                  to="/create"
                  className="hidden sm:flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-brand-600 hover:bg-brand-500 text-sm font-medium transition-colors"
                >
                  <PlusCircle size={15} />
                  New Poll
                </Link>
                <Link
                  to="/dashboard"
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-white/10 text-sm transition-colors"
                >
                  <LayoutDashboard size={15} />
                  <span className="hidden sm:inline">Dashboard</span>
                </Link>
                <button
                  onClick={handleLogout}
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-white/10 text-sm text-slate-400 hover:text-white transition-colors"
                >
                  <LogOut size={15} />
                  <span className="hidden sm:inline">Logout</span>
                </button>
              </>
            ) : (
              <>
                <Link
                  to="/login"
                  className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg hover:bg-white/10 text-sm transition-colors"
                >
                  <LogIn size={15} />
                  Login
                </Link>
                <Link
                  to="/register"
                  className="px-3 py-1.5 rounded-lg bg-brand-600 hover:bg-brand-500 text-sm font-medium transition-colors"
                >
                  Sign up
                </Link>
              </>
            )}
          </div>
        </nav>
      </header>

      {/* ── Page Content ── */}
      <main className="flex-1">
        <Outlet />
      </main>

      {/* ── Footer ── */}
      <footer className="border-t border-white/5 py-6 text-center text-xs text-slate-600">
        Pollster &mdash; live polling, built with React + Go + Redis + MongoDB
      </footer>
    </div>
  )
}
