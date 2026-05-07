const CONFIG_KEY = 'chat_config'

export interface Config {
  gatewayUrl: string
  chatRequestVersion: number
}

export function loadConfig(): Config | null {
  const raw = localStorage.getItem(CONFIG_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as Config
  } catch {
    return null
  }
}

export function saveConfig(config: Config): void {
  localStorage.setItem(CONFIG_KEY, JSON.stringify(config))
}
