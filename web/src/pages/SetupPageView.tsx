import './SetupPage.css'

interface Props {
  gatewayUrl: string
  loading: boolean
  error: string
  disabled: boolean
  onUrlChange: (url: string) => void
  onConnect: () => void
  onKeyDown: (e: React.KeyboardEvent<HTMLInputElement>) => void
}

function TelegramIcon() {
  return (
    <svg width="36" height="36" viewBox="0 0 24 24" fill="none">
      <path
        d="M12 2C6.477 2 2 6.477 2 12s4.477 10 10 10 10-4.477 10-10S17.523 2 12 2zm4.93 6.858l-1.693 7.98c-.127.565-.46.703-.933.437l-2.578-1.9-1.244 1.197c-.137.138-.253.253-.52.253l.186-2.63 4.796-4.333c.208-.186-.046-.29-.323-.104L7.844 14.6l-2.53-.79c-.55-.172-.56-.55.115-.813l9.621-3.708c.458-.166.859.112.88.569z"
        fill="#ffffff"
      />
    </svg>
  )
}

function ConnectIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 12h14M13 6l6 6-6 6" />
    </svg>
  )
}

export default function SetupPageView({ gatewayUrl, loading, error, disabled, onUrlChange, onConnect, onKeyDown }: Props) {
  return (
    <div className="setup-page">
      <div className="setup-card">
        <div className="setup-icon-wrap"><TelegramIcon /></div>
        <div className="setup-header">
          <h1 className="setup-title">Connect to Server</h1>
          <p className="setup-subtitle">Enter your gateway URL to get started</p>
        </div>
        <div className="setup-input-wrap">
          <span className="setup-label">Gateway URL</span>
          <input
            className="setup-input"
            type="text"
            placeholder="http://localhost:8080"
            value={gatewayUrl}
            onChange={e => onUrlChange(e.target.value)}
            onKeyDown={onKeyDown}
          />
        </div>
        <button className="setup-button" onClick={onConnect} disabled={disabled}>
          {loading ? <span className="setup-spinner" /> : <ConnectIcon />}
          {loading ? 'Connecting...' : 'Connect'}
        </button>
        <p className="setup-error">{error || '\u00a0'}</p>
      </div>
    </div>
  )
}
