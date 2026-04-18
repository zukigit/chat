import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { loadConfig } from '../config'
import { getToken, setToken } from '../auth'
import './auth.css'

export default function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const config = loadConfig()

  useEffect(() => {
    if (!config) navigate('/setup')
    else if (getToken()) navigate('/')
  }, [])

  async function handleLogin() {
    setError('')
    setLoading(true)
    try {
      const config = loadConfig()
      if (!config) {
        navigate('/setup')
        return
      }
      const res = await fetch(`${config.gatewayUrl}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const json = await res.json()
      if (!res.ok) {
        setError(json?.message ?? 'Login failed')
        return
      }
      setToken(json.data.token)
      navigate('/')
    } catch {
      setError('Could not reach server. Check your connection.')
    } finally {
      setLoading(false)
    }
  }

  const disabled = loading || !username.trim() || !password.trim()

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-header">
          <h1 className="auth-title">Log In</h1>
        </div>
        <div className="auth-fields">
          <div className="auth-input-wrap">
            <span className="auth-label">Username</span>
            <input
              className="auth-input"
              type="text"
              placeholder="your_username"
              value={username}
              autoComplete="username"
              onChange={e => setUsername(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && !disabled && handleLogin()}
            />
          </div>
          <div className="auth-input-wrap">
            <span className="auth-label">Password</span>
            <input
              className="auth-input"
              type="password"
              placeholder="••••••••"
              value={password}
              autoComplete="current-password"
              onChange={e => setPassword(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && !disabled && handleLogin()}
            />
          </div>
        </div>
        <button className="auth-button" onClick={handleLogin} disabled={disabled}>
          {loading && <span className="auth-spinner" />}
          {loading ? 'Signing in...' : 'Sign In'}
        </button>
        {error && <p className="auth-error">{error}</p>}
        <p className="auth-switch">
          Don't have an account? <Link to="/signup">Sign up</Link>
        </p>
      </div>
    </div>
  )
}
