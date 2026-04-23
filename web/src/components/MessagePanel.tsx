import { useEffect, useRef, useState } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import type { ApiConversation } from '../api/conversationsApi'
import type { StoredMessage } from '../messageStore'

interface Props {
  conversation: ApiConversation | null
  messages: StoredMessage[]
  currentUsername: string
  onSend: (conversationId: number, content: string) => void
}

export default function MessagePanel({ conversation, messages, currentUsername, onSend }: Props) {
  const [input, setInput] = useState('')
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

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

  function handleSend() {
    const text = input.trim()
    if (!text || !conversation) return
    onSend(conversation.id, text)
    setInput('')
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
        {messages.length === 0 && (
          <div className="date-divider"><span>No messages yet</span></div>
        )}
        {messages.map(m => {
          const isOwn = m.sender_id === (conversation.members.find(mem => mem.username === currentUsername)?.user_id ?? '')
          return (
            <div key={m.id} className={`msg-row ${isOwn ? 'out' : 'in'}`}>
              <div className="msg-bubble">
                {m.content}
                <span className="msg-time">{new Date(m.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
              </div>
            </div>
          )
        })}
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
