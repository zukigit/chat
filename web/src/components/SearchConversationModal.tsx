import { useState, useRef, useEffect } from 'react'
import './chat.css'
import { avatarColor, avatarInitials } from './avatarUtils'
import { getToken } from '../auth'
import { loadConfig } from '../config'
import type { ApiConversation } from '../api/conversationsApi'

interface SearchResult {
  id: number
  is_group: boolean
  name: string
  updated_at: string
  members: { user_id: string; username: string; display_name: string; avatar_url: string }[]
}

interface Props {
  open: boolean
  onClose: () => void
  onSelect: (conv: ApiConversation) => void
}

export default function SearchConversationModal({ open, onClose, onSelect }: Props) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [results, setResults] = useState<SearchResult[]>([])
  const [searching, setSearching] = useState(false)
  const [hasSearched, setHasSearched] = useState(false)
  const currentUsername = useRef('')

  useEffect(() => {
    if (open) {
      setResults([])
      setSearching(false)
      setHasSearched(false)
      if (inputRef.current) inputRef.current.value = ''
      try {
        currentUsername.current = JSON.parse(atob(getToken()!.split('.')[1])).sub ?? ''
      } catch { /* ignore */ }
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
      const res = await fetch(`${gatewayUrl}/conversations/search?name=${encodeURIComponent(query.trim())}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
      const json = await res.json()
      if (json.success && json.data?.conversations) {
        setResults(json.data.conversations)
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

  function getDisplayName(conv: SearchResult) {
    if (conv.is_group) return conv.name
    const other = conv.members.find(m => m.username !== currentUsername.current)
    return other?.display_name || other?.username || ''
  }

  return (
    <div className={`modal-overlay${open ? ' modal-open' : ''}`} onClick={onClose}>
      <div className="modal-content" onClick={e => e.stopPropagation()}>
        <div className="modal-header">
          <span className="modal-title">Search Conversations</span>
          <button className="modal-close" onClick={onClose}>
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <line x1="18" y1="6" x2="6" y2="18" />
              <line x1="6" y1="6" x2="18" y2="18" />
            </svg>
          </button>
        </div>
        <form className="modal-search-row" onSubmit={e => { e.preventDefault(); doSearch() }}>
          <input
            ref={inputRef}
            className="modal-search-input"
            placeholder="Group name or username"
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
            <div className="modal-empty">Enter a name to search</div>
          )}
          {hasSearched && results.length === 0 && (
            <div className="modal-empty">No conversations found</div>
          )}
          {results.map(c => (
            <div
              key={c.id}
              className="modal-user-row modal-user-row-clickable"
              onClick={() => {
                onSelect(c as ApiConversation)
                onClose()
              }}
            >
              <div className="avatar" style={{ background: avatarColor(getDisplayName(c)) }}>
                {avatarInitials(getDisplayName(c), '')}
              </div>
              <div className="item-body">
                <div className="item-name">{getDisplayName(c)}</div>
                <div className="item-preview">{c.is_group ? 'Group' : 'Direct Message'}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
