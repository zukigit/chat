import { useState, useRef, useEffect } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import { getToken, getUserId } from '../auth'
import { loadConfig } from '../config'
import { acceptFriendRequest, rejectFriendRequest } from '../api/friendsApi'

interface SearchResultUser {
  user_id: string
  user_name: string
  display_name: string
  avatar_url: string
  friendship_status: string
  friendship_initiator_userid: string
}

interface Props {
  open: boolean
  onClose: () => void
  onOpen: () => void
  onStartChat?: (username: string) => Promise<void>
  onAccepted?: (username: string) => void
  onDeclined?: (username: string) => void
}

type RequestState = 'idle' | 'loading' | 'success' | 'error' | 'accepted' | 'declined'

function SearchResults({ open, onStartChat, onAccepted, onDeclined }: { open: boolean; onStartChat?: (username: string) => Promise<void>; onAccepted?: (username: string) => void; onDeclined?: (username: string) => void }) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [results, setResults] = useState<SearchResultUser[]>([])
  const [requestStates, setRequestStates] = useState<Record<string, RequestState>>({})
  const [requestErrors, setRequestErrors] = useState<Record<string, string>>({})
  const [searching, setSearching] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)
  const [chatLoadingId, setChatLoadingId] = useState<string | null>(null)
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const currentUserId = useRef('')

  useEffect(() => {
    if (open) {
      setResults([])
      setRequestStates({})
      setRequestErrors({})
      setSearching(false)
      setHasSearched(false)
      setChatLoadingId(null)
      setActionLoading(null)
      if (inputRef.current) inputRef.current.value = ''
      currentUserId.current = getUserId() ?? ''
    }
  }, [open])

  async function doSearch() {
    const query = inputRef.current?.value ?? ''
    if (!query.trim()) return
    setSearching(true)
    try {
      const config = loadConfig()
      const gatewayUrl = config?.gatewayUrl ?? ''
      const token = getToken()
      const res = await fetch(`${gatewayUrl}/users/search?q=${encodeURIComponent(query.trim())}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      const json = await res.json()
      if (json.success && Array.isArray(json.data)) {
        setResults(json.data)
      } else {
        setResults([])
      }
      setHasSearched(true)
    } catch {
      setResults([])
      setHasSearched(true)
    } finally {
      setSearching(false)
    }
  }

  async function handleAdd(username: string, userId: string) {
    setRequestStates(s => ({ ...s, [userId]: 'loading' }))
    setRequestErrors(e => { const n = { ...e }; delete n[userId]; return n })
    try {
      const config = loadConfig()
      const gatewayUrl = config?.gatewayUrl ?? ''
      const token = getToken()
      const res = await fetch(`${gatewayUrl}/friends/request`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({ username }),
      })
      const json = await res.json()
      if (json.success) {
        setRequestStates(s => ({ ...s, [userId]: 'success' }))
      } else {
        setRequestStates(s => ({ ...s, [userId]: 'error' }))
        setRequestErrors(e => ({ ...e, [userId]: json.message ?? 'Request failed' }))
      }
    } catch {
      setRequestStates(s => ({ ...s, [userId]: 'error' }))
      setRequestErrors(e => ({ ...e, [userId]: 'Network error' }))
    }
  }

  async function handleAccept(username: string, userId: string) {
    if (actionLoading) return
    setActionLoading(`${userId}-accept`)
    setRequestErrors(e => { const n = { ...e }; delete n[userId]; return n })
    try {
      await acceptFriendRequest(username)
      setRequestStates(s => ({ ...s, [userId]: 'accepted' }))
      onAccepted?.(username)
    } catch (err) {
      setRequestErrors(e => ({ ...e, [userId]: (err as Error).message }))
    } finally {
      setActionLoading(null)
    }
  }

  async function handleDecline(username: string, userId: string) {
    if (actionLoading) return
    setActionLoading(`${userId}-decline`)
    setRequestErrors(e => { const n = { ...e }; delete n[userId]; return n })
    try {
      await rejectFriendRequest(username)
      setRequestStates(s => ({ ...s, [userId]: 'declined' }))
      onDeclined?.(username)
    } catch (err) {
      setRequestErrors(e => ({ ...e, [userId]: (err as Error).message }))
    } finally {
      setActionLoading(null)
    }
  }

  async function handleChat(u: SearchResultUser) {
    if (!onStartChat || chatLoadingId) return
    setChatLoadingId(u.user_id)
    try {
      await onStartChat(u.user_name)
    } catch {
      // error handled by parent
    } finally {
      setChatLoadingId(null)
    }
  }

  return (
    <>
      <form className="modal-search-row" onSubmit={e => { e.preventDefault(); doSearch() }}>
        <input
          ref={inputRef}
          className="modal-search-input"
          placeholder="Username or email"
        />
        <button type="submit" className="icon-btn" disabled={searching}>
          {searching ? (
            <svg className="search-spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M21 12a9 9 0 1 1-6.22-8.56" />
            </svg>
          ) : (
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <circle cx="11" cy="11" r="8" />
              <line x1="21" y1="21" x2="16.65" y2="16.65" />
            </svg>
          )}
        </button>
      </form>
      <div className="modal-user-list">
        {!hasSearched && (
          <div className="modal-empty">Enter a username to search</div>
        )}
        {hasSearched && results.length === 0 && (
          <div className="modal-empty">No users found</div>
        )}
        {results.map(u => {
          const state = requestStates[u.user_id] ?? 'idle'
          const isAccepted = u.friendship_status === 'accepted'
          const isPending = u.friendship_status === 'pending'
          const isIncomingRequest = isPending && u.friendship_initiator_userid !== currentUserId.current
          const isOutgoingRequest = isPending && u.friendship_initiator_userid === currentUserId.current
          return (
            <div key={u.user_id} className="modal-user-row">
              <div className="avatar" style={{ background: avatarColor(u.user_name) }}>
                {avatarInitials(u.display_name, u.user_name)}
              </div>
              <div className="item-body">
                <div className="item-name">{u.display_name || u.user_name}</div>
                <div className="item-preview">@{u.user_name}</div>
                {requestErrors[u.user_id] && (
                  <div className="error-text">{requestErrors[u.user_id]}</div>
                )}
              </div>
              {(isAccepted || state === 'accepted') ? (
                <button
                  className="action-btn primary modal-action-fixed"
                  title="Send message"
                  disabled={chatLoadingId !== null}
                  onClick={() => handleChat(u)}
                >
                  {chatLoadingId === u.user_id ? (
                    <svg className="search-spinner icon-xs" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                      <path d="M21 12a9 9 0 1 1-6.22-8.56" />
                    </svg>
                  ) : (
                    'Chat'
                  )}
                </button>
              ) : isIncomingRequest && state !== 'declined' ? (
                <div className="modal-request-actions">
                  <button
                    className="action-btn primary modal-action-sm"
                    onClick={() => handleAccept(u.user_name, u.user_id)}
                    disabled={actionLoading !== null}
                  >
                    {actionLoading === `${u.user_id}-accept` ? '…' : 'Accept'}
                  </button>
                  <button
                    className="action-btn secondary modal-action-sm"
                    onClick={() => handleDecline(u.user_name, u.user_id)}
                    disabled={actionLoading !== null}
                  >
                    {actionLoading === `${u.user_id}-decline` ? '…' : 'Decline'}
                  </button>
                </div>
              ) : isOutgoingRequest || state === 'success' ? (
                <span className="modal-added-label modal-action-fixed">Sent</span>
              ) : state === 'loading' ? (
                <span className="modal-action-fixed" style={{ display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <svg className="search-spinner icon-xs" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M21 12a9 9 0 1 1-6.22-8.56" />
                  </svg>
                </span>
              ) : (
                <button className="action-btn primary modal-action-fixed" onClick={() => handleAdd(u.user_name, u.user_id)}>
                  Add
                </button>
              )}
            </div>
          )
        })}
      </div>
    </>
  )
}

export default function AddFriendModal({ open, onClose, onStartChat, onAccepted, onDeclined }: Props) {
  return (
    <div className={`modal-overlay${open ? ' modal-open' : ''}`} onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span className="modal-title">Search Users</span>
          <button className="modal-close" onClick={onClose}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <SearchResults open={open} onStartChat={onStartChat} onAccepted={onAccepted} onDeclined={onDeclined} />
      </div>
    </div>
  )
}
