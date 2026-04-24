import { getToken } from '../auth'
import { loadConfig } from '../config'

export interface ConversationMember {
  user_id: string
  username: string
  display_name: string
  avatar_url: string
}

export interface ApiConversation {
  id: number
  is_group: boolean
  name: string
  updated_at: string
  members: ConversationMember[]
}

function gatewayUrl(): string {
  const config = loadConfig()
  if (!config?.gatewayUrl) throw new Error('gateway URL not configured')
  return config.gatewayUrl
}

function authHeader(): HeadersInit {
  const token = getToken()
  if (!token) throw new Error('not authenticated')
  return { Authorization: `Bearer ${token}` }
}

export async function fetchConversations(): Promise<ApiConversation[]> {
  const res = await fetch(`${gatewayUrl()}/conversations`, { headers: authHeader() })
  const body = await res.json()
  if (!res.ok || !body.success) throw new Error(body.message ?? 'failed to fetch conversations')
  return (body.data?.conversations ?? []) as ApiConversation[]
}

export async function createConversation(membersUsername: string[], isGroup = false, name = ''): Promise<number> {
  const res = await fetch(`${gatewayUrl()}/conversations`, {
    method: 'POST',
    headers: { ...authHeader(), 'Content-Type': 'application/json' },
    body: JSON.stringify({ is_group: isGroup, name, members_username: membersUsername }),
  })
  const body = await res.json()
  if (!res.ok || !body.success) throw new Error(body.message ?? 'failed to create conversation')
  return body.data.conversation_id as number
}
