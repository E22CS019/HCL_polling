import { Link } from 'react-router-dom'
import { Users, Clock, Lock } from 'lucide-react'
import { timeAgo } from '../../utils/format'
import type { Poll } from '../../api/types'

interface PollCardProps {
  poll: Poll
}

export default function PollCard({ poll }: PollCardProps) {
  const isExpired = poll.ends_at ? new Date(poll.ends_at) < new Date() : false

  return (
    <Link
      to={`/poll/${poll.id}`}
      className="card hover:border-brand-500/30 hover:bg-white/8 transition-all group block animate-fade-in"
    >
      <div className="flex items-start justify-between gap-3">
        <h3 className="font-semibold text-white group-hover:text-brand-300 transition-colors line-clamp-2 leading-snug">
          {poll.title}
        </h3>
        {(poll.closed || isExpired) && (
          <span className="shrink-0 inline-flex items-center gap-1 text-xs text-slate-500 bg-white/5 px-2 py-0.5 rounded-full border border-white/10">
            <Lock size={10} /> Closed
          </span>
        )}
      </div>

      {poll.description && (
        <p className="text-sm text-slate-400 mt-1.5 line-clamp-2">{poll.description}</p>
      )}

      <div className="mt-4 flex items-center gap-4 text-xs text-slate-500">
        <span className="flex items-center gap-1">
          <Users size={12} />
          {poll.total_votes.toLocaleString()} vote{poll.total_votes !== 1 ? 's' : ''}
        </span>
        <span className="flex items-center gap-1">
          <Clock size={12} />
          {timeAgo(poll.created_at)}
        </span>
        <span>{poll.options.length} options</span>
      </div>
    </Link>
  )
}
