import { useState } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import { acceptFriendRequest, rejectFriendRequest } from '../api/friendsApi'
import type { Friend, FriendRequest } from './fakeData'

interface Props {
  friends: Friend[]
  friendRequests: FriendRequest[]
  onStartChat: (friend: Friend) => void
  onAccepted?: (req: FriendRequest) => void
  onDeclined?: (req: FriendRequest) => void
  onRefresh?: () => Promise<void>
}

export default function FriendsList({ friends, friendRequests, onStartChat, onAccepted, onDeclined, onRefresh }: Props) {
  const [requestsOpen, setRequestsOpen] = useState(true)
  const [friendsOpen, setFriendsOpen] = useState(true)
  const [loadingId, setLoadingId] = useState<string | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})
  const [refreshing, setRefreshing] = useState<'requests' | 'friends' | null>(null)

  async function handleRefresh(section: 'requests' | 'friends', e: React.MouseEvent) {
    e.stopPropagation()
    if (refreshing || !onRefresh) return
    setRefreshing(section)
    try { await onRefresh() } finally { setRefreshing(null) }
  }

  async function handleAction(req: FriendRequest, action: 'accept' | 'decline') {
    if (loadingId) return
    const key = `${req.id}-${action}`
    setLoadingId(key)
    setErrors(e => { const n = { ...e }; delete n[req.id]; return n })
    try {
      if (action === 'accept') {
        await acceptFriendRequest(req.username)
        onAccepted?.(req)
      } else {
        await rejectFriendRequest(req.username)
        onDeclined?.(req)
      }
    } catch (err) {
      setErrors(e => ({ ...e, [req.id]: (err as Error).message }))
    } finally {
      setLoadingId(null)
    }
  }

  return (
    <>
      <div className="sidebar-list">

        {/* ── Friend Requests section ── */}
        <div className="section-header" onClick={() => setRequestsOpen(o => !o)}>
          <span className="section-title">Friend Requests</span>
          <span className="section-meta">
            {friendRequests.length > 0 && (
              <span className="unread-badge" style={{ marginRight: 6 }}>{friendRequests.length}</span>
            )}
            <button
              className={`refresh-btn${refreshing === 'requests' ? ' spinning' : ''}`}
              onClick={e => handleRefresh('requests', e)}
              disabled={refreshing !== null}
              title="Refresh"
            >↺</button>
            <span className="section-chevron">{requestsOpen ? '▾' : '▸'}</span>
          </span>
        </div>

        {requestsOpen && (
          friendRequests.length === 0
            ? <div className="section-empty">No pending requests</div>
            : friendRequests.map(req => (
              <div key={req.id} className="list-item request-item">
                <div className="avatar" style={{ background: avatarColor(req.username) }}>
                  {avatarInitials(req.displayName, req.username)}
                </div>
                <div className="item-body">
                  <div className="item-top">
                    <span className="item-name">{req.displayName || req.username}</span>
                  </div>
                  <div className="item-preview">
                    <span className="item-username">@{req.username}</span>
                  </div>
                  {errors[req.id] && (
                    <div className="req-error">{errors[req.id]}</div>
                  )}
                  <div className="request-actions">
                    <button
                      className="req-btn accept"
                      onClick={() => handleAction(req, 'accept')}
                      disabled={loadingId !== null}
                    >
                      {loadingId === `${req.id}-accept` ? '…' : 'Accept'}
                    </button>
                    <button
                      className="req-btn decline"
                      onClick={() => handleAction(req, 'decline')}
                      disabled={loadingId !== null}
                    >
                      {loadingId === `${req.id}-decline` ? '…' : 'Decline'}
                    </button>
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
            <button
              className={`refresh-btn${refreshing === 'friends' ? ' spinning' : ''}`}
              onClick={e => handleRefresh('friends', e)}
              disabled={refreshing !== null}
              title="Refresh"
            >↺</button>
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
            <div className="avatar" style={{ background: avatarColor(f.username) }}>
              {avatarInitials(f.displayName, f.username)}
            </div>
            <div className="item-body">
              <div className="item-top">
                <span className="item-name">{f.displayName || f.username}</span>
              </div>
              <div className="item-preview">
                <span className="item-username">@{f.username}</span>
              </div>
            </div>
          </div>
        ))}

      </div>
    </>
  )
}
