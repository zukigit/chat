import { getToken } from '../auth'
import { loadConfig } from '../config'

export interface ApiFriend {
  user_id: string
  username: string
  display_name: string
  avatar_url: string
  status: 'accepted' | 'pending'
}

export async function fetchFriends(): Promise<ApiFriend[]> {
  const token = getToken()
  if (!token) throw new Error('not authenticated')

  const config = loadConfig()
  if (!config?.gatewayUrl) throw new Error('gateway URL not configured')

  const res = await fetch(`${config.gatewayUrl}/friends`, {
    headers: { Authorization: `Bearer ${token}` },
  })

  const body = await res.json()
  if (!res.ok || !body.success) {
    throw new Error(body.message ?? 'failed to fetch friends')
  }

  return (body.data ?? []) as ApiFriend[]
}
