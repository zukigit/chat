import { useEffect, useRef, useState } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import type { Conversation, Message } from './fakeData'

interface Props {
  conversation: Conversation | null
  messages: Message[]
}

export default function MessagePanel({ conversation, messages }: Props) {
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

  return (
    <div className="chat-main">
      {/* Header */}
      <div className="chat-header">
        <div className="avatar avatar-sm" style={{ background: avatarColor(conversation.username) }}>
          {avatarInitials(conversation.name, conversation.username)}
          {conversation.online && <span className="online-dot" style={{ borderColor: 'var(--bg-header)' }} />}
        </div>
        <div className="chat-header-info">
          <div className="chat-header-name">{conversation.name || `@${conversation.username}`}</div>
          <div className={`chat-header-status${conversation.online ? ' online' : ''}`}>
            {conversation.online ? 'online' : 'last seen recently'}
          </div>
        </div>
      </div>

      {/* Messages */}
      <div className="messages-scroll">
        <div className="date-divider"><span>Today</span></div>
        {messages.map(m => (
          <div key={m.id} className={`msg-row ${m.own ? 'out' : 'in'}`}>
            <div className="msg-bubble">
              {m.text}
              <span className="msg-time">{m.time}</span>
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
          onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); setInput('') } }}
        />
        <button className="send-btn" onClick={() => setInput('')} aria-label="Send">
          ➤
        </button>
      </div>
    </div>
  )
}
