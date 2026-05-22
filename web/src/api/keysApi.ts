import { loadConfig } from '../config'
import { getToken } from '../auth'

export interface MyKeysResponse {
  encrypted_private_key: string
  public_key: string
  is_e2ee_ready: boolean
}

export async function getMyKeys(): Promise<MyKeysResponse> {
  const config = loadConfig()
  const token = getToken()
  if (!config || !token) throw new Error('not authenticated')
  const res = await fetch(`${config.gatewayUrl}/keys/me`, {
    headers: { 'Authorization': `Bearer ${token}` },
  })
  if (!res.ok) {
    const json = await res.json()
    throw new Error(json?.message ?? 'failed to fetch keys')
  }
  const json = await res.json()
  return json.data as MyKeysResponse
}

export async function setupKeys(publicKey: string, encryptedPrivateKey: string): Promise<boolean> {
  const config = loadConfig()
  const token = getToken()
  if (!config || !token) throw new Error('not authenticated')
  const res = await fetch(`${config.gatewayUrl}/keys/setup`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({ public_key: publicKey, encrypted_private_key: encryptedPrivateKey }),
  })
  if (!res.ok) {
    const json = await res.json()
    throw new Error(json?.message ?? 'failed to setup keys')
  }
  const json = await res.json()
  return json.data?.is_e2ee_ready ?? false
}

export async function getPublicKeys(userIds: string[]): Promise<Record<string, string>> {
  const config = loadConfig()
  const token = getToken()
  if (!config || !token) throw new Error('not authenticated')
  const res = await fetch(`${config.gatewayUrl}/keys/batch`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
    body: JSON.stringify({ user_ids: userIds }),
  })
  if (!res.ok) {
    const json = await res.json()
    throw new Error(json?.message ?? 'failed to fetch public keys')
  }
  const json = await res.json()
  return json.data as Record<string, string>
}
