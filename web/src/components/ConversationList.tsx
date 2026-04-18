import './chat.css'
import type { Conversation } from './fakeData'

interface Props {
  conversations: Conversation[]
  activeId: string | null
  onSelect: (conv: Conversation) => void
}

function initials(name: string) {
  return name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
}

export default function ConversationList({ conversations, activeId, onSelect }: Props) {
  return (
    <>
      <div className="sidebar-header">
        <span className="sidebar-title">Messages</span>
      </div>
      <input className="sidebar-search" placeholder="Search conversations…" readOnly />
      <div className="sidebar-list">
        {conversations.map(c => (
          <div
            key={c.id}
            className={`list-item${activeId === c.id ? ' active' : ''}`}
            onClick={() => onSelect(c)}
          >
            <div className="avatar" style={{ background: c.avatarColor }}>
              {initials(c.name)}
              {c.online && <span className="online-dot" />}
            </div>
            <div className="item-body">
              <div className="item-top">
                <span className="item-name">{c.name}</span>
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
