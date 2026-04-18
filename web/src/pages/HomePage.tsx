import { useNavigate } from 'react-router-dom'
import { getToken, removeToken } from '../auth'

export default function HomePage() {
  const navigate = useNavigate()

  async function handleLogout() {
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

  return (
    <div>
      <button onClick={handleLogout}>Logout</button>
    </div>
  )
}
