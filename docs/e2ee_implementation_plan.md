# E2EE Implementation Plan — zukigit/chat

## Overview

This document outlines the step-by-step plan to add End-to-End Encryption (E2EE) to the
zukigit/chat project. The system uses **OAuth for authentication** and a **user-set PIN** as
the only secret that unlocks the encryption key. The server stores only encrypted blobs and
public keys — it can never decrypt messages.

**Stack:** Go backend · TypeScript frontend · PostgreSQL · NATS · Docker

---

## Architecture Summary

```
User sets PIN once
      │
      └──► PBKDF2 + AES-GCM ──► encrypts ECDH private key ──► server stores encrypted blob
                                                                server stores public key

Per session:
OAuth login ──► fetch encrypted blob ──► user enters PIN ──► decrypt private key
                                                                      │
                                                          import as non-extractable CryptoKey
                                                          into JS memory (not localStorage)
                                                                      │
                                                          PIN and raw key bytes are discarded

Messaging:
Sender fetches recipient public key ──► ECDH derive shared secret ──► AES-GCM encrypt message
                                                                               │
                                                                    send ciphertext to server
                                                          Server stores/forwards ciphertext only
```

---

## Phase 1 — Database (PostgreSQL)

**File:** `backend/sqls/init/` — add a new migration file.

### 1.1 Key columns — already done ✅

The following are already in place:

```
users
  └── encrypted_private_key  TEXT    -- AES-GCM blob, useless without PIN

public_keys
  └── key                    TEXT    -- SPKI base64, safe as plaintext
  └── user_id                FK → users
```

The separate `public_keys` table is a better design than a single column — it naturally
supports multiple keys per user in the future (e.g. per-device keys or key rotation).

**No changes needed to these tables.**

### 1.2 Update messages table

```sql
ALTER TABLE messages
  ADD COLUMN iv TEXT NOT NULL DEFAULT '';
```

Every encrypted message needs its own random IV stored alongside the ciphertext.
The existing `content` column will store the base64 ciphertext instead of plaintext.

---

## Phase 2 — Backend API (Go)

Add two new REST endpoints. The gateway service proxies these to the backend.

### 2.1 Save keys — `POST /api/users/keys`

Called once during signup after key generation on the client.

```
Request body:
{
  "public_key":            "base64 SPKI...",
  "encrypted_private_key": "base64 blob (salt+iv+ciphertext)..."
}

Response:
{ "ok": true }
```

Handler:
```
validate JWT → get user_id
  │
  ├──► UPDATE users SET encrypted_private_key = $1 WHERE id = $2
  └──► INSERT INTO public_keys (user_id, key) VALUES ($1, $2)
       ON CONFLICT (user_id) DO UPDATE SET key = EXCLUDED.key
```

> If `public_keys` has a unique constraint on `user_id`, upsert as above.
> If it supports multiple keys per user (no unique constraint), use plain INSERT.

### 2.2 Fetch own keys — `GET /api/users/keys`

Called on every login to retrieve the encrypted blob and public key for this user.

```
Response:
{
  "public_key":            "base64 SPKI...",   -- from public_keys.key
  "encrypted_private_key": "base64 blob..."    -- from users.encrypted_private_key
}
```

Handler:
```sql
SELECT u.encrypted_private_key, pk.key AS public_key
FROM users u
JOIN public_keys pk ON pk.user_id = u.id
WHERE u.id = $1   -- authenticated user_id from JWT
```

### 2.3 Fetch another user's public key — `GET /api/users/:id/public-key`

Called before encrypting a message to that user.

```
Response:
{ "public_key": "base64 SPKI..." }
```

Handler:
```sql
SELECT key FROM public_keys WHERE user_id = $1
```

### 2.4 Update message handler

The existing send-message handler must stop reading/writing plaintext.
It now accepts `content` (base64 ciphertext) and `iv` (base64) and stores them as-is.
The server performs no decryption — it is a dumb relay.

---

## Phase 3 — Frontend Crypto Module (TypeScript)

**New file:** `web/src/lib/crypto.ts`

This module handles all cryptographic operations using the browser's native **Web Crypto API**
(zero external dependencies).

### 3.1 Key generation (signup only)

```typescript
export async function generateKeyPair(): Promise<CryptoKeyPair> {
  return crypto.subtle.generateKey(
    { name: "ECDH", namedCurve: "P-256" },
    true,          // extractable so we can export and encrypt it
    ["deriveKey"]
  );
}
```

### 3.2 Encrypt private key with PIN

```typescript
export async function encryptPrivateKey(
  privateKeyBuffer: ArrayBuffer,
  pin: string
): Promise<string> {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv   = crypto.getRandomValues(new Uint8Array(12));

  const keyMaterial = await crypto.subtle.importKey(
    "raw", new TextEncoder().encode(pin), "PBKDF2", false, ["deriveKey"]
  );
  const wrappingKey = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt, iterations: 310_000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false, ["encrypt"]
  );

  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv }, wrappingKey, privateKeyBuffer
  );

  // Bundle: salt(16) + iv(12) + ciphertext
  const bundle = new Uint8Array([...salt, ...iv, ...new Uint8Array(ciphertext)]);
  return bufferToBase64(bundle);
}
```

