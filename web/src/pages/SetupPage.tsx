import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { saveConfig } from '../config'
import SetupPageView from './SetupPageView'

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
      if (err instanceof TypeError) {
        setError(`Could not reach server at "${gatewayUrl.replace(/\/$/, '')}". Check the URL and try again.`)
      } else {
        setError(err instanceof Error ? err.message : 'Connection failed')
      }
    } finally {
      setLoading(false)
    }
  }

  const disabled = loading || !gatewayUrl.trim()

  return (
    <SetupPageView
      gatewayUrl={gatewayUrl}
      loading={loading}
      error={error}
      disabled={disabled}
      onUrlChange={setGatewayUrl}
      onConnect={handleConnect}
      onKeyDown={e => e.key === 'Enter' && !disabled && handleConnect()}
    />
  )
}

