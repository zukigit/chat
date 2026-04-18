import { useState, useEffect } from 'react'
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

  useEffect(() => {
    fetchFriends()
      .then(data => {
        setFriends(
          data
            .filter(f => f.status === 'accepted')
            .map(f => ({ id: f.user_id, username: f.username, displayName: f.display_name }))
        )
        setFriendRequests(
          data
            .filter(f => f.status === 'pending')
            .map(f => ({ id: f.user_id, username: f.username, displayName: f.display_name }))
        )
      })
      .catch(console.error)
  }, [])

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
        {/* Tab bar */}
        <div className="sidebar-tabs">
          <button
            className={`sidebar-tab${tab === 'conversations' ? ' active' : ''}`}
            onClick={() => setTab('conversations')}
          >
            💬 Chats
          </button>
          <button
            className={`sidebar-tab${tab === 'friends' ? ' active' : ''}`}
            onClick={() => setTab('friends')}
          >
            👥 Friends
          </button>
          <button className="logout-btn" onClick={handleLogout} title="Logout" style={{ margin: '0 8px' }}>
            🚪
          </button>
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
          />
        )}
      </div>

      {/* Right panel */}
      <MessagePanel conversation={activeConv} messages={messages} />
    </div>
  )
}

