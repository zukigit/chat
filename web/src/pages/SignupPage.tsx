import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { loadConfig } from '../config'
import './auth.css'

export default function SignupPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const config = loadConfig()
  
  useEffect(() => {
    if (!config) navigate('/setup')
  }, [])

  async function handleSignup() {
    setError('')
    if (password !== confirm) {
      setError('Passwords do not match')
      return
    }
    setLoading(true)
    try {
      const config = loadConfig()
      if (!config) {
        navigate('/setup')
        return
      }
      const res = await fetch(`${config.gatewayUrl}/signup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password }),
      })
      const json = await res.json()
      if (!res.ok) {
        setError(json?.message ?? 'Signup failed')
        return
      }
      navigate('/login')
    } catch {
      setError('Could not reach server. Check your connection.')
    } finally {
      setLoading(false)
    }
  }

  const disabled = loading || !username.trim() || !password.trim() || !confirm.trim()

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-header">
          <h1 className="auth-title">Sign Up</h1>
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
            />
          </div>
          <div className="auth-input-wrap">
            <span className="auth-label">Password</span>
            <input
              className="auth-input"
              type="password"
              placeholder="••••••••"
              value={password}
              autoComplete="new-password"
              onChange={e => setPassword(e.target.value)}
            />
          </div>
          <div className="auth-input-wrap">
            <span className="auth-label">Confirm Password</span>
            <input
              className="auth-input"
              type="password"
              placeholder="••••••••"
              value={confirm}
              autoComplete="new-password"
              onChange={e => setConfirm(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && !disabled && handleSignup()}
            />
          </div>
        </div>
        <button className="auth-button" onClick={handleSignup} disabled={disabled}>
          {loading && <span className="auth-spinner" />}
          {loading ? 'Creating account...' : 'Create Account'}
        </button>
        <p className="auth-error">{error}</p>
        <p className="auth-switch">
          Already have an account? <Link to="/login">Log in</Link>
        </p>
      </div>
    </div>
  )
}