310,000 PBKDF2 iterations is the OWASP 2023 recommendation — it makes brute-forcing a
6-digit PIN take meaningful time even if the encrypted blob is leaked.

### 3.3 Decrypt private key with PIN

```typescript
export async function decryptPrivateKey(
  encryptedBlob: string,
  pin: string
): Promise<ArrayBuffer> {
  const bundle     = base64ToBuffer(encryptedBlob);
  const salt       = bundle.slice(0, 16);
  const iv         = bundle.slice(16, 28);
  const ciphertext = bundle.slice(28);

  const keyMaterial = await crypto.subtle.importKey(
    "raw", new TextEncoder().encode(pin), "PBKDF2", false, ["deriveKey"]
  );
  const wrappingKey = await crypto.subtle.deriveKey(
    { name: "PBKDF2", salt, iterations: 310_000, hash: "SHA-256" },
    keyMaterial,
    { name: "AES-GCM", length: 256 },
    false, ["decrypt"]
  );

  return crypto.subtle.decrypt({ name: "AES-GCM", iv }, wrappingKey, ciphertext);
  // Wrong PIN throws here — catch this and show "Incorrect PIN" to user
}
```

### 3.4 Session key store (in-memory only)

```typescript
// Module-level — lives only for the duration of the browser session
let _privateKey: CryptoKey | null = null;

export async function loadPrivateKey(
  encryptedBlob: string,
  pin: string
): Promise<void> {
  const privateKeyBuffer = await decryptPrivateKey(encryptedBlob, pin);

  // Import as NON-EXTRACTABLE — XSS can use it to decrypt but can never read raw bytes
  _privateKey = await crypto.subtle.importKey(
    "pkcs8",
    privateKeyBuffer,
    { name: "ECDH", namedCurve: "P-256" },
    false,          // ← non-extractable
    ["deriveKey"]
  );
  // raw buffer is now garbage-collectable; PIN was never stored
}

export function getPrivateKey(): CryptoKey {
  if (!_privateKey) throw new Error("Session locked — enter PIN");
  return _privateKey;
}

export function lockSession(): void {
  _privateKey = null;
}
```

### 3.5 Message encryption

```typescript
export async function encryptMessage(
  plaintext: string,
  recipientPublicKeyBase64: string
): Promise<{ ciphertext: string; iv: string }> {
  const recipientPublicKey = await crypto.subtle.importKey(
    "spki",
    base64ToBuffer(recipientPublicKeyBase64),
    { name: "ECDH", namedCurve: "P-256" },
    false, []
  );

  const sharedKey = await crypto.subtle.deriveKey(
    { name: "ECDH", public: recipientPublicKey },
    getPrivateKey(),
    { name: "AES-GCM", length: 256 },
    false, ["encrypt"]
  );

  const iv         = crypto.getRandomValues(new Uint8Array(12));
  const ciphertext = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv },
    sharedKey,
    new TextEncoder().encode(plaintext)
  );

  return { ciphertext: bufferToBase64(ciphertext), iv: bufferToBase64(iv) };
}
```

### 3.6 Message decryption

```typescript
export async function decryptMessage(
  ciphertextBase64: string,
  ivBase64: string,
  senderPublicKeyBase64: string
): Promise<string> {
  const senderPublicKey = await crypto.subtle.importKey(
    "spki",
    base64ToBuffer(senderPublicKeyBase64),
    { name: "ECDH", namedCurve: "P-256" },
    false, []
  );

  const sharedKey = await crypto.subtle.deriveKey(
    { name: "ECDH", public: senderPublicKey },
    getPrivateKey(),
    { name: "AES-GCM", length: 256 },
    false, ["decrypt"]
  );

  const decrypted = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: base64ToBuffer(ivBase64) },
    sharedKey,
    base64ToBuffer(ciphertextBase64)
  );

  return new TextDecoder().decode(decrypted);
}
```

### 3.7 Utilities

```typescript
export const bufferToBase64 = (buf: ArrayBuffer): string =>
  btoa(String.fromCharCode(...new Uint8Array(buf)));

export const base64ToBuffer = (b64: string): Uint8Array =>
  Uint8Array.from(atob(b64), c => c.charCodeAt(0));
```

---

## Phase 4 — Frontend User Flows

### 4.1 Signup flow

```
User completes OAuth signup
        │
        ▼
Show PIN setup screen:
  "Set an encryption PIN — you will enter this once per session"
  [PIN input] [Confirm PIN input] [Continue]
        │
        ▼
generateKeyPair()
exportKey("spki",  keyPair.publicKey)  → publicKeyBuffer
exportKey("pkcs8", keyPair.privateKey) → privateKeyBuffer
encryptPrivateKey(privateKeyBuffer, pin) → encryptedBlob
        │
        ▼
POST /api/users/keys { public_key, encrypted_private_key }
        │
        ▼
loadPrivateKey(encryptedBlob, pin)   ← private key into memory, PIN discarded
        │
        ▼
Enter chat — fully unlocked ✅
```

