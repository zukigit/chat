import { useState } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'

interface User {
  id: string
  username: string
  displayName: string
}

const FAKE_USERS: User[] = [
  { id: 'u1', username: 'james',   displayName: 'James Chen'     },
  { id: 'u2', username: 'kate',    displayName: 'Kate Rodriguez' },
  { id: 'u3', username: 'liam',    displayName: ''               },
  { id: 'u4', username: 'nina',    displayName: 'Nina Park'      },
  { id: 'u5', username: 'oscar',   displayName: 'Oscar Reyes'    },
]

interface Props {
  onClose: () => void
}

export default function AddFriendModal({ onClose }: Props) {
  const [search, setSearch] = useState('')
  const [addedIds, setAddedIds] = useState<Set<string>>(new Set())

  const filtered = FAKE_USERS.filter(u =>
    u.username.toLowerCase().includes(search.toLowerCase()) ||
    u.displayName.toLowerCase().includes(search.toLowerCase())
  )

  function handleAdd(id: string) {
    setAddedIds(prev => new Set(prev).add(id))
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span className="modal-title">Add Friend</span>
          <button className="modal-close" onClick={onClose}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <div className="modal-search-row">
          <input
            className="modal-search-input"
            placeholder="Username or email"
            value={search}
            onChange={e => setSearch(e.target.value)}
          />
          <button className="modal-connect-btn">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          </button>
        </div>
        <div className="modal-user-list">
          {filtered.length === 0 ? (
            <div className="modal-empty">No users found</div>
          ) : (
            filtered.map(u => (
              <div key={u.id} className="modal-user-row">
                <div className="avatar" style={{ background: avatarColor(u.username) }}>
                  {avatarInitials(u.displayName, u.username)}
                </div>
                <div className="item-body">
                  <div className="item-name">{u.displayName || u.username}</div>
                  <div className="item-preview">@{u.username}</div>
                </div>
                {addedIds.has(u.id) ? (
                  <span className="modal-added-label">Sent</span>
                ) : (
                  <button className="req-btn accept" onClick={() => handleAdd(u.id)}>
                    <svg className="add-friend-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <line x1="12" y1="5" x2="12" y2="19" />
                      <line x1="5" y1="12" x2="19" y2="12" />
                    </svg>
                    Add
                  </button>
                )}
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  )
}
