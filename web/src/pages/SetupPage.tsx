import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { saveConfig } from '../config'

export default function SetupPage() {
  const [gatewayUrl, setGatewayUrl] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  async function handleConnect() {
    setError('')
    setLoading(true)
    try {
      const url = gatewayUrl.replace(/\/$/, '')
      const res = await fetch(`${url}/version`)
      if (!res.ok) throw new Error(`Server returned ${res.status}`)
      const json = await res.json()
      const version: number = json?.data?.version
      if (!version) throw new Error('Invalid version response')
      saveConfig({ gatewayUrl: url, version })
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Connection failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <h1>Setup</h1>
      <label>
        Gateway URL
        <input
          type="text"
          placeholder="http://localhost:8080"
          value={gatewayUrl}
          onChange={e => setGatewayUrl(e.target.value)}
        />
      </label>
      <button onClick={handleConnect} disabled={loading || !gatewayUrl}>
        {loading ? 'Connecting...' : 'Connect'}
      </button>
      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  )
}
