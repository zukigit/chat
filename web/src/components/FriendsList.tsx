import { useState } from 'react'
import './chat.css'
import type { Friend, FriendRequest } from './fakeData'

interface Props {
  friends: Friend[]
  friendRequests: FriendRequest[]
  onStartChat: (friend: Friend) => void
  onAcceptRequest?: (req: FriendRequest) => void
  onDeclineRequest?: (req: FriendRequest) => void
}

function initials(name: string) {
  return name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
}

export default function FriendsList({ friends, friendRequests, onStartChat, onAcceptRequest, onDeclineRequest }: Props) {
  const [requestsOpen, setRequestsOpen] = useState(true)
  const [friendsOpen, setFriendsOpen] = useState(true)

  return (
    <>
      <input className="sidebar-search" placeholder="Search friends…" readOnly style={{ marginTop: 10 }} />
      <div className="sidebar-list">

        {/* ── Friend Requests section ── */}
        <div className="section-header" onClick={() => setRequestsOpen(o => !o)}>
          <span className="section-title">Friend Requests</span>
          <span className="section-meta">
            {friendRequests.length > 0 && (
              <span className="unread-badge" style={{ marginRight: 6 }}>{friendRequests.length}</span>
            )}
            <span className="section-chevron">{requestsOpen ? '▾' : '▸'}</span>
          </span>
        </div>

        {requestsOpen && (
          friendRequests.length === 0
            ? <div className="section-empty">No pending requests</div>
            : friendRequests.map(req => (
              <div key={req.id} className="list-item request-item">
                <div className="avatar" style={{ background: req.avatarColor }}>
                  {initials(req.displayName)}
                </div>
                <div className="item-body">
                  <div className="item-top">
                    <span className="item-name">{req.displayName}</span>
                  </div>
                  <div className="item-preview">
                    <span className="item-username">@{req.username}</span>
                  </div>
                  <div className="request-actions">
                    <button className="req-btn accept" onClick={() => onAcceptRequest?.(req)}>Accept</button>
                    <button className="req-btn decline" onClick={() => onDeclineRequest?.(req)}>Decline</button>
                  </div>
                </div>
              </div>
            ))
        )}

        {/* ── Friends section ── */}
        <div className="section-header" onClick={() => setFriendsOpen(o => !o)}>
          <span className="section-title">Friends</span>
          <span className="section-meta">
            <span style={{ fontSize: 12, color: 'var(--color-sub)', marginRight: 6 }}>{friends.length}</span>
            <span className="section-chevron">{friendsOpen ? '▾' : '▸'}</span>
          </span>
        </div>

        {friendsOpen && friends.map(f => (
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
