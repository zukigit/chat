import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'

interface Props {
  username: string
}

export default function ProfilePanel({ username }: Props) {
  return (
    <div className="sidebar-list">
      <div className="list-item" style={{ cursor: 'default', padding: '20px 14px', flexDirection: 'column', alignItems: 'center', gap: 10 }}>
        <div className="avatar" style={{ background: avatarColor(username), width: 64, height: 64, fontSize: 24 }}>
          {avatarInitials(username, username)}
        </div>
        <div style={{ textAlign: 'center' }}>
          <div style={{ fontSize: 16, fontWeight: 600 }}>{username}</div>
        </div>
      </div>
    </div>
  )
}
