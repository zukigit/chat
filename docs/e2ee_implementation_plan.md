# E2EE Implementation Plan — zukigit/chat (age-encryption)

## Overview

This document outlines the step-by-step plan to add End-to-End Encryption (E2EE) to the
zukigit/chat project. The system uses **OAuth for authentication** and a **user-set PIN** as
the only secret that unlocks the encryption key. The server stores only encrypted blobs and
public keys — it can never decrypt messages.

Encryption is handled by **[age](https://age-encryption.org/)** (`filippo.io/age` on the
backend, `age-encryption` npm package on the frontend). age natively supports **multiple
recipients**: a single message is encrypted once and can be decrypted by any of N recipients
without re-encrypting the payload. This makes group/room messaging efficient and future-proof.

**Stack:** Go backend · TypeScript frontend · PostgreSQL · NATS · Docker

---

## Architecture Summary

```
User sets PIN once
      │
      └──► PBKDF2 + AES-GCM ──► encrypts age X25519 identity (private key) ──► server stores encrypted blob
                                                                                  server stores age recipient (public key)

Per session:
OAuth login ──► fetch encrypted blob ──► user enters PIN ──► decrypt age identity
                                                                      │
                                                          hold identity in JS memory only
                                                          (not localStorage, not extractable)
                                                                      │
                                                          PIN and raw key bytes are discarded

Messaging (supports N recipients):
Sender fetches public keys of ALL recipients (including self for sender-copy)
      │
      ▼
age.encrypt(plaintext, [recipient1, recipient2, ...recipientN])
      │                    └─ each recipient gets their own encrypted file key header stanza
      ▼
Single age ciphertext ──► send to server  (server stores/forwards opaque bytes)

Decryption:
Receiver's age identity ──► age.decrypt(ciphertext) ──► plaintext
      (age finds the matching header stanza automatically)
```

---

## Why age Instead of Raw ECDH

| Concern | Raw ECDH (previous plan) | age |
|---|---|---|
| Multiple recipients | Re-encrypt per recipient or complex KEM wrapping | Built-in: one ciphertext, N recipient stanzas |
| Key format | SPKI/PKCS8 base64, manual import | `age1...` bech32 strings, simple API |
| Algorithm agility risk | Manual AES-GCM wiring | age spec is fixed; no negotiation |
| Group/room messages | O(N) encryptions | O(1) encryption, O(1) decryption |
| Forward secrecy (optional) | Manual ephemeral key management | age supports passphrase & X25519 recipients natively |
| Auditability | Custom crypto glue | Audited spec + reference implementations |

---

## Phase 1 — Database (PostgreSQL)

**File:** `backend/sqls/init/` — add a new migration file.

### 1.1 Key columns

Replace ECDH SPKI columns with age-compatible columns. The shape is nearly identical
but the values are age-formatted strings.

```sql
-- users: stores the age identity (private key) encrypted by the user's PIN
ALTER TABLE users
  ADD COLUMN IF NOT EXISTS encrypted_age_identity TEXT;
  -- AES-GCM blob: salt(16) + iv(12) + ciphertext of the raw age identity bytes
  -- Replaces: encrypted_private_key

-- public_keys: stores the age recipient string (public key)
-- Column rename is optional; the stored value changes from SPKI base64 to age1... string
ALTER TABLE public_keys
  RENAME COLUMN key TO age_recipient;
  -- Value example: "age1xy3qfp8z..."
  -- Replaces: base64 SPKI
```

The separate `public_keys` table stays, and its multi-key-per-user design is now even more
useful: each device can have its own age recipient, all added as recipients to every message
sent to that user.

### 1.2 Update messages table

```sql
ALTER TABLE messages
  ADD COLUMN IF NOT EXISTS age_ciphertext TEXT NOT NULL DEFAULT '';
  -- Full age binary payload, base64-encoded.
  -- The IV is embedded inside the age format; no separate iv column needed.
  -- The existing `content` column can be dropped or left for migration purposes.
```

---

## Phase 2 — Backend API (Go)

Add/update REST endpoints. The gateway service proxies these to the backend.

### 2.1 Go dependency

```bash
go get filippo.io/age
```

The backend itself **does not encrypt or decrypt messages** — it is a dumb relay.
`filippo.io/age` is imported only if the backend ever needs to validate key format
or issue keys server-side (not required in this plan).

