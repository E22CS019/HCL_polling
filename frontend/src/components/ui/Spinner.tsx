import { clsx } from 'clsx'

interface SpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const sizes = { sm: 'w-4 h-4', md: 'w-7 h-7', lg: 'w-12 h-12' }

export default function Spinner({ size = 'md', className }: SpinnerProps) {
  return (
    <div
      className={clsx(
        'animate-spin rounded-full border-2 border-brand-500/30 border-t-brand-400',
        sizes[size],
        className,
      )}
      role="status"
      aria-label="Loading"
    />
  )
}
