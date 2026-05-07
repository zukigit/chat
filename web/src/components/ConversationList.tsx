import { useState } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import type { ApiConversation } from '../api/conversationsApi'
import type { StoredMessage } from '../messageStore'
import SearchConversationModal from './SearchConversationModal'

interface Props {
  conversations: ApiConversation[]
  activeId: number | null
  currentUsername: string
  messages: Record<number, StoredMessage[]>
  onSelect: (conv: ApiConversation) => void
}

export default function ConversationList({ conversations, activeId, currentUsername, messages, onSelect }: Props) {
  const [showSearchModal, setShowSearchModal] = useState(false)

  const searchIcon = (
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
  )

  if (conversations.length === 0) {
    return (
      <div className="friends-wrapper">
        <div className="sidebar-list-empty">
          No conversations yet
        </div>
        <button className="fab" title="Search Conversations" onClick={() => setShowSearchModal(true)}>
          {searchIcon}
        </button>
        <SearchConversationModal open={showSearchModal} onClose={() => setShowSearchModal(false)} onSelect={onSelect} />
      </div>
    )
  }

  return (
    <div className="friends-wrapper">
      <div className="sidebar-list">
        {conversations.map(c => {
          const otherMember = c.members.find(m => m.username !== currentUsername)
          const displayName = c.is_group ? c.name : (otherMember?.display_name || otherMember?.username || '')
          const username = otherMember?.username || ''
          const convMessages = messages[c.id] ?? []
          const lastMsg = convMessages.length > 0 ? convMessages[convMessages.length - 1] : null
          const preview = lastMsg ? (lastMsg.content.length > 40 ? lastMsg.content.slice(0, 40) + '…' : lastMsg.content) : 'No messages yet'
          const time = lastMsg
            ? new Date(lastMsg.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
            : new Date(c.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
          return (
            <div
              key={c.id}
              className={`list-item${activeId === c.id ? ' active' : ''}`}
              onClick={() => onSelect(c)}
            >
              <div className="avatar" style={{ background: avatarColor(username) }}>
                {avatarInitials(displayName, username)}
              </div>
              <div className="item-body">
                <div className="item-top">
                  <span className="item-name">{displayName}</span>
                  <span className="item-time">{time}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span className="item-preview">{preview}</span>
                </div>
              </div>
            </div>
          )
        })}
      </div>
      <button className="fab" title="Search Conversations" onClick={() => setShowSearchModal(true)}>
        {searchIcon}
      </button>
      <SearchConversationModal open={showSearchModal} onClose={() => setShowSearchModal(false)} onSelect={onSelect} />
    </div>
  )
}
