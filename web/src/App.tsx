import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { loadConfig } from './config'

export default function App() {
  const navigate = useNavigate()
  const config = loadConfig()

  useEffect(() => {
    if (!config) navigate('/setup')
  }, [])

  if (!config) return null

  return <h1>Hello World</h1>
}
