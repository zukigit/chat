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
}

export default function FriendsList({ friends, friendRequests, onStartChat, onAccepted, onDeclined }: Props) {
  const [requestsOpen, setRequestsOpen] = useState(true)
  const [friendsOpen, setFriendsOpen] = useState(false)
  const [loadingId, setLoadingId] = useState<string | null>(null)
  const [errors, setErrors] = useState<Record<string, string>>({})

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
        <div className="friends-wrapper">
        <div className="sidebar-list">

          {/* ── Friend Requests section ── */}
          <div className="section-header" onClick={() => setRequestsOpen(o => !o)}>
            <span className="section-title">Friend Requests</span>
            <span className="section-meta">
              <span style={{ fontSize: 12, color: 'var(--color-sub)', marginRight: 6 }}>{friendRequests.length}</span>
              <svg className={`section-chevron${requestsOpen ? ' open' : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="6 9 12 15 18 9" />
              </svg>
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
              <svg className={`section-chevron${friendsOpen ? ' open' : ''}`} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="6 9 12 15 18 9" />
              </svg>
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
        <button className="fab-send-request" title="Add Friend">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
            <circle cx="8.5" cy="7" r="4" />
            <line x1="20" y1="8" x2="20" y2="14" />
            <line x1="23" y1="11" x2="17" y2="11" />
          </svg>
        </button>
      </div>
    </>
  )
}
