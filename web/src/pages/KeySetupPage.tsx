import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { loadConfig } from '../config'
import { getToken } from '../auth'
import { generateIdentity, encryptPrivateKey, setPrivateKey } from '../crypto'
import { setupKeys } from '../api/keysApi'
import './auth.css'

export default function KeySetupPage() {
  const [pin, setPin] = useState('')
  const [confirmPin, setConfirmPin] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    if (!loadConfig()) navigate('/setup')
    if (!getToken()) navigate('/login')
  }, [])

  async function handleSetup() {
    setError('')
    if (pin.length < 4) {
      setError('PIN must be at least 4 characters')
      return
    }
    if (pin !== confirmPin) {
      setError('PINs do not match')
      return
    }
    setLoading(true)
    await new Promise(r => requestAnimationFrame(() => requestAnimationFrame(r)))
    try {
      const identity = await generateIdentity()
      const encryptedPrivateKey = await encryptPrivateKey(identity.privateKey, pin)
      await setupKeys(identity.publicKey, encryptedPrivateKey)
      setPrivateKey(identity.privateKey)
      navigate('/')
    } catch (err: any) {
      setError(err.message ?? 'Failed to set up encryption keys')
      setLoading(false)
    }
  }

  const disabled = loading || pin.length < 4 || pin !== confirmPin

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-header">
          <h1 className="auth-title">Set Up Encryption</h1>
          <p className="auth-label" style={{ fontSize: 14, color: '#888', marginTop: 8 }}>
            Your messages will be end-to-end encrypted. Choose a PIN to protect your private key.
            If you forget your PIN, you cannot recover your messages.
          </p>
        </div>
        <div className="auth-fields">
          <div className="auth-input-wrap">
            <span className="auth-label">PIN</span>
            <input
              className="auth-input"
              type="password"
              placeholder="Enter a PIN (min 4 characters)"
              value={pin}
              onChange={e => setPin(e.target.value)}
              disabled={loading}
            />
          </div>
          <div className="auth-input-wrap">
            <span className="auth-label">Confirm PIN</span>
            <input
              className="auth-input"
              type="password"
              placeholder="Confirm your PIN"
              value={confirmPin}
              onChange={e => setConfirmPin(e.target.value)}
              onKeyDown={e => e.key === 'Enter' && !disabled && handleSetup()}
              disabled={loading}
            />
          </div>
        </div>
        <button className="auth-button" onClick={handleSetup} disabled={disabled}>
          {loading && <span className="auth-spinner" />}
          {loading ? 'Setting up...' : 'Set Up Encryption'}
        </button>
        {error && <p className="auth-error">{error}</p>}
      </div>
    </div>
  )
}
