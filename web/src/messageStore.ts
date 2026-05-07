export interface StoredMessage {
  id: string
  conversation_id: number
  sender_id: string
  reply_to_message_id: string | null
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
  status: 'sending' | 'sent' | 'delivered' | 'seen' | 'failed'
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
    all[key].sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
  }
  saveAll(all)
  const sent = loadSent().filter(s => !(s.conversation_id === msg.conversation_id && s.content === msg.content && !s.tempId.startsWith('sent-remote-')))
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

export function addRemoteSentMessage(conversationId: number, content: string): void {
  const all = loadSent()
  if (all.some(s => s.conversation_id === conversationId && s.content === content && s.tempId.startsWith('sent-remote-'))) return
  const tempId = `sent-remote-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
  all.push({ tempId, conversation_id: conversationId, content, status: 'sending', created_at: new Date().toISOString() })
  saveSent(all)
}

export function markSentDelivered(_conversationId: number, _messageId?: string): void {
  const all = loadSent()
  const sending = all.filter(x => x.conversation_id === _conversationId && x.status === 'sending')
  if (sending.length > 0) {
    sending[sending.length - 1].status = 'delivered'
    saveSent(all)
  }
}

export function markSentSeen(_conversationId: number, _messageId?: string): void {
  const all = loadSent()
  all.forEach(s => {
    if (s.conversation_id === _conversationId && (s.status === 'sent' || s.status === 'delivered')) {
      s.status = 'seen'
    }
  })
  saveSent(all)
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

export function markSentFailed(conversationId: number, content: string): void {
  const all = loadSent()
  const s = all.find(x => x.conversation_id === conversationId && x.content === content && x.status === 'sending')
  if (s) {
    s.status = 'failed'
    saveSent(all)
  }
}

export function retrySentMessage(tempId: string): void {
  const all = loadSent()
  const s = all.find(x => x.tempId === tempId)
  if (s && s.status === 'failed') {
    s.status = 'sending'
    saveSent(all)
  }
}
