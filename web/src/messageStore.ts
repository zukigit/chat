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

export interface SentMessage {
  tempId: string
  conversation_id: number
  content: string
  status: 'sending' | 'sent' | 'delivered'
  created_at?: string
}

const STORAGE_KEY = 'chat_messages'
const SENT_KEY = 'chat_sent'

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

function loadSent(): SentMessage[] {
  try {
    const raw = localStorage.getItem(SENT_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

function saveSent(data: SentMessage[]): void {
  localStorage.setItem(SENT_KEY, JSON.stringify(data))
}

export function getMessages(conversationId: number): StoredMessage[] {
  const all = loadAll()
  return all[String(conversationId)] ?? []
}

export function getSentMessages(conversationId: number): SentMessage[] {
  return loadSent().filter(s => s.conversation_id === conversationId)
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
  const sent = loadSent().filter(s => !(s.conversation_id === msg.conversation_id && s.content === msg.content))
  saveSent(sent)
}

export function addSentMessage(conversationId: number, content: string): SentMessage {
  const tempId = `sent-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  const sent: SentMessage = { tempId, conversation_id: conversationId, content, status: 'sending', created_at: new Date().toISOString() }
  const all = loadSent()
  all.push(sent)
  saveSent(all)
  return sent
}

export function markSentDelivered(conversationId: number): void {
  const all = loadSent()
  const s = all.find(x => x.conversation_id === conversationId && x.status === 'sending')
  if (s) {
    s.status = 'delivered'
    saveSent(all)
  }
}

export function markSentByContent(conversationId: number, content: string): void {
  const all = loadSent()
  const s = all.find(x => x.conversation_id === conversationId && x.content === content && x.status === 'sending')
  if (s) {
    s.status = 'sent'
    saveSent(all)
  }
}

export function removeSent(tempId: string): void {
  const all = loadSent().filter(s => s.tempId !== tempId)
  saveSent(all)
}

export function clearMessages(): void {
  localStorage.removeItem(STORAGE_KEY)
  localStorage.removeItem(SENT_KEY)
}
