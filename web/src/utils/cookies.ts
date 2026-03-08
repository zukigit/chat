// Cookie utility functions for token management

const TOKEN_COOKIE_NAME = "auth_token";
const TOKEN_EXPIRY_DAYS = 7; // Cookie expiration in days

/**
 * Set a cookie with the given name, value, and expiration
 */
export function setCookie(
  name: string,
  value: string,
  days: number = TOKEN_EXPIRY_DAYS,
): void {
  const expires = new Date();
  expires.setTime(expires.getTime() + days * 24 * 60 * 60 * 1000);

  // Set cookie with security options
  // - Secure: only sent over HTTPS (disabled for localhost dev)
  // - SameSite=Strict: prevent CSRF attacks
  const isSecure = window.location.protocol === "https:";
  const secureFlag = isSecure ? "; Secure" : "";

  document.cookie = `${name}=${encodeURIComponent(value)}; expires=${expires.toUTCString()}; path=/; SameSite=Strict${secureFlag}`;
}

/**
 * Get a cookie value by name
 */
export function getCookie(name: string): string | null {
  const nameEQ = `${name}=`;
  const cookies = document.cookie.split(";");

  for (let cookie of cookies) {
    cookie = cookie.trim();
    if (cookie.startsWith(nameEQ)) {
      return decodeURIComponent(cookie.substring(nameEQ.length));
    }
  }

  return null;
}

/**
 * Delete a cookie by name
 */
export function deleteCookie(name: string): void {
  document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
}

// Token-specific helpers
export function setToken(token: string): void {
  setCookie(TOKEN_COOKIE_NAME, token);
}

export function getToken(): string | null {
  return getCookie(TOKEN_COOKIE_NAME);
}

export function removeToken(): void {
  deleteCookie(TOKEN_COOKIE_NAME);
}
