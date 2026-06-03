const CONFIG_KEY = 'chat_config'

export interface Config {
  gatewayUrl: string
  chatRequestVersion: number
}

export function loadConfig(): Config | null {
  const raw = localStorage.getItem(CONFIG_KEY)
  if (raw) {
    try {
      return JSON.parse(raw) as Config
    } catch {
      // fall through to env var check
    }
  }

  const envUrl = import.meta.env.VITE_GATEWAY_URL
  if (envUrl) {
    const config: Config = { gatewayUrl: envUrl, chatRequestVersion: 1 }
    saveConfig(config)
    return config
  }

  return null
}

export function saveConfig(config: Config): void {
  localStorage.setItem(CONFIG_KEY, JSON.stringify(config))
}
