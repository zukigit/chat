import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import type { Conversation } from './fakeData'

interface Props {
  conversations: Conversation[]
  activeId: string | null
  onSelect: (conv: Conversation) => void
}

export default function ConversationList({ conversations, activeId, onSelect }: Props) {
  return (
    <>
      <div className="sidebar-list">
        {conversations.map(c => (
          <div
            key={c.id}
            className={`list-item${activeId === c.id ? ' active' : ''}`}
            onClick={() => onSelect(c)}
          >
            <div className="avatar" style={{ background: avatarColor(c.username) }}>
              {avatarInitials(c.name, c.username)}
              {c.online && <span className="online-dot" />}
            </div>
            <div className="item-body">
              <div className="item-top">
                <span className="item-name">{c.name || c.username}</span>
                <span className="item-time">{c.time}</span>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span className="item-preview">{c.lastMessage}</span>
                {c.unread > 0 && (
                  <span className="unread-badge">{c.unread > 99 ? '99+' : c.unread}</span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </>
  )
}
