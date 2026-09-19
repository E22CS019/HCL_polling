import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { toast } from 'react-hot-toast'
import { Share2, Copy, Users, Clock, CheckCircle2, Lock } from 'lucide-react'
import { getPoll, vote, getVoteStatus, closePoll } from '../api/polls'
import type { Poll, VoteStatusResponse } from '../api/types'
import { useLiveResults } from '../hooks/useLiveResults'
import { useAuth } from '../context/AuthContext'
import ResultBar from '../components/ui/ResultBar'
import Button from '../components/ui/Button'
import Spinner from '../components/ui/Spinner'
import LiveBadge from '../components/ui/LiveBadge'
import { timeAgo, formatDate } from '../utils/format'

export default function PollPage() {
  const { id } = useParams<{ id: string }>()
  const { user } = useAuth()

  const [poll,         setPoll]         = useState<Poll | null>(null)
  const [loading,      setLoading]      = useState(true)
  const [voteStatus,   setVoteStatus]   = useState<VoteStatusResponse | null>(null)
  const [selected,     setSelected]     = useState<Set<string>>(new Set())
  const [submitting,   setSubmitting]   = useState(false)
  const [showResults,  setShowResults]  = useState(false)
  const [closing,      setClosing]      = useState(false)

  const { result, status } = useLiveResults(id ?? '')

  // ── Load poll + vote status ──────────────────────────────────────────────
  useEffect(() => {
    if (!id) return
    setLoading(true)

    Promise.all([
      getPoll(id),
      getVoteStatus(id),
    ]).then(([p, vs]) => {
      setPoll(p)
      setVoteStatus(vs)
      if (vs.voted) {
        setSelected(new Set(vs.option_ids))
        setShowResults(true)
      }
    }).catch(() => {
      toast.error('Failed to load poll.')
    }).finally(() => setLoading(false))
  }, [id])

  // ── Voting ───────────────────────────────────────────────────────────────
  function toggleOption(optId: string) {
    if (!poll || voteStatus?.voted) return
    setSelected(prev => {
      const next = new Set(prev)
      if (poll.multi_choice) {
        next.has(optId) ? next.delete(optId) : next.add(optId)
      } else {
        return new Set([optId])
      }
      return next
    })
  }

  async function handleVote() {
    if (!id || selected.size === 0) return
    setSubmitting(true)
    try {
      await vote(id, Array.from(selected))
      setVoteStatus({ voted: true, option_ids: Array.from(selected) })
      setShowResults(true)
      toast.success('Vote recorded!')
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response?.status
      if (status === 409) {
        toast.error('You have already voted on this poll.')
        setVoteStatus({ voted: true, option_ids: Array.from(selected) })
        setShowResults(true)
      } else if (status === 410) {
        toast.error('This poll is closed.')
      } else {
        toast.error('Vote failed. Please try again.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  // ── Close poll ──────────────────────────────────────────────────────────
  async function handleClose() {
    if (!id) return
    setClosing(true)
    try {
      await closePoll(id)
      setPoll(prev => prev ? { ...prev, closed: true } : prev)
      toast.success('Poll closed.')
    } catch {
      toast.error('Failed to close poll.')
    } finally {
      setClosing(false)
    }
  }

  // ── Share ────────────────────────────────────────────────────────────────
  function copyLink() {
    navigator.clipboard.writeText(window.location.href)
    toast.success('Link copied to clipboard!')
  }

  async function shareNative() {
    if (navigator.share) {
      await navigator.share({ title: poll?.title, url: window.location.href })
    } else {
      copyLink()
    }
  }

  // ── Helpers ──────────────────────────────────────────────────────────────
  const isExpired  = poll?.ends_at ? new Date(poll.ends_at) < new Date() : false
  const isClosed   = poll?.closed || isExpired
  const isCreator  = user && poll && user.id === poll.creator_id
  const totalVotes = result?.total_votes ?? poll?.total_votes ?? 0

  const winnerVotes = result
    ? Math.max(...result.options.map(o => o.votes))
    : 0

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!poll) {
    return (
      <div className="text-center py-24">
        <p className="text-slate-400 text-lg mb-4">Poll not found.</p>
        <Link to="/" className="text-brand-400 hover:underline text-sm">← Back home</Link>
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto px-4 py-12">
      {/* Header */}
      <div className="mb-8">
        <div className="flex items-start justify-between gap-4 mb-3">
          <h1 className="text-2xl md:text-3xl font-bold leading-snug">{poll.title}</h1>
          <div className="flex items-center gap-2 shrink-0">
            {isClosed ? (
              <span className="inline-flex items-center gap-1.5 text-xs text-slate-500 bg-white/5 px-2.5 py-1 rounded-full border border-white/10">
                <Lock size={11} /> Closed
              </span>
            ) : (
              <LiveBadge status={status} />
            )}
          </div>
        </div>

        {poll.description && (
          <p className="text-slate-400 text-sm mb-4">{poll.description}</p>
        )}

        <div className="flex flex-wrap items-center gap-4 text-xs text-slate-500">
          <span className="flex items-center gap-1">
            <Users size={12} />
            {totalVotes.toLocaleString()} vote{totalVotes !== 1 ? 's' : ''}
          </span>
          <span className="flex items-center gap-1">
            <Clock size={12} />
            Created {timeAgo(poll.created_at)}
          </span>
          {poll.ends_at && (
            <span className="flex items-center gap-1">
              <Clock size={12} />
              {isExpired ? 'Ended' : 'Ends'} {formatDate(poll.ends_at)}
            </span>
          )}
          {poll.multi_choice && (
            <span className="text-brand-400">Multi-choice</span>
          )}
        </div>
      </div>

      {/* Voting or Results */}
      {showResults ? (
        <div className="flex flex-col gap-3 mb-6">
          <div className="flex items-center justify-between mb-1">
            <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider">Results</h2>
            {voteStatus?.voted && (
              <span className="flex items-center gap-1 text-xs text-green-400">
                <CheckCircle2 size={13} /> You voted
              </span>
            )}
          </div>
          {result?.options.map((opt, i) => (
            <ResultBar
              key={opt.id}
              option={opt}
              index={i}
              isSelected={voteStatus?.option_ids?.includes(opt.id)}
              isWinner={!isClosed && opt.votes === winnerVotes && winnerVotes > 0}
            />
          )) ?? poll.options.map((opt, i) => (
            <ResultBar
              key={opt.id}
              option={{ id: opt.id, text: opt.text, votes: opt.votes, percentage: totalVotes ? (opt.votes / totalVotes) * 100 : 0 }}
              index={i}
              isSelected={voteStatus?.option_ids?.includes(opt.id)}
              isWinner={false}
            />
          ))}
        </div>
      ) : (
        <div className="flex flex-col gap-3 mb-6">
          <h2 className="text-sm font-semibold text-slate-400 uppercase tracking-wider mb-1">
            {isClosed ? 'This poll is closed' : 'Cast your vote'}
          </h2>
          {poll.options.map(opt => {
            const isChosen = selected.has(opt.id)
            return (
              <button
                key={opt.id}
                onClick={() => toggleOption(opt.id)}
                disabled={isClosed}
                className={`w-full text-left p-4 rounded-xl border transition-all ${
                  isChosen
                    ? 'border-brand-500/60 bg-brand-500/15 text-white'
                    : 'border-white/10 bg-white/3 text-slate-300 hover:border-white/25 hover:bg-white/6'
                } disabled:cursor-not-allowed`}
              >
                <div className="flex items-center gap-3">
                  <div className={`w-4 h-4 rounded-${poll.multi_choice ? 'sm' : 'full'} border-2 flex items-center justify-center shrink-0 transition-colors ${
                    isChosen ? 'border-brand-400 bg-brand-500' : 'border-slate-600'
                  }`}>
                    {isChosen && (
                      poll.multi_choice
                        ? <span className="text-white text-[10px] font-bold">✓</span>
                        : <span className="w-1.5 h-1.5 rounded-full bg-white" />
                    )}
                  </div>
                  <span className="text-sm">{opt.text}</span>
                </div>
              </button>
            )
          })}

          {!isClosed && (
            <div className="flex items-center gap-3 mt-2">
              <Button
                onClick={handleVote}
                loading={submitting}
                disabled={selected.size === 0}
                size="lg"
                className="flex-1"
              >
                Submit Vote
              </Button>
              <button
                onClick={() => setShowResults(true)}
                className="text-sm text-slate-500 hover:text-slate-300 transition-colors"
              >
                See results →
              </button>
            </div>
          )}
        </div>
      )}

      {/* Share + Actions */}
      <div className="card flex flex-col sm:flex-row gap-3 items-start sm:items-center justify-between mt-2">
        <div className="flex-1 min-w-0">
          <p className="text-xs text-slate-500 mb-1">Share this poll</p>
          <p className="text-sm text-slate-300 truncate font-mono">{window.location.href}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" size="sm" onClick={copyLink}>
            <Copy size={14} /> Copy
          </Button>
          <Button variant="secondary" size="sm" onClick={shareNative}>
            <Share2 size={14} /> Share
          </Button>
        </div>
      </div>

      {/* Creator actions */}
      {isCreator && !isClosed && (
        <div className="mt-4 flex justify-end">
          <Button
            variant="danger"
            size="sm"
            loading={closing}
            onClick={handleClose}
          >
            <Lock size={13} /> Close Poll
          </Button>
        </div>
      )}
    </div>
  )
}
