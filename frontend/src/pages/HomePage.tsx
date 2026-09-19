import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { PlusCircle, Zap, BarChart2, Users } from 'lucide-react'
import { listRecent } from '../api/polls'
import type { Poll } from '../api/types'
import PollCard from '../components/ui/PollCard'
import Spinner from '../components/ui/Spinner'
import { useAuth } from '../context/AuthContext'

export default function HomePage() {
  const { user } = useAuth()
  const [polls, setPolls]   = useState<Poll[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    listRecent()
      .then(setPolls)
      .catch(() => setPolls([]))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div className="max-w-6xl mx-auto px-4 py-12">
      {/* Hero */}
      <section className="text-center mb-16">
        <div className="inline-flex items-center gap-2 text-xs font-medium text-brand-400 bg-brand-500/10 border border-brand-500/20 px-3 py-1 rounded-full mb-6">
          <Zap size={12} />
          Real-time polling, powered by Redis + SSE
        </div>
        <h1 className="text-5xl md:text-6xl font-extrabold mb-4 leading-tight">
          Create polls.<br />
          <span className="text-gradient">Watch results live.</span>
        </h1>
        <p className="text-slate-400 text-lg max-w-xl mx-auto mb-8">
          Share a link, your audience votes, everyone sees the results update instantly — no refresh needed.
        </p>
        <div className="flex items-center justify-center gap-3 flex-wrap">
          {user ? (
            <Link
              to="/create"
              className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-brand-600 hover:bg-brand-500 font-semibold transition-all text-base"
            >
              <PlusCircle size={18} />
              Create a Poll
            </Link>
          ) : (
            <>
              <Link
                to="/register"
                className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-brand-600 hover:bg-brand-500 font-semibold transition-all text-base"
              >
                Get started — it's free
              </Link>
              <Link
                to="/login"
                className="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-white/10 hover:bg-white/15 font-medium transition-all text-base"
              >
                Log in
              </Link>
            </>
          )}
        </div>
      </section>

      {/* Feature pills */}
      <div className="flex flex-wrap justify-center gap-3 mb-16">
        {[
          { icon: <Zap size={14} />,        text: 'Sub-second live updates via SSE' },
          { icon: <BarChart2 size={14} />,  text: 'Redis-powered vote counters' },
          { icon: <Users size={14} />,      text: 'Multi-choice polls supported' },
        ].map(f => (
          <div key={f.text} className="flex items-center gap-2 text-xs text-slate-400 glass px-3 py-2 rounded-full">
            <span className="text-brand-400">{f.icon}</span>
            {f.text}
          </div>
        ))}
      </div>

      {/* Recent polls */}
      <section>
        <h2 className="text-xl font-bold mb-6 flex items-center gap-2">
          <BarChart2 size={18} className="text-brand-400" />
          Recent Polls
        </h2>
        {loading ? (
          <div className="flex justify-center py-16">
            <Spinner size="lg" />
          </div>
        ) : polls.length === 0 ? (
          <div className="text-center py-16 text-slate-500">
            <BarChart2 size={40} className="mx-auto mb-3 opacity-30" />
            <p>No polls yet. Be the first to create one!</p>
          </div>
        ) : (
          <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {polls.map(p => <PollCard key={p.id} poll={p} />)}
          </div>
        )}
      </section>
    </div>
  )
}