### 2.2 Save keys — `POST /api/users/keys`

Called once during signup after key generation on the client.

```
Request body:
{
  "age_recipient":           "age1xy3qfp8z...",         -- X25519 public key (bech32)
  "encrypted_age_identity":  "base64 blob (salt+iv+ciphertext of age identity bytes)..."
}

Response:
{ "ok": true }
```

Handler:

```
validate JWT → get user_id
  │
  ├──► UPDATE users SET encrypted_age_identity = $1 WHERE id = $2
  └──► INSERT INTO public_keys (user_id, age_recipient) VALUES ($1, $2)
       ON CONFLICT (user_id) DO UPDATE SET age_recipient = EXCLUDED.age_recipient
```

> For multi-device support (no unique constraint on user_id), use plain INSERT
> and return a key_id so the client can reference individual keys.

### 2.3 Fetch own keys — `GET /api/users/keys`

Called on every login.

```
Response:
{
  "age_recipient":          "age1xy3qfp8z...",
  "encrypted_age_identity": "base64 blob..."
}
```

Handler:

```sql
SELECT u.encrypted_age_identity, pk.age_recipient
FROM users u
JOIN public_keys pk ON pk.user_id = u.id
WHERE u.id = $1   -- authenticated user_id from JWT
```

### 2.4 Fetch another user's public keys — `GET /api/users/:id/public-keys`

Returns **all** age recipients for a user (one per device). The sender must encrypt
to all of them so the recipient can read the message on any device.

```
Response:
{
  "age_recipients": [
    "age1xy3qfp8z...",
    "age1ab7mn2k..."
  ]
}
```

Handler:

```sql
SELECT age_recipient FROM public_keys WHERE user_id = $1
```

### 2.5 Fetch public keys for a room/group — `GET /api/rooms/:id/members/public-keys`

For group chats, returns recipients for all room members at once to avoid N+1 fetches.

```
Response:
{
  "recipients_by_user": {
    "user_42": ["age1xy3qfp8z...", "age1ab7mn2k..."],
    "user_99": ["age1cd8op3j..."]
  }
}
```

### 2.6 Update message handler

Accept `age_ciphertext` (base64 age payload) and store as-is. No decryption on server.

```
Request body:
{
  "room_id":        123,
  "age_ciphertext": "base64 age binary..."
}
```

---

## Phase 3 — Frontend Crypto Module (TypeScript)

**New file:** `web/src/lib/crypto.ts`

Uses the **`age-encryption`** npm package plus the browser's native Web Crypto API for
PIN-based key wrapping.

```bash
npm install age-encryption
```

### 3.1 Key generation (signup only)

```typescript
import * as age from "age-encryption";

export async function generateAgeIdentity(): Promise<age.Identity> {
  // age.generateIdentity() returns a new X25519 identity (private key object)
  // identity.toString() → "AGE-SECRET-KEY-1..."  (bech32, keep secret)
  // identity.recipient().toString() → "age1..."   (bech32, share publicly)
  return age.generateIdentity();
}
```

### 3.2 Encrypt age identity with PIN (key wrapping)

The raw age identity string is encrypted with a PIN-derived AES-GCM key before
being sent to the server. This is identical in structure to the previous plan.

```typescript
export async function encryptAgeIdentity(
  identityStr: string,   // "AGE-SECRET-KEY-1..."
  pin: string
): Promise<string> {
  const identityBytes = new TextEncoder().encode(identityStr);
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
    { name: "AES-GCM", iv }, wrappingKey, identityBytes
  );

  // Bundle: salt(16) + iv(12) + ciphertext
  const bundle = new Uint8Array([...salt, ...iv, ...new Uint8Array(ciphertext)]);
  return bufferToBase64(bundle);
}
```

310,000 PBKDF2 iterations is the OWASP 2023 recommendation — it makes brute-forcing a
6-digit PIN take meaningful time even if the encrypted blob is leaked.

### 3.3 Decrypt age identity with PIN

```typescript
export async function decryptAgeIdentity(
  encryptedBlob: string,
  pin: string
): Promise<string> {
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

  const identityBytes = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv }, wrappingKey, ciphertext
  );
  // Wrong PIN throws here — catch and show "Incorrect PIN" to user
  return new TextDecoder().decode(identityBytes);  // "AGE-SECRET-KEY-1..."
}
```

