import { getToken } from '../auth'
import { loadConfig } from '../config'

export interface ApiFriend {
  user_id: string
  username: string
  display_name: string
  avatar_url: string
  status: 'accepted' | 'pending'
}

function gatewayUrl(): string {
  const config = loadConfig()
  if (!config?.gatewayUrl) throw new Error('gateway URL not configured')
  return config.gatewayUrl
}

function authHeader(): HeadersInit {
  const token = getToken()
  if (!token) throw new Error('not authenticated')
  return { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' }
}

export async function fetchFriends(): Promise<ApiFriend[]> {
  const res = await fetch(`${gatewayUrl()}/friends`, { headers: authHeader() })
  const body = await res.json()
  if (!res.ok || !body.success) throw new Error(body.message ?? 'failed to fetch friends')
  return (body.data ?? []) as ApiFriend[]
}

export async function acceptFriendRequest(username: string): Promise<void> {
  const res = await fetch(`${gatewayUrl()}/friends/accept`, {
    method: 'POST',
    headers: authHeader(),
    body: JSON.stringify({ username }),
  })
  const body = await res.json()
  if (!res.ok || !body.success) throw new Error(body.message ?? 'failed to accept request')
}

export async function rejectFriendRequest(username: string): Promise<void> {
  const res = await fetch(`${gatewayUrl()}/friends/reject`, {
    method: 'POST',
    headers: authHeader(),
    body: JSON.stringify({ username }),
  })
  const body = await res.json()
  if (!res.ok || !body.success) throw new Error(body.message ?? 'failed to decline request')
}