### 4.2 Login flow (returning user, any device)

```
User completes OAuth login
        │
        ▼
GET /api/users/keys → { encrypted_private_key, public_key }
        │
        ▼
Show PIN entry screen:
  "Enter your encryption PIN to unlock your messages"
  [PIN input] [Unlock]
        │
        ├── Wrong PIN → crypto.subtle.decrypt throws → show "Incorrect PIN"
        └── Correct PIN → loadPrivateKey() → enter chat ✅
```

### 4.3 Sending a message

```
User types message and hits send
        │
        ▼
GET /api/users/:recipientId/public-key  (can be cached per session)
        │
        ▼
encryptMessage(plaintext, recipientPublicKey)
→ { ciphertext, iv }
        │
        ▼
POST /api/messages { content: ciphertext, iv }   ← server sees no plaintext
```

### 4.4 Receiving a message

```
Message arrives via NATS/WebSocket: { content: ciphertext, iv, sender_id }
        │
        ▼
GET /api/users/:senderId/public-key  (cached)
        │
        ▼
decryptMessage(ciphertext, iv, senderPublicKey)
→ plaintext shown in UI
```

---

## Phase 5 — Security Hardening

### 5.1 Content Security Policy (CSP)

Add to the gateway / web server response headers:

```
Content-Security-Policy:
  default-src 'self';
  script-src 'self';
  connect-src 'self' wss://your-domain;
  style-src 'self' 'unsafe-inline';
```

A strong CSP is the primary defence against XSS. Without it, even non-extractable
in-memory keys can be abused by injected scripts.

### 5.2 Public key verification (optional, recommended)

Users should be able to verify each other's key fingerprints out-of-band to prevent
a server-substitution attack (where the server swaps a public key).

```typescript
// Generate a short fingerprint for display in UI
async function keyFingerprint(publicKeyBase64: string): Promise<string> {
  const hash = await crypto.subtle.digest("SHA-256", base64ToBuffer(publicKeyBase64));
  return Array.from(new Uint8Array(hash))
    .slice(0, 8)
    .map(b => b.toString(16).padStart(2, "0"))
    .join(":")
    .toUpperCase();
}
// e.g. "A3:F1:7C:22:09:4B:E8:11"
// Show in contact profile — users can compare over phone/video to verify
```

### 5.3 PIN strength guidance

Enforce a minimum of 6 characters. Encourage alphanumeric PINs in the UI copy.
The 310,000 PBKDF2 iterations compensate for weak PINs but cannot eliminate the risk
if users choose "123456".

### 5.4 What the server must never do

The backend handlers for messages must not attempt to parse or log `content`.
Add a lint/review rule: no decryption logic lives in the backend.

---

## Phase 6 — What Happens on Edge Cases

| Scenario | Behaviour |
|---|---|
| User forgets PIN | Cannot decrypt old messages — no recovery possible. Show clear warning at PIN setup. |
| User clears browser data | In-memory key gone — re-enter PIN at next login (PIN + blob re-fetched from server). |
| User opens a second tab | New tab has no in-memory key — PIN prompt appears for that tab separately. |
| User logs out | Call `lockSession()` to null out the in-memory key immediately. |
| Server is compromised | Attacker gets encrypted blobs + public keys — useless without PINs. Messages remain encrypted. |
| New device | OAuth login → GET /api/users/keys → PIN prompt → unlocked. Works on any device. |

---

## Implementation Order

1. **Database migration** — `messages` table only (`users.encrypted_private_key` and `public_keys.key` already exist)
2. **Backend endpoints** — `POST /keys`, `GET /keys`, `GET /:id/public-key`
3. **Backend message handler** — treat `content` as opaque ciphertext, store `iv`
4. **`web/src/lib/crypto.ts`** — full crypto module
5. **Signup page** — PIN setup UI + key generation flow
6. **Login page** — PIN entry UI + key load flow
7. **Message send** — replace plaintext send with `encryptMessage`
8. **Message receive** — replace plaintext render with `decryptMessage`
9. **CSP headers** — add to gateway config
10. **Key fingerprint UI** — show in contact/profile view

Each step is independently testable. Steps 1–3 can be done without touching the frontend,
and steps 4–8 can be developed against a local backend with placeholder encrypted blobs.

---

## What the Server Knows vs What It Cannot Know

| Data | Server stores | Server can read |
|---|---|---|
| Public keys | ✅ Yes | ✅ Yes (by design — they are public) |
| Encrypted private key blob | ✅ Yes | ❌ No — useless without PIN |
| Message ciphertext | ✅ Yes | ❌ No — useless without private keys |
| Message IV | ✅ Yes | ❌ Not useful alone |
| PIN | ❌ Never stored | ❌ Never |
| Plaintext messages | ❌ Never | ❌ Never |
