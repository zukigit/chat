import { useEffect, useRef, useState } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import type { ApiConversation } from '../api/conversationsApi'
import type { StoredMessage, SentMessage } from '../messageStore'
import { addSentMessage } from '../messageStore'

interface Props {
  conversation: ApiConversation | null
  messages: StoredMessage[]
  sentMessages: SentMessage[]
  currentUsername: string
  onSend: (conversationId: number, content: string, tempId: string) => void
}

export default function MessagePanel({ conversation, messages, sentMessages, currentUsername, onSend }: Props) {
  const [input, setInput] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, sentMessages])

  if (!conversation) {
    return (
      <div className="chat-main">
        <div className="chat-empty">
          <div className="chat-empty-icon">💬</div>
          <div className="chat-empty-text">Select a conversation to start chatting</div>
        </div>
      </div>
    )
  }

  const otherMember = conversation.members.find(m => m.username !== currentUsername)
  const displayName = conversation.is_group ? conversation.name : (otherMember?.display_name || otherMember?.username || '')
  const username = otherMember?.username || ''
  const currentUserId = conversation.members.find(mem => mem.username === currentUsername)?.user_id ?? ''

  function handleSend() {
    const text = input.trim()
    if (!text || !conversation) return
    const sent = addSentMessage(conversation.id, text)
    onSend(conversation.id, text, sent.tempId)
    setInput('')
  }

  function renderStatus(status: string) {
    if (status === 'sending') {
      return (
        <span className="msg-status">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        </span>
      )
    }
    if (status === 'sent') {
      return (
        <span className="msg-status">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="18 6 7 17 2 12" />
            <polyline points="22 6 11 17" />
          </svg>
        </span>
      )
    }
    if (status === 'delivered') {
      return (
        <span className="msg-status msg-delivered">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
            <polyline points="18 6 7 17 2 12" />
            <polyline points="22 6 11 17" />
          </svg>
        </span>
      )
    }
    return null
  }

  return (
    <div className="chat-main">
      {/* Header */}
      <div className="chat-header">
        <div className="avatar avatar-sm" style={{ background: avatarColor(username) }}>
          {avatarInitials(displayName, username)}
        </div>
        <div className="chat-header-info">
          <div className="chat-header-name">{displayName}</div>
        </div>
      </div>

      {/* Messages */}
      <div className="messages-scroll">
        {messages.length === 0 && sentMessages.length === 0 && (
          <div className="date-divider"><span>No messages yet</span></div>
        )}
        {messages.map(m => {
          const isOwn = m.sender_id === currentUserId
          return (
            <div key={m.id} className={`msg-row ${isOwn ? 'out' : 'in'}`}>
              <div className="msg-bubble">
                {m.content}
                <span className="msg-time">{new Date(m.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
              </div>
            </div>
          )
        })}
        {sentMessages.map(s => (
          <div key={s.tempId} className="msg-row out">
            <div className="msg-bubble">
              {s.content}
              {renderStatus(s.status)}
            </div>
          </div>
        ))}
        <div ref={bottomRef} />
      </div>

      {/* Input */}
      <div className="chat-input-bar">
        <input
          className="chat-input"
          placeholder="Write a message…"
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend() } }}
        />
        <button className="send-btn" onClick={handleSend} aria-label="Send">
          ➤
        </button>
      </div>
    </div>
  )
}
