const AVATAR_COLORS = [
  '#5288c1', '#9c27b0', '#2e7d32', '#e65100',
  '#ad1457', '#00695c', '#1565c0', '#6a1b9a',
  '#558b2f', '#d84315', '#4527a0', '#00838f',
]

/** Deterministic color derived from username so it stays consistent. */
export function avatarColor(username: string | undefined): string {
  const u = username ?? ''
  let hash = 0
  for (let i = 0; i < u.length; i++) {
    hash = (hash * 31 + u.charCodeAt(i)) >>> 0
  }
  return AVATAR_COLORS[hash % AVATAR_COLORS.length]
}

/**
 * Returns 2-character initials.
 * Uses displayName if provided, otherwise falls back to username.
 */
export function avatarInitials(displayName: string | undefined, username: string | undefined): string {
  const name = (displayName ?? '').trim() || (username ?? '').trim()
  if (!name) return '?'
  return name.split(' ').map(w => w[0]).join('').slice(0, 2).toUpperCase()
}
