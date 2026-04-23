import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import type { ApiConversation } from '../api/conversationsApi'

interface Props {
  conversations: ApiConversation[]
  activeId: number | null
  currentUsername: string
  onSelect: (conv: ApiConversation) => void
}

export default function ConversationList({ conversations, activeId, currentUsername, onSelect }: Props) {
  return (
    <>
      <div className="sidebar-list">
        {conversations.map(c => {
          const otherMember = c.members.find(m => m.username !== currentUsername)
          const displayName = c.is_group ? c.name : (otherMember?.display_name || otherMember?.username || '')
          const username = otherMember?.username || ''
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
                  <span className="item-time">{new Date(c.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                </div>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span className="item-preview">No messages yet</span>
                </div>
              </div>
            </div>
          )
        })}
      </div>
    </>
  )
}