### 3.4 Session identity store (in-memory only)

```typescript
// Module-level — lives only for the duration of the browser session
let _identity: age.Identity | null = null;

export async function loadAgeIdentity(
  encryptedBlob: string,
  pin: string
): Promise<void> {
  const identityStr = await decryptAgeIdentity(encryptedBlob, pin);
  _identity = age.parseIdentity(identityStr);
  // identityStr is now garbage-collectable; PIN was never stored
}

export function getIdentity(): age.Identity {
  if (!_identity) throw new Error("Session locked — enter PIN");
  return _identity;
}

export function lockSession(): void {
  _identity = null;
}
```

### 3.5 Message encryption (multiple recipients)

```typescript
export async function encryptMessage(
  plaintext: string,
  recipientStrings: string[]   // ["age1xy3qfp8z...", "age1ab7mn2k...", ...]
): Promise<string> {
  // Parse all recipient public keys
  const recipients = recipientStrings.map(r => age.parseRecipient(r));

  // age.encrypt handles the multi-recipient header automatically:
  //   - generates a random file key
  //   - wraps it once per recipient (X25519 ECDH stanza)
  //   - encrypts the payload once with the file key (ChaCha20-Poly1305)
  const encrypter = new age.Encrypter();
  recipients.forEach(r => encrypter.addRecipient(r));

  const plaintextBytes = new TextEncoder().encode(plaintext);
  const ageCiphertext  = await encrypter.encrypt(plaintextBytes);

  return bufferToBase64(ageCiphertext);
}
```

Callers should include **all recipient devices** plus the **sender's own recipient** so the
sender can also read their own sent messages.

### 3.6 Message decryption

```typescript
export async function decryptMessage(
  ageCiphertextBase64: string
): Promise<string> {
  const ciphertextBytes = base64ToBuffer(ageCiphertextBase64);

  // age.decrypt tries each identity stanza in the header until one matches
  const decrypter = new age.Decrypter();
  decrypter.addIdentity(getIdentity());

  const plaintext = await decrypter.decrypt(ciphertextBytes, "uint8array");
  return new TextDecoder().decode(plaintext);
}
```

Note: the sender's public key is **no longer needed** for decryption. age's header
contains all necessary key-agreement information. This simplifies the receive path.

### 3.7 Utilities

```typescript
export const bufferToBase64 = (buf: ArrayBuffer | Uint8Array): string =>
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
generateAgeIdentity()
  → identity            (age.Identity — private)
  → identity.recipient().toString()  → ageRecipient  (public, "age1...")
  → identity.toString()              → identityStr   ("AGE-SECRET-KEY-1...")
        │
        ▼
encryptAgeIdentity(identityStr, pin) → encryptedBlob
        │
        ▼
POST /api/users/keys { age_recipient, encrypted_age_identity: encryptedBlob }
        │
        ▼
loadAgeIdentity(encryptedBlob, pin)   ← identity into memory, PIN discarded
        │
        ▼
Enter chat — fully unlocked ✅
```

### 4.2 Login flow (returning user, any device)

```
User completes OAuth login
        │
        ▼
GET /api/users/keys → { age_recipient, encrypted_age_identity }
        │
        ▼
Show PIN entry screen:
  "Enter your encryption PIN to unlock your messages"
  [PIN input] [Unlock]
        │
        ├── Wrong PIN → AES-GCM decrypt throws → show "Incorrect PIN"
        └── Correct PIN → loadAgeIdentity() → enter chat ✅
```

### 4.3 Sending a message (DM or group)

```
User types message and hits send
        │
        ▼
Collect all recipients:
  GET /api/users/:recipientId/public-keys  → their age recipients (all devices)
  +  own age recipient (from session, for sender-copy)
        │
        ▼
encryptMessage(plaintext, [...recipientAgeStrings, ownAgeRecipient])
→ ageCiphertext  (single blob readable by all recipients)
        │
        ▼
POST /api/messages { room_id, age_ciphertext }   ← server sees no plaintext
```

For group/room messages use `GET /api/rooms/:id/members/public-keys` to fetch all
recipients in one round trip.

### 4.4 Receiving a message

```
Message arrives via NATS/WebSocket: { age_ciphertext, sender_id, room_id }
        │
        ▼
decryptMessage(age_ciphertext)
→ plaintext shown in UI
```

