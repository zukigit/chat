import './chat.css'
import type { Friend } from './fakeData'

interface Props {
  friends: Friend[]
  onStartChat: (friend: Friend) => void
}

function initials(name: string) {
  return name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
}

export default function FriendsList({ friends, onStartChat }: Props) {
  return (
    <>
      <div className="sidebar-header">
        <span className="sidebar-title">Friends</span>
        <span style={{ fontSize: 13, color: 'var(--color-sub)' }}>{friends.length} total</span>
      </div>
      <input className="sidebar-search" placeholder="Search friends…" readOnly />
      <div className="sidebar-list">
        {friends.map(f => (
          <div
            key={f.id}
            className="list-item"
            onClick={() => onStartChat(f)}
            title="Start chat"
          >
            <div className="avatar" style={{ background: f.avatarColor }}>
              {initials(f.displayName)}
              {f.online && <span className="online-dot" />}
            </div>
            <div className="item-body">
              <div className="item-top">
                <span className="item-name">{f.displayName}</span>
              </div>
              <div className="item-preview">
                <span className="item-username">@{f.username}</span>
                {f.online && <span className="online-label" style={{ marginLeft: 8 }}>● online</span>}
              </div>
            </div>
          </div>
        ))}
      </div>
    </>
  )
}
