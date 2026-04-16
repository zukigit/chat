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
