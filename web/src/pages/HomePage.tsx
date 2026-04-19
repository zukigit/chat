import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { getToken, removeToken } from '../auth'
import ConversationList from '../components/ConversationList'
import FriendsList from '../components/FriendsList'
import MessagePanel from '../components/MessagePanel'
import '../components/chat.css'
import {
  FAKE_CONVERSATIONS,
  FAKE_MESSAGES,
  type Conversation,
  type Friend,
  type FriendRequest,
} from '../components/fakeData'
import { fetchFriends } from '../api/friendsApi'

type Tab = 'conversations' | 'friends'

export default function HomePage() {
  const navigate = useNavigate()
  const [tab, setTab] = useState<Tab>('conversations')
  const [activeConv, setActiveConv] = useState<Conversation | null>(null)
  const [friends, setFriends] = useState<Friend[]>([])
  const [friendRequests, setFriendRequests] = useState<FriendRequest[]>([])
  const [menuOpen, setMenuOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => { loadFriends().catch(console.error) }, [])

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    if (menuOpen) document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [menuOpen])

  async function loadFriends() {
    const data = await fetchFriends()
    setFriends(data.filter(f => f.status === 'accepted').map(f => ({ id: f.user_id, username: f.username, displayName: f.display_name })))
    setFriendRequests(data.filter(f => f.status === 'pending').map(f => ({ id: f.user_id, username: f.username, displayName: f.display_name })))
  }

  function handleAccepted(req: FriendRequest) {
    setFriendRequests(prev => prev.filter(r => r.id !== req.id))
    setFriends(prev => [...prev, { id: req.id, username: req.username, displayName: req.displayName }])
  }

  function handleDeclined(req: FriendRequest) {
    setFriendRequests(prev => prev.filter(r => r.id !== req.id))
  }

  async function handleLogout() {
    if (!confirm('Are you sure you want to logout?')) return
    const token = getToken()
    if (token) {
      const config = JSON.parse(localStorage.getItem('chat_config') ?? '{}')
      if (config.gatewayUrl) {
        await fetch(`${config.gatewayUrl}/logout`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
        }).catch(() => {})
      }
    }
    removeToken()
    navigate('/login')
  }

  function handleSelectConversation(conv: Conversation) {
    setActiveConv(conv)
    setTab('conversations')
  }

  function handleStartChat(friend: Friend) {
    const existing = FAKE_CONVERSATIONS.find(c => c.username === friend.username)
    if (existing) {
      setActiveConv(existing)
      setTab('conversations')
    }
  }

  const messages = activeConv ? (FAKE_MESSAGES[activeConv.id] ?? []) : []

  return (
    <div className="chat-layout">
      {/* Sidebar */}
      <div className="chat-sidebar">
        {/* Sidebar header with hamburger menu */}
        <div className="sidebar-header-bar">
          <div className="sidebar-menu-wrap" ref={menuRef}>
            {tab === 'friends' ? (
              <button
                className="hamburger-btn"
                onClick={() => setTab('conversations')}
                aria-label="Back"
              >
                <svg className="back-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="15 18 9 12 15 6" />
                </svg>
              </button>
            ) : (
              <button
                className="hamburger-btn"
                onClick={() => setMenuOpen(o => !o)}
                aria-label="Menu"
              >
                <span className="hamburger-icon">
                  <span /><span /><span />
                </span>
              </button>
            )}
            {menuOpen && tab !== 'friends' && (
              <div className="menu-dropdown">
                <button className="menu-item" onClick={() => { setMenuOpen(false); setTab('friends') }}>
                  <svg className="menu-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                    <circle cx="9" cy="7" r="4" />
                    <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                    <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                  </svg>
                  Friends
                </button>
                <div className="menu-divider" />
                <button className="menu-item menu-item-danger" onClick={() => { setMenuOpen(false); handleLogout() }}>
                  <svg className="menu-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
                    <polyline points="16 17 21 12 16 7" />
                    <line x1="21" y1="12" x2="9" y2="12" />
                  </svg>
                  Logout
                </button>
              </div>
            )}
          </div>
          <input
            className="sidebar-search-inline"
            placeholder={tab === 'friends' ? 'Search Friends' : 'Search'}
            readOnly
          />
        </div>

        {tab === 'conversations' ? (
          <ConversationList
            conversations={FAKE_CONVERSATIONS}
            activeId={activeConv?.id ?? null}
            onSelect={handleSelectConversation}
          />
        ) : (
          <FriendsList
            friends={friends}
            friendRequests={friendRequests}
            onStartChat={handleStartChat}
            onAccepted={handleAccepted}
            onDeclined={handleDeclined}
            onRefresh={loadFriends}
          />
        )}
      </div>

      {/* Right panel */}
      <MessagePanel conversation={activeConv} messages={messages} />
    </div>
  )
}

