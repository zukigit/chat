import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { loadConfig } from '../config'
import { getToken, removeToken } from '../auth'
import { decryptPrivateKey, setPrivateKey } from '../crypto'
import { getMyKeys } from '../api/keysApi'
import './auth.css'

export default function PinEntryPage() {
  const [pin, setPin] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [encryptedPrivateKey, setEncryptedPrivateKey] = useState('')
  const navigate = useNavigate()

  const [fetchError, setFetchError] = useState('')

  useEffect(() => {
    if (!loadConfig()) navigate('/setup')
    if (!getToken()) navigate('/login')

    async function fetchKeys() {
      try {
        const keys = await getMyKeys()
        if (!keys.is_e2ee_ready) {
          navigate('/key-setup')
          return
        }
        setEncryptedPrivateKey(keys.encrypted_private_key)
        setFetchError('')
      } catch {
        setFetchError('Failed to load keys. Please try again.')
      }
    }
    fetchKeys()
  }, [])

  async function handleUnlock() {
    setError('')
    if (!pin) return
    setLoading(true)
    await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))
    try {
      const privateKey = await decryptPrivateKey(encryptedPrivateKey, pin)
      setPrivateKey(privateKey)
      navigate('/')
    } catch {
      setError('Incorrect PIN. Please try again.')
      setPin('')
      setLoading(false)
    }
  }

  async function handleRetryKeys() {
    setFetchError('')
    try {
      const keys = await getMyKeys()
      if (!keys.is_e2ee_ready) {
        navigate('/key-setup')
        return
      }
      setEncryptedPrivateKey(keys.encrypted_private_key)
    } catch {
      setFetchError('Failed to load keys. Please try again.')
    }
  }

  function handleLogout() {
    removeToken()
    navigate('/login')
  }

  const disabled = loading || !pin

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-header">
          <h1 className="auth-title">Enter PIN</h1>
          <p className="auth-label" style={{ fontSize: 14, color: '#888', marginTop: 8 }}>
            Enter your PIN to unlock your encryption keys for this session.
          </p>
        </div>
        {fetchError ? (
          <>
            <p className="auth-error">{fetchError}</p>
            <button className="auth-button" onClick={handleRetryKeys}>Retry</button>
            <p className="auth-switch">
              <a href="/login" onClick={e => { e.preventDefault(); handleLogout() }}>Log out</a>
            </p>
          </>
        ) : encryptedPrivateKey ? (
          <>
            <div className="auth-fields">
              <div className="auth-input-wrap">
                <span className="auth-label">PIN</span>
                <input
                  className="auth-input"
                  type="password"
                  placeholder="Enter your PIN"
                  value={pin}
                  onChange={e => setPin(e.target.value)}
                  onKeyDown={e => e.key === 'Enter' && !disabled && handleUnlock()}
                  disabled={loading}
                  autoFocus
                />
              </div>
            </div>
            <button className="auth-button" onClick={handleUnlock} disabled={disabled}>
              {loading && <span className="auth-spinner" />}
              {loading ? 'Unlocking...' : 'Unlock'}
            </button>
            {error && <p className="auth-error">{error}</p>}
            <p className="auth-switch">
              Not your account? <a href="/login" onClick={e => { e.preventDefault(); handleLogout() }}>Log out</a>
            </p>
          </>
        ) : (
          <div className="auth-spinner" style={{ width: 32, height: 32, borderWidth: 3, margin: '24px auto' }} />
        )}
      </div>
    </div>
  )
}
