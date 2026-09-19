import { useEffect, useRef, useState } from 'react'
import { streamUrl } from '../api/polls'
import type { PollResult } from '../api/types'

type Status = 'connecting' | 'live' | 'error' | 'closed'

/**
 * Opens an SSE connection to the backend stream endpoint and keeps
 * PollResult state updated in real time.
 *
 * The EventSource is reconnected automatically by the browser on network
 * hiccups (standard SSE behaviour). We expose a `status` value so the UI
 * can show a "live" indicator.
 */
export function useLiveResults(pollId: string, initialResult?: PollResult) {
  const [result, setResult]   = useState<PollResult | undefined>(initialResult)
  const [status, setStatus]   = useState<Status>('connecting')
  const esRef                 = useRef<EventSource | null>(null)

  useEffect(() => {
    if (!pollId) return

    const es = new EventSource(streamUrl(pollId))
    esRef.current = es

    es.addEventListener('result', (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as PollResult
        setResult(data)
        setStatus('live')
      } catch {
        // malformed message — ignore
      }
    })

    es.onerror = () => {
      setStatus('error')
      // The browser will automatically retry; update status when it reconnects.
    }

    es.onopen = () => setStatus('live')

    return () => {
      es.close()
      esRef.current = null
      setStatus('closed')
    }
  }, [pollId])

  return { result, status }
}
