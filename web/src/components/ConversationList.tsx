import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import type { ApiConversation } from '../api/conversationsApi'
import type { StoredMessage } from '../messageStore'

interface Props {
  conversations: ApiConversation[]
  activeId: number | null
  currentUsername: string
  messages: Record<number, StoredMessage[]>
  onSelect: (conv: ApiConversation) => void
}

export default function ConversationList({ conversations, activeId, currentUsername, messages, onSelect }: Props) {
  if (conversations.length === 0) {
    return (
      <div className="sidebar-list-empty">
        No conversations yet
      </div>
    )
  }

  return (
    <>
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
    </>
  )
}
