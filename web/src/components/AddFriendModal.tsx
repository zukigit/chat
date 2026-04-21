import { useState, useRef, useEffect } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import { getToken } from '../auth'
import { loadConfig } from '../config'

interface SearchResultUser {
  user_id: string
  user_name: string
  display_name: string
  avatar_url: string
}

interface Props {
  open: boolean
  onClose: () => void
  onOpen: () => void
}

type RequestState = 'idle' | 'loading' | 'success' | 'error'

function SearchResults({ open }: { open: boolean }) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [results, setResults] = useState<SearchResultUser[]>([])
  const [requestStates, setRequestStates] = useState<Record<string, RequestState>>({})
  const [requestErrors, setRequestErrors] = useState<Record<string, string>>({})
  const [searching, setSearching] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)

  useEffect(() => {
    if (open) {
      setResults([])
      setRequestStates({})
      setRequestErrors({})
      setSearching(false)
      setHasSearched(false)
      if (inputRef.current) inputRef.current.value = ''
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
              {state === 'success' ? (
                <span className="modal-added-label">Sent</span>
              ) : state === 'loading' ? (
                <svg className="search-spinner icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M21 12a9 9 0 1 1-6.22-8.56" />
                </svg>
              ) : (
                <button className="action-btn primary" onClick={() => handleAdd(u.user_name, u.user_id)}>
                  <svg className="icon-xs" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <line x1="12" y1="5" x2="12" y2="19" />
                    <line x1="5" y1="12" x2="19" y2="12" />
                  </svg>
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

export default function AddFriendModal({ open, onClose }: Props) {
  return (
    <div className={`modal-overlay${open ? ' modal-open' : ''}`} onClick={onClose}>
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
        <SearchResults open={open} />
      </div>
    </div>
  )
}
