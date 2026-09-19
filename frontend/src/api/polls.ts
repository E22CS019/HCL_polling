import client from './client'
import { getFingerprint } from '../utils/fingerprint'
import type { Poll, PollResult, VoteStatusResponse } from './types'

const fp = () => ({ headers: { 'X-Voter-Fingerprint': getFingerprint() } })

export const listRecent = () =>
  client.get<Poll[]>('/polls').then(r => r.data)

export const getPoll = (id: string) =>
  client.get<Poll>(`/polls/${id}`).then(r => r.data)

export const getMyPolls = () =>
  client.get<Poll[]>('/polls/mine').then(r => r.data)

export const createPoll = (data: {
  title: string
  description: string
  options: string[]
  multi_choice: boolean
  ends_at?: string | null
}) => client.post<Poll>('/polls', data).then(r => r.data)

export const vote = (pollId: string, optionIds: string[]) =>
  client
    .post<PollResult>(`/polls/${pollId}/vote`, { option_ids: optionIds }, fp())
    .then(r => r.data)

export const getResults = (pollId: string) =>
  client.get<PollResult>(`/polls/${pollId}/results`).then(r => r.data)

export const getVoteStatus = (pollId: string) =>
  client.get<VoteStatusResponse>(`/polls/${pollId}/vote-status`, fp()).then(r => r.data)

export const closePoll = (pollId: string) =>
  client.post(`/polls/${pollId}/close`).then(r => r.data)

/** Returns the SSE stream URL for a poll (used with EventSource). */
export const streamUrl = (pollId: string) => {
  const base = import.meta.env.VITE_API_URL
    ? `${import.meta.env.VITE_API_URL}/api/v1`
    : '/api/v1'
  return `${base}/polls/${pollId}/stream`
}
