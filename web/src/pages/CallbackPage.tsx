import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { loadConfig } from '../config'
import { setToken } from '../auth'
import { getMyKeys } from '../api/keysApi'

export default function CallbackPage() {
  const [searchParams] = useSearchParams()
  const [error, setError] = useState('')
  const navigate = useNavigate()

  useEffect(() => {
    const urlError = searchParams.get('error')
    if (urlError) {
      setError(decodeURIComponent(urlError))
      return
    }

    const shortLivedToken = searchParams.get('token')
    if (!shortLivedToken) {
      setError('Missing token parameter')
      return
    }

    async function exchangeToken() {
      const config = loadConfig()
      if (!config) {
        setError('Configuration not found. Please set up the gateway URL.')
        return
      }

      try {
        const res = await fetch(`${config.gatewayUrl}/token/exchange`, {
          method: 'POST',
          headers: { 'Authorization': `Bearer ${shortLivedToken}` },
        })
        const json = await res.json()
        if (!res.ok) {
          setError(json?.message ?? 'Token exchange failed')
          return
        }
        setToken(json.data.token)
        window.history.replaceState({}, '', '/callback')

        const keys = await getMyKeys()
        if (keys.is_e2ee_ready) {
          navigate('/pin-entry')
        } else {
          navigate('/key-setup')
        }
      } catch {
        setError('Could not reach server. Check your connection.')
      }
    }

    exchangeToken()
  }, [])

  if (error) {
    return (
      <div className="auth-page">
        <div className="auth-card">
          <div className="auth-header">
            <h1 className="auth-title">Authentication Error</h1>
          </div>
          <p className="auth-error">{error}</p>
          <button className="auth-button" onClick={() => navigate('/login')}>
            Back to Login
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="auth-page">
      <div className="auth-card">
        <div className="auth-header">
          <h1 className="auth-title">Signing in...</h1>
        </div>
        <div className="auth-spinner" style={{ width: 32, height: 32, borderWidth: 3 }} />
      </div>
    </div>
  )
}
