import { clsx } from 'clsx'
import { fmtPct } from '../../utils/format'
import type { OptionResult } from '../../api/types'

interface ResultBarProps {
  option: OptionResult
  isSelected?: boolean
  isWinner?: boolean
  index: number
}

const COLORS = [
  'from-brand-500 to-brand-400',
  'from-purple-500 to-purple-400',
  'from-pink-500 to-pink-400',
  'from-cyan-500 to-cyan-400',
  'from-amber-500 to-amber-400',
  'from-emerald-500 to-emerald-400',
  'from-rose-500 to-rose-400',
  'from-indigo-500 to-indigo-400',
  'from-teal-500 to-teal-400',
  'from-orange-500 to-orange-400',
]

export default function ResultBar({ option, isSelected, isWinner, index }: ResultBarProps) {
  const color = COLORS[index % COLORS.length]
  const pct   = Math.min(100, Math.max(0, option.percentage))

  return (
    <div
      className={clsx(
        'relative p-4 rounded-xl border transition-all',
        isSelected
          ? 'border-brand-500/50 bg-brand-500/10'
          : 'border-white/10 bg-white/3',
        isWinner && 'ring-1 ring-brand-400/40',
      )}
    >
      <div className="flex items-center justify-between mb-2 gap-2">
        <span className="text-sm font-medium text-slate-200 leading-tight">{option.text}</span>
        <div className="flex items-center gap-2 shrink-0">
          <span className="text-xs text-slate-400">{option.votes.toLocaleString()}</span>
          <span className={clsx('text-sm font-bold tabular-nums', isWinner ? 'text-brand-300' : 'text-slate-300')}>
            {fmtPct(pct)}
          </span>
        </div>
      </div>

      {/* Bar track */}
      <div className="h-2 rounded-full bg-white/5 overflow-hidden">
        <div
          className={clsx('h-full rounded-full bg-gradient-to-r vote-bar', color)}
          style={{ width: `${pct}%` }}
        />
      </div>

      {isSelected && (
        <span className="absolute top-2 right-2 text-[10px] text-brand-400 font-semibold uppercase tracking-wider">
          your vote
        </span>
      )}
    </div>
  )
}
