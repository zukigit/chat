const TOKEN_COOKIE = 'token'

function jwtMaxAge(token: string): number {
  try {
    const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
    const secondsLeft = payload.exp - Math.floor(Date.now() / 1000)
    return secondsLeft > 0 ? secondsLeft : 0
  } catch {
    return 0
  }
}

export function getToken(): string | null {
  const match = document.cookie
    .split('; ')
    .find(row => row.startsWith(TOKEN_COOKIE + '='))
  return match ? decodeURIComponent(match.split('=')[1]) : null
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
