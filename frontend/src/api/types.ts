export interface User {
  id: string
  username: string
  email: string
  created_at: string
}

export interface PollOption {
  id: string
  text: string
  votes: number
}

export interface Poll {
  id: string
  creator_id: string
  title: string
  description: string
  options: PollOption[]
  multi_choice: boolean
  ends_at: string | null
  created_at: string
  total_votes: number
  closed: boolean
}

export interface OptionResult {
  id: string
  text: string
  votes: number
  percentage: number
}

export interface PollResult {
  poll_id: string
  total_votes: number
  options: OptionResult[]
  updated_at: string
}

export interface AuthResponse {
  token: string
  user: User
}

export interface VoteStatusResponse {
  voted: boolean
  option_ids: string[]
}
