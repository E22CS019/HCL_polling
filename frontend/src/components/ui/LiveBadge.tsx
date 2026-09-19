import { clsx } from 'clsx'

type Status = 'connecting' | 'live' | 'error' | 'closed'

interface LiveBadgeProps {
  status: Status
}

const config: Record<Status, { bg: string; text: string; dot: string; label: string }> = {
  live:       { bg: 'bg-green-500/15',  text: 'text-green-400',  dot: 'live-dot',                              label: 'Live' },
  connecting: { bg: 'bg-yellow-500/15', text: 'text-yellow-400', dot: 'w-2 h-2 rounded-full bg-yellow-400 animate-pulse', label: 'Connecting' },
  error:      { bg: 'bg-red-500/15',    text: 'text-red-400',    dot: 'w-2 h-2 rounded-full bg-red-400',       label: 'Reconnecting' },
  closed:     { bg: 'bg-slate-500/15',  text: 'text-slate-400',  dot: 'w-2 h-2 rounded-full bg-slate-400',    label: 'Closed' },
}

export default function LiveBadge({ status }: LiveBadgeProps) {
  const { bg, text, dot, label } = config[status]
  return (
    <span className={clsx('inline-flex items-center gap-1.5 text-xs font-medium px-2.5 py-1 rounded-full', bg, text)}>
      <span className={dot} aria-hidden="true" />
      {label}
    </span>
  )
}
