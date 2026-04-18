export interface Friend {
  id: string
  username: string
  displayName: string
  online: boolean
  avatarColor: string
}

export interface Conversation {
  id: string
  name: string
  username: string
  lastMessage: string
  time: string
  unread: number
  online: boolean
  avatarColor: string
}

export interface Message {
  id: string
  text: string
  sender: string
  time: string
  own: boolean
}

export const FAKE_FRIENDS: Friend[] = [
  { id: '1', username: 'alice',   displayName: 'Alice Smith',    online: true,  avatarColor: '#5288c1' },
  { id: '2', username: 'bob',     displayName: 'Bob Johnson',    online: false, avatarColor: '#9c27b0' },
  { id: '3', username: 'carol',   displayName: 'Carol Williams', online: true,  avatarColor: '#2e7d32' },
  { id: '4', username: 'dave',    displayName: 'Dave Brown',     online: false, avatarColor: '#e65100' },
  { id: '5', username: 'emma',    displayName: 'Emma Davis',     online: true,  avatarColor: '#ad1457' },
  { id: '6', username: 'frank',   displayName: 'Frank Miller',   online: false, avatarColor: '#00695c' },
]

export const FAKE_CONVERSATIONS: Conversation[] = [
  { id: '1', name: 'Alice Smith',    username: 'alice',  lastMessage: 'See you tomorrow! 👋',      time: '09:41', unread: 3, online: true,  avatarColor: '#5288c1' },
  { id: '2', name: 'Bob Johnson',    username: 'bob',    lastMessage: 'Sounds good to me',          time: 'Mon',   unread: 0, online: false, avatarColor: '#9c27b0' },
  { id: '3', name: 'Carol Williams', username: 'carol',  lastMessage: 'Did you see the latest PR?', time: '09:12', unread: 1, online: true,  avatarColor: '#2e7d32' },
  { id: '4', name: 'Dave Brown',     username: 'dave',   lastMessage: 'Let me check and get back',  time: 'Sun',   unread: 0, online: false, avatarColor: '#e65100' },
  { id: '5', name: 'Emma Davis',     username: 'emma',   lastMessage: '😂 That\'s hilarious',       time: '08:55', unread: 7, online: true,  avatarColor: '#ad1457' },
]

export const FAKE_MESSAGES: Record<string, Message[]> = {
  '1': [
    { id: 'm1', text: 'Hey! How\'s it going?',                         sender: 'alice', time: '09:30', own: false },
    { id: 'm2', text: 'Pretty good, just working on the new feature.', sender: 'me',    time: '09:31', own: true  },
    { id: 'm3', text: 'Nice! Which one?',                              sender: 'alice', time: '09:31', own: false },
    { id: 'm4', text: 'The chat UI — going for a Telegram vibe 😄',    sender: 'me',    time: '09:33', own: true  },
    { id: 'm5', text: 'Oh that\'s cool! Send me a screenshot when done.',sender:'alice', time: '09:34', own: false },
    { id: 'm6', text: 'Sure! Almost there.',                           sender: 'me',    time: '09:38', own: true  },
    { id: 'm7', text: 'See you tomorrow! 👋',                          sender: 'alice', time: '09:41', own: false },
  ],
  '2': [
    { id: 'm1', text: 'Hey Bob, are you free this week?',   sender: 'me',  time: '13:00', own: true  },
    { id: 'm2', text: 'Yeah, Thursday works.',              sender: 'bob', time: '13:05', own: false },
    { id: 'm3', text: 'Great, let\'s sync at 3pm.',         sender: 'me',  time: '13:06', own: true  },
    { id: 'm4', text: 'Sounds good to me',                  sender: 'bob', time: '13:10', own: false },
  ],
  '3': [
    { id: 'm1', text: 'Did you see the latest PR?',         sender: 'carol', time: '09:10', own: false },
    { id: 'm2', text: 'Not yet, checking now.',             sender: 'me',    time: '09:11', own: true  },
    { id: 'm3', text: 'There\'s a conflict in auth.go',     sender: 'carol', time: '09:12', own: false },
  ],
  '4': [
    { id: 'm1', text: 'Can you review my changes?',         sender: 'me',  time: '14:00', own: true  },
    { id: 'm2', text: 'Let me check and get back',          sender: 'dave', time: '14:20', own: false },
  ],
  '5': [
    { id: 'm1', text: 'Did you hear what happened today?',  sender: 'emma', time: '08:50', own: false },
    { id: 'm2', text: 'No, what?',                          sender: 'me',   time: '08:52', own: true  },
    { id: 'm3', text: 'The deploy went to production at 3am 💀', sender: 'emma', time: '08:53', own: false },
    { id: 'm4', text: 'Oh no 😂',                           sender: 'me',   time: '08:54', own: true  },
    { id: 'm5', text: '😂 That\'s hilarious',               sender: 'emma', time: '08:55', own: false },
  ],
}
