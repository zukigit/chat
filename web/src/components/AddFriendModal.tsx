import { useState, useCallback, useRef } from 'react'
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

function SearchResults() {
  const inputRef = useRef<HTMLInputElement>(null)
  const [results, setResults] = useState<SearchResultUser[]>([])
  const [addedIds, setAddedIds] = useState<Set<string>>(new Set())
  const [searching, setSearching] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)

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

  function handleAdd(id: string) {
    setAddedIds(prev => new Set(prev).add(id))
  }

  return (
    <>
      <form className="modal-search-row" onSubmit={e => { e.preventDefault(); doSearch() }}>
        <input
          ref={inputRef}
          className="modal-search-input"
          placeholder="Username or email"
        />
        <button type="submit" className="modal-connect-btn" disabled={searching}>
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
        {results.map(u => (
          <div key={u.user_id} className="modal-user-row">
            <div className="avatar" style={{ background: avatarColor(u.user_name) }}>
              {avatarInitials(u.display_name, u.user_name)}
            </div>
            <div className="item-body">
              <div className="item-name">{u.display_name || u.user_name}</div>
              <div className="item-preview">@{u.user_name}</div>
            </div>
            {addedIds.has(u.user_id) ? (
              <span className="modal-added-label">Sent</span>
            ) : (
              <button className="req-btn accept" onClick={() => handleAdd(u.user_id)}>
                <svg className="add-friend-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="12" y1="5" x2="12" y2="19" />
                  <line x1="5" y1="12" x2="19" y2="12" />
                </svg>
                Add
              </button>
            )}
          </div>
        ))}
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
        <SearchResults />
      </div>
    </div>
  )
}
