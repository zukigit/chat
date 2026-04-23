export interface StoredMessage {
  id: number
  conversation_id: number
  sender_id: string
  reply_to_message_id: number | null
  content: string
  message_type: string
  media_url: string | null
  is_edited: boolean
  deleted_at: string | null
  created_at: string
  updated_at: string
}

const STORAGE_KEY = 'chat_messages'

function loadAll(): Record<string, StoredMessage[]> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    return raw ? JSON.parse(raw) : {}
  } catch {
    return {}
  }
}

function saveAll(data: Record<string, StoredMessage[]>): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
}

export function getMessages(conversationId: number): StoredMessage[] {
  const all = loadAll()
  return all[String(conversationId)] ?? []
}

export function addMessage(msg: StoredMessage): void {
  const all = loadAll()
  const key = String(msg.conversation_id)
  if (!all[key]) all[key] = []
  if (!all[key].some(m => m.id === msg.id)) {
    all[key].push(msg)
    all[key].sort((a, b) => a.id - b.id)
  }
  saveAll(all)
}

export function clearMessages(): void {
  localStorage.removeItem(STORAGE_KEY)
}
