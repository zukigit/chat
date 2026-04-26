import { useState, useEffect, useRef, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { getToken, removeToken, getUsername } from '../auth'
import ConversationList from '../components/ConversationList'
import FriendsList from '../components/FriendsList'
import ProfilePanel from '../components/ProfilePanel'
import MessagePanel from '../components/MessagePanel'
import '../components/chat.css'
import {
  type Friend,
  type FriendRequest,
} from '../components/fakeData'
import { fetchFriends } from '../api/friendsApi'
import { fetchConversations, createConversation, type ApiConversation } from '../api/conversationsApi'
import { avatarColor, avatarInitials } from '../components/avatarUtils'
import { useChatSession } from '../useChatSession'
import { getMessages, getSentMessages, clearMessages, type StoredMessage, type SentMessage } from '../messageStore'

type Tab = 'conversations' | 'friends' | 'profile'

export default function HomePage() {
  const navigate = useNavigate()
  const [tab, setTab] = useState<Tab>('conversations')
  const [activeConv, setActiveConv] = useState<ApiConversation | null>(null)
  const [conversations, setConversations] = useState<ApiConversation[]>([])
  const [friends, setFriends] = useState<Friend[]>([])
  const [friendRequests, setFriendRequests] = useState<FriendRequest[]>([])
  const [menuOpen, setMenuOpen] = useState(false)
  const [refreshingFriends, setRefreshingFriends] = useState(false)
  const [allMessages, setAllMessages] = useState<Record<number, StoredMessage[]>>({})
  const [sentMessages, setSentMessages] = useState<SentMessage[]>([])
  const menuRef = useRef<HTMLDivElement>(null)

  const handleIncomingMessage = useCallback((msg: StoredMessage) => {
    setAllMessages(prev => {
      const convId = msg.conversation_id
      const existing = prev[convId] ?? []
      if (existing.some(m => m.id === msg.id)) return prev
      return { ...prev, [convId]: [...existing, msg].sort((a, b) => a.id - b.id) }
    })
    setSentMessages(getSentMessages(msg.conversation_id))
  }, [])

  const handleDelivered = useCallback((conversationId: number) => {
    setSentMessages(getSentMessages(conversationId))
  }, [])

  const { connected, error: wsError, retryCountdown, send, markAllRead } = useChatSession(handleIncomingMessage, handleDelivered, () => {
    loadConversations().catch(console.error)
  })

  useEffect(() => {
    loadFriends().catch(console.error)
    loadConversations().catch(console.error)
  }, [])

  useEffect(() => {
    const restored: Record<number, StoredMessage[]> = {}
    conversations.forEach(c => {
      const msgs = getMessages(c.id)
      if (msgs.length > 0) restored[c.id] = msgs
    })
    setAllMessages(restored)
    if (activeConv) {
      setSentMessages(getSentMessages(activeConv.id))
    }
  }, [conversations, activeConv])

  // Mark all received messages as read when a conversation is opened.
  useEffect(() => {
    if (!activeConv) return
    const username = getUsername() ?? ''
    const currentUserId = activeConv.members.find(mem => mem.username === username)?.user_id ?? ''
    const msgs = allMessages[activeConv.id] ?? []
    markAllRead(activeConv.id, msgs, currentUserId)
  }, [activeConv?.id, markAllRead])

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

  async function loadConversations() {
    const data = await fetchConversations()
    setConversations(data)
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
    clearMessages()
    navigate('/login')
  }

  function handleSelectConversation(conv: ApiConversation) {
    setActiveConv(conv)
    setTab('conversations')
  }

  async function handleStartChat(friend: Friend) {
    const existing = conversations.find(c =>
      c.members.some(m => m.username === friend.username)
    )
    if (existing) {
      setActiveConv(existing)
      setTab('conversations')
      return
    }

    const conversationId = await createConversation([friend.username])
    let updated = await fetchConversations()
    setConversations(updated)
    const conv = updated.find(c => c.id === conversationId)
    if (conv) {
      setActiveConv(conv)
      setTab('conversations')
    }
  }

  const messages = activeConv ? (allMessages[activeConv.id] ?? []) : []
  const currentSent = activeConv ? sentMessages.filter(s => s.conversation_id === activeConv.id) : []
  const currentUsername = getUsername() ?? ''

  function handleSendMessage(conversationId: number, content: string, _tempId: string) {
    send(conversationId, content)
    setSentMessages(getSentMessages(conversationId))
  }

  return (
    <div className="chat-layout">
      {/* Sidebar */}
      <div className="chat-sidebar">
        {/* Sidebar header with hamburger menu */}
        <div className="sidebar-header-bar">
          <div className="sidebar-menu-wrap" ref={menuRef}>
            {tab === 'friends' || tab === 'profile' ? (
              <button
                className="icon-btn-circle"
                onClick={() => setTab('conversations')}
                aria-label="Back"
              >
                <svg className="back-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <polyline points="15 18 9 12 15 6" />
                </svg>
              </button>
            ) : (
              <button
                className="icon-btn-circle"
                onClick={() => setMenuOpen(o => !o)}
                aria-label="Menu"
              >
                <span className="hamburger-icon">
                  <span /><span /><span />
                </span>
              </button>
            )}
            {menuOpen && tab !== 'friends' && (
              <div className="dropdown">
                <button className="dropdown-item" onClick={() => { setMenuOpen(false); setTab('profile') }}>
                  <div className="avatar avatar-sm" style={{ background: avatarColor(currentUsername), width: 20, height: 20, fontSize: 9 }}>
                    {avatarInitials(currentUsername, currentUsername)}
                  </div>
                  {currentUsername}
                </button>
                <div className="dropdown-divider" />
                <button className="dropdown-item" onClick={() => { setMenuOpen(false); setTab('friends') }}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                    <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
                    <circle cx="9" cy="7" r="4" />
                    <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
                    <path d="M16 3.13a4 4 0 0 1 0 7.75" />
                  </svg>
                  Friends
                </button>
                <div className="dropdown-divider" />
                <button className="dropdown-item danger" onClick={() => { setMenuOpen(false); handleLogout() }}>
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
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
          {tab === 'conversations' && (
            <div className={`ws-status${connected ? ' connected' : wsError ? ' error' : ''}`} title={wsError ? `Retrying in ${retryCountdown}s` : (connected ? 'Connected' : 'Connecting...')}>
              <span className="ws-dot" />
            </div>
          )}
          {tab === 'friends' && (
            <button
              className={`icon-btn-circle${refreshingFriends ? ' spinning' : ''}`}
              disabled={refreshingFriends}
              title="Refresh"
              onClick={async () => {
                if (refreshingFriends) return
                setRefreshingFriends(true)
                try { await loadFriends() } finally { setRefreshingFriends(false) }
              }}
            >
              <svg className="icon-sm" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="23 4 23 10 17 10" />
                <polyline points="1 20 1 14 7 14" />
                <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10" />
                <path d="M20.49 15a9 9 0 0 1-14.85 3.36L1 14" />
              </svg>
            </button>
          )}
        </div>

        {tab === 'conversations' ? (
          <ConversationList
            conversations={conversations}
            activeId={activeConv?.id ?? null}
            currentUsername={currentUsername}
            messages={allMessages}
            onSelect={handleSelectConversation}
          />
        ) : tab === 'friends' ? (
          <FriendsList
            friends={friends}
            friendRequests={friendRequests}
            onStartChat={handleStartChat}
            onAccepted={handleAccepted}
            onDeclined={handleDeclined}
          />
        ) : (
          <ProfilePanel username={currentUsername} />
        )}
      </div>

      {/* Right panel */}
      <MessagePanel conversation={activeConv} messages={messages} sentMessages={currentSent} currentUsername={currentUsername} onSend={handleSendMessage} />
    </div>
  )
}