No sender public key fetch needed — age resolves the correct stanza automatically.

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

A strong CSP is the primary defence against XSS. Without it, even in-memory identities
can be abused by injected scripts.

### 5.2 Public key (recipient) verification

Users should be able to verify each other's age recipient fingerprints out-of-band to
prevent a server-substitution attack.

```typescript
export async function recipientFingerprint(ageRecipient: string): Promise<string> {
  const bytes = new TextEncoder().encode(ageRecipient);
  const hash  = await crypto.subtle.digest("SHA-256", bytes);
  return Array.from(new Uint8Array(hash))
    .slice(0, 8)
    .map(b => b.toString(16).padStart(2, "0"))
    .join(":")
    .toUpperCase();
}
// e.g. "A3:F1:7C:22:09:4B:E8:11"
// Show in contact profile — users compare over phone/video to verify
```

### 5.3 PIN strength guidance

Enforce a minimum of 6 characters. Encourage alphanumeric PINs in the UI copy.
The 310,000 PBKDF2 iterations compensate for weak PINs but cannot eliminate the risk
if users choose "123456".

### 5.4 What the server must never do

The backend handlers for messages must not attempt to parse or log `age_ciphertext`.
Add a lint/review rule: no decryption logic lives in the backend.

### 5.5 Recipient list privacy

The age ciphertext header reveals **how many** recipients a message has (one stanza
per recipient) but not **who** they are. For additional metadata privacy, the frontend
may pad the recipient count or use a symmetric re-encryption scheme, but this is out
of scope for initial implementation.

---

## Phase 6 — What Happens on Edge Cases

| Scenario | Behaviour |
|---|---|
| User forgets PIN | Cannot decrypt old messages — no recovery possible. Show clear warning at PIN setup. |
| User clears browser data | In-memory identity gone — re-enter PIN at next login (PIN + blob re-fetched from server). |
| User opens a second tab | New tab has no in-memory identity — PIN prompt appears for that tab separately. |
| User logs out | Call `lockSession()` to null out the in-memory identity immediately. |
| Server is compromised | Attacker gets encrypted blobs + age recipients — useless without PINs. Messages remain encrypted. |
| New device | OAuth login → GET /api/users/keys → PIN prompt → unlocked. New device gets its own age identity; old messages encrypted to old identity are unreadable on the new device unless re-encrypted. |
| New device (re-encryption) | Admin/sender can re-encrypt stored messages to the new device's recipient — out of scope v1. |
| Group member added | Existing messages are not automatically re-encrypted. New member reads only messages sent after joining (standard forward-secrecy tradeoff). |

---

## Implementation Order

1. **Database migration** — rename/add `encrypted_age_identity`, `age_recipient`, `age_ciphertext` columns
2. **Backend endpoints** — `POST /keys`, `GET /keys`, `GET /:id/public-keys`, `GET /rooms/:id/members/public-keys`
3. **Backend message handler** — accept and store `age_ciphertext` as opaque blob; drop `iv` column
4. **`npm install age-encryption`** — add to frontend
5. **`web/src/lib/crypto.ts`** — full crypto module (age identity generation, PIN wrapping, encrypt/decrypt)
6. **Signup page** — PIN setup UI + age identity generation flow
7. **Login page** — PIN entry UI + identity load flow
8. **Message send** — replace plaintext send with `encryptMessage(plaintext, allRecipients)`
9. **Message receive** — replace plaintext render with `decryptMessage(ageCiphertext)`
10. **CSP headers** — add to gateway config
11. **Recipient fingerprint UI** — show in contact/profile view

Each step is independently testable. Steps 1–3 can be done without touching the frontend,
and steps 4–9 can be developed against a local backend with placeholder ciphertexts.

---

## What the Server Knows vs What It Cannot Know

| Data | Server stores | Server can read |
|---|---|---|
| age recipients (public keys) | ✅ Yes | ✅ Yes (by design — they are public) |
| Encrypted age identity blob | ✅ Yes | ❌ No — useless without PIN |
| age ciphertext (messages) | ✅ Yes | ❌ No — useless without identities |
| Number of message recipients | ✅ Yes (header stanza count) | ✅ Yes (metadata only) |
| PIN | ❌ Never stored | ❌ Never |
| Plaintext messages | ❌ Never | ❌ Never |
| age identity (private key) | ❌ Never in plaintext | ❌ Never |