import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { loadConfig } from '../config'
import { getToken, setToken } from '../auth'
import './auth.css'

export default function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [githubError, setGithubError] = useState('')
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
        <div className="auth-divider"><span>or</span></div>
        <button
          className="auth-social-button"
          onClick={async () => {
            setGithubError('')
            const cfg = loadConfig()
            if (!cfg) {
              setGithubError('Configuration not found. Please set up the gateway URL.')
              return
            }
            try {
              const res = await fetch(`${cfg.gatewayUrl}/oauth/github/url`, { method: 'POST' })
              const json = await res.json()
              if (!res.ok) {
                setGithubError(json?.message ?? 'Failed to start GitHub authentication')
                return
              }
              if (json?.data?.url) {
                window.location.href = json.data.url
              } else {
                setGithubError('No authorization URL returned by server')
              }
            } catch {
              setGithubError('Could not reach server. Check your connection.')
            }
          }}
        >
          <svg viewBox="0 0 24 24" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/></svg>
          Continue with GitHub
        </button>
        {githubError && <p className="auth-error">{githubError}</p>}
        <p className="auth-switch">
          Don't have an account? <Link to="/signup">Sign up</Link>
        </p>
      </div>
    </div>
  )
}
