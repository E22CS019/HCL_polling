import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { toast } from 'react-hot-toast'
import { PlusCircle, BarChart2, Lock, ExternalLink } from 'lucide-react'
import { getMyPolls, closePoll } from '../api/polls'
import type { Poll } from '../api/types'
import { useAuth } from '../context/AuthContext'
import Spinner from '../components/ui/Spinner'
import Button from '../components/ui/Button'
import { timeAgo } from '../utils/format'

export default function DashboardPage() {
  const { user }           = useAuth()
  const [polls, setPolls]   = useState<Poll[]>([])
  const [loading, setLoading] = useState(true)
  const [closingId, setClosingId] = useState<string | null>(null)

  useEffect(() => {
    getMyPolls()
      .then(setPolls)
      .catch(() => toast.error('Failed to load your polls.'))
      .finally(() => setLoading(false))
  }, [])

  async function handleClose(id: string) {
    setClosingId(id)
    try {
      await closePoll(id)
      setPolls(prev => prev.map(p => p.id === id ? { ...p, closed: true } : p))
      toast.success('Poll closed.')
    } catch {
      toast.error('Failed to close poll.')
    } finally {
      setClosingId(null)
    }
  }

  return (
    <div className="max-w-4xl mx-auto px-4 py-12">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold">My Polls</h1>
          <p className="text-slate-400 text-sm mt-1">
            {user?.username ? `Welcome back, ${user.username}` : 'Manage your polls'}
          </p>
        </div>
        <Link
          to="/create"
          className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-brand-600 hover:bg-brand-500 font-medium text-sm transition-colors"
        >
          <PlusCircle size={15} />
          New Poll
        </Link>
      </div>

      {loading ? (
        <div className="flex justify-center py-16">
          <Spinner size="lg" />
        </div>
      ) : polls.length === 0 ? (
        <div className="text-center py-16 card">
          <BarChart2 size={40} className="mx-auto mb-3 text-slate-600" />
          <p className="text-slate-400 mb-4">You haven't created any polls yet.</p>
          <Link
            to="/create"
            className="inline-flex items-center gap-1.5 px-4 py-2 rounded-xl bg-brand-600 hover:bg-brand-500 font-medium text-sm transition-colors"
          >
            <PlusCircle size={15} />
            Create your first poll
          </Link>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {polls.map(poll => {
            const isExpired = poll.ends_at ? new Date(poll.ends_at) < new Date() : false
            const isClosed  = poll.closed || isExpired

            return (
              <div key={poll.id} className="card hover:border-white/20 transition-all">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-semibold text-white truncate">{poll.title}</h3>
                      {isClosed && (
                        <span className="inline-flex items-center gap-1 text-xs text-slate-500 bg-white/5 px-2 py-0.5 rounded-full border border-white/10">
                          <Lock size={10} /> Closed
                        </span>
                      )}
                    </div>
                    <div className="flex flex-wrap items-center gap-3 mt-2 text-xs text-slate-500">
                      <span>{poll.total_votes.toLocaleString()} votes</span>
                      <span>{poll.options.length} options</span>
                      <span>Created {timeAgo(poll.created_at)}</span>
                      {poll.multi_choice && <span className="text-brand-400">Multi-choice</span>}
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <Link
                      to={`/poll/${poll.id}`}
                      className="p-2 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
                      title="View poll"
                    >
                      <ExternalLink size={15} />
                    </Link>
                    {!isClosed && (
                      <Button
                        variant="ghost"
                        size="sm"
                        loading={closingId === poll.id}
                        onClick={() => handleClose(poll.id)}
                        className="text-red-400 hover:text-red-300 hover:bg-red-500/10"
                      >
                        <Lock size={13} />
                        Close
                      </Button>
                    )}
                  </div>
                </div>

                {/* Mini results preview */}
                {poll.total_votes > 0 && (
                  <div className="mt-4 grid grid-cols-1 gap-1.5">
                    {poll.options.slice(0, 4).map(opt => {
                      const pct = poll.total_votes > 0 ? (opt.votes / poll.total_votes) * 100 : 0
                      return (
                        <div key={opt.id}>
                          <div className="flex justify-between text-xs text-slate-500 mb-1">
                            <span className="truncate max-w-[70%]">{opt.text}</span>
                            <span>{pct.toFixed(1)}%</span>
                          </div>
                          <div className="h-1.5 rounded-full bg-white/5 overflow-hidden">
                            <div
                              className="h-full rounded-full bg-brand-500/70 vote-bar"
                              style={{ width: `${pct}%` }}
                            />
                          </div>
                        </div>
                      )
                    })}
                    {poll.options.length > 4 && (
                      <p className="text-xs text-slate-600 mt-1">+{poll.options.length - 4} more options</p>
                    )}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}
