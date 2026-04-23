import { useEffect, useRef, useState } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import type { Message } from './fakeData'
import type { ApiConversation } from '../api/conversationsApi'

interface Props {
  conversation: ApiConversation | null
  messages: Message[]
  currentUsername: string
}

export default function MessagePanel({ conversation, messages, currentUsername }: Props) {
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
