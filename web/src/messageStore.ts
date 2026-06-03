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
  remote?: boolean
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

  // Remove sent entries that correspond to this now-received message.
  // Always runs, even when the message was deduplicated (handles reconnect).
  const ageArmorStart = '-----BEGIN AGE ENCRYPTED FILE-----'
  const sent = loadSent().filter(s => {
    if (s.conversation_id !== msg.conversation_id) return true
    // Remote entries are kept for cross-device status tracking
    if (s.remote) return true
    // Local sent entry: matched by tempId === server-assigned message ID
    if (s.tempId === msg.id) return false
    // Stale age-armor entries from pre-fix data
    if (s.content.startsWith(ageArmorStart)) return false
    return true
  })
  saveSent(sent)
}

/**
 * Returns true if a locally-created sent entry exists for this message.
 * Used to avoid calling addRemoteSentMessage for messages sent locally.
 */
export function hasPendingSent(conversationId: number, messageId: string): boolean {
  return loadSent().some(s =>
    s.conversation_id === conversationId &&
    s.tempId === messageId &&
    !s.tempId.startsWith('sent-remote-')
  )
}

function generateUUID(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16)
  })
}

export function addSentMessage(conversationId: number, content: string): SentMessage {
  const tempId = generateUUID()
  const sent: SentMessage = { tempId, conversation_id: conversationId, content, status: 'sending', created_at: new Date().toISOString() }
  const all = loadSent()
  all.push(sent)
  saveSent(all)
  return sent
}

export function addRemoteSentMessage(conversationId: number, messageId: string, content: string, createdAt?: string): void {
  const all = loadSent()
  if (all.some(s => s.conversation_id === conversationId && s.tempId === messageId)) return
  all.push({ tempId: messageId, conversation_id: conversationId, content, status: 'sent', created_at: createdAt || new Date().toISOString(), remote: true })
  saveSent(all)
}

export function markSentSent(conversationId: number, messageId?: string): void {
  const all = loadSent()
  const msg = all.find(x => x.conversation_id === conversationId && x.tempId === messageId && x.status === 'sending')
  if (msg) {
    msg.status = 'sent'
    saveSent(all)
  }
}

export function markSentDelivered(conversationId: number, messageId?: string): void {
  const all = loadSent()
  const msg = all.find(x => x.conversation_id === conversationId && x.tempId === messageId && x.status === 'sent')
  if (msg) {
    msg.status = 'delivered'
    saveSent(all)
  }
}

export function markSentSeen(conversationId: number, messageId?: string): void {
  const all = loadSent()
  const msg = all.find(x => x.conversation_id === conversationId && x.tempId === messageId && (x.status === 'sent' || x.status === 'delivered'))
  if (msg) {
    msg.status = 'seen'
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

export function markSentFailed(conversationId: number, messageId?: string): void {
  const all = loadSent()
  const s = all.find(x => x.conversation_id === conversationId && x.tempId === messageId && (x.status === 'sending' || x.status === 'sent'))
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
