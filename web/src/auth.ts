const TOKEN_COOKIE = 'token'

function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    return JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
  } catch {
    return null
  }
}

function jwtMaxAge(token: string): number {
  const payload = decodeJwtPayload(token)
  if (!payload || typeof payload.exp !== 'number') return 0
  const secondsLeft = payload.exp - Math.floor(Date.now() / 1000)
  return secondsLeft > 0 ? secondsLeft : 0
}

export function getToken(): string | null {
  const match = document.cookie
    .split('; ')
    .find(row => row.startsWith(TOKEN_COOKIE + '='))
  if (!match) return null
  const token = decodeURIComponent(match.split('=')[1])
  return isTokenExpired(token) ? null : token
}

export function isTokenExpired(token: string): boolean {
  const payload = decodeJwtPayload(token)
  if (!payload || typeof payload.exp !== 'number') return true
  return payload.exp - Math.floor(Date.now() / 1000) <= 0
}

export function getUserId(): string | null {
  const token = getToken()
  if (!token) return null
  const payload = decodeJwtPayload(token)
  return (payload?.sub ?? payload?.user_id ?? null) as string | null
}

export function getUsername(): string | null {
  const token = getToken()
  if (!token) return null
  const payload = decodeJwtPayload(token)
  return (payload?.username ?? payload?.user_name ?? null) as string | null
}

export function setToken(token: string): void {
  const maxAge = jwtMaxAge(token)
  if (maxAge <= 0) return
  const secure = location.protocol === 'https:' ? '; Secure' : ''
  document.cookie =
    `${TOKEN_COOKIE}=${encodeURIComponent(token)}; Max-Age=${maxAge}; SameSite=Strict; Path=/${secure}`
}

export function removeToken(): void {
  document.cookie = `${TOKEN_COOKIE}=; Max-Age=0; SameSite=Strict; Path=/`
}
