import { x25519 } from '@noble/curves/ed25519.js'
import { sha256 } from '@noble/hashes/sha2.js'
import { hkdf } from '@noble/hashes/hkdf.js'
import { hmac } from '@noble/hashes/hmac.js'
import { pbkdf2Async } from '@noble/hashes/pbkdf2.js'
import { chacha20poly1305 } from '@noble/ciphers/chacha.js'
import { gcm } from '@noble/ciphers/aes.js'
import { randomBytes, concatBytes } from '@noble/ciphers/utils.js'
import { bech32, bech32m, base64 } from '@scure/base'

const AGE_PUBLIC_KEY_PREFIX = 'age1'
const AGE_PRIVATE_KEY_PREFIX = 'AGE-SECRET-KEY-1'
const AGE_HKDF_INFO = 'age-encryption.org/v1/X25519'
const AGE_ARMOR_BEGIN = '-----BEGIN AGE ENCRYPTED FILE-----'
const AGE_ARMOR_END = '-----END AGE ENCRYPTED FILE-----'
const AGE_ARMOR_SEPARATOR = '---'

function encodePublicKey(publicKey: Uint8Array): string {
  return AGE_PUBLIC_KEY_PREFIX + bech32m.encode('age', bech32m.toWords(publicKey))
}

function decodePublicKey(s: string): Uint8Array {
  if (!s.startsWith(AGE_PUBLIC_KEY_PREFIX)) throw new Error('invalid age public key')
  const data = s.slice(AGE_PUBLIC_KEY_PREFIX.length)
  const { prefix, words } = bech32m.decode(data)
  if (prefix !== 'age') throw new Error('invalid age public key prefix')
  return bech32m.fromWords(words)
}

function encodePrivateKey(privateKey: Uint8Array): string {
  const data = concatBytes(new Uint8Array([0x01]), privateKey)
  return AGE_PRIVATE_KEY_PREFIX + bech32.encode('age-secret-key-', bech32.toWords(data))
}

function decodePrivateKey(s: string): Uint8Array {
  if (!s.startsWith(AGE_PRIVATE_KEY_PREFIX)) throw new Error('invalid age private key')
  const data = s.slice(AGE_PRIVATE_KEY_PREFIX.length)
  const { prefix, words } = bech32.decode(data)
  if (prefix !== 'age-secret-key-') throw new Error('invalid age private key prefix')
  const bytes = bech32.fromWords(words)
  if (bytes[0] !== 0x01) throw new Error('invalid age private key version')
  return bytes.slice(1)
}

function base64Encode(data: Uint8Array): string {
  return base64.encode(data)
}

function base64Decode(s: string): Uint8Array {
  return base64.decode(s)
}

function wrapArmor(headerLines: string[], body: Uint8Array): string {
  const headerText = headerLines.join('\n')
  const headerB64 = base64Encode(new TextEncoder().encode(headerText))
  const bodyB64 = base64Encode(body)

  const wrap64 = (s: string) => {
    const lines: string[] = []
    for (let i = 0; i < s.length; i += 64) {
      lines.push(s.slice(i, i + 64))
    }
    return lines.join('\n')
  }

  return `${AGE_ARMOR_BEGIN}\n${wrap64(headerB64)}\n${AGE_ARMOR_SEPARATOR}\n${wrap64(bodyB64)}\n${AGE_ARMOR_END}`
}

function unwrapArmor(armor: string): { headerText: string; body: Uint8Array } {
  const lines = armor.trim().split('\n')
  if (lines[0] !== AGE_ARMOR_BEGIN) throw new Error('invalid age armor: missing begin')
  if (lines[lines.length - 1] !== AGE_ARMOR_END) throw new Error('invalid age armor: missing end')

  const headerLines: string[] = []
  let bodyLines: string[] = []
  let inHeader = true

  for (let i = 1; i < lines.length - 1; i++) {
    if (lines[i] === AGE_ARMOR_SEPARATOR) {
      inHeader = false
      continue
    }
    if (inHeader) {
      headerLines.push(lines[i])
    } else {
      bodyLines.push(lines[i])
    }
  }

  const headerB64 = headerLines.join('')
  const bodyB64 = bodyLines.join('')

  return {
    headerText: new TextDecoder().decode(base64Decode(headerB64)),
    body: base64Decode(bodyB64),
  }
}

let privateKey: string | null = null

export function setPrivateKey(pk: string): void {
  privateKey = pk
}

export function clearPrivateKey(): void {
  privateKey = null
}

export function hasPrivateKey(): boolean {
  return privateKey !== null
}

export function generateIdentity(): { publicKey: string; privateKey: string } {
  const sk = x25519.utils.randomSecretKey()
  const pk = x25519.getPublicKey(sk)
  return { publicKey: encodePublicKey(pk), privateKey: encodePrivateKey(sk) }
}

export async function encrypt(plaintext: string, recipients: string[]): Promise<string> {
  const fileKey = randomBytes(16)
  const nonce = randomBytes(12)

  const headerLines: string[] = []

  for (const recipient of recipients) {
    const recipientPub = decodePublicKey(recipient)
    const ephemeralSk = x25519.utils.randomSecretKey()
    const ephemeralPub = x25519.getPublicKey(ephemeralSk)

    const sharedSecret = x25519.getSharedSecret(ephemeralSk, recipientPub)
    const wrapKey = hkdf(sha256, sharedSecret, new Uint8Array(0), new TextEncoder().encode(AGE_HKDF_INFO), 32)

    const encryptedFileKey = new Uint8Array(16)
    for (let i = 0; i < 16; i++) {
      encryptedFileKey[i] = fileKey[i] ^ wrapKey[i]
    }

    headerLines.push(`-> X25519 ${base64Encode(ephemeralPub)}`)
    headerLines.push(`X25519 ${base64Encode(encryptedFileKey)}`)
  }

  const headerText = headerLines.join('\n')
  const mac = hmac(sha256, fileKey, new TextEncoder().encode(headerText)).slice(0, 16)
  headerLines.push(`--- ${base64Encode(mac)}`)

  const cipher = chacha20poly1305(fileKey, nonce)
  const encrypted = cipher.encrypt(new TextEncoder().encode(plaintext))

  return wrapArmor(headerLines, concatBytes(nonce, encrypted))
}

export async function decrypt(armor: string): Promise<string> {
  if (!privateKey) throw new Error('private key not available')

  const sk = decodePrivateKey(privateKey)
  const { headerText, body } = unwrapArmor(armor)

  const headerLines = headerText.split('\n')
  const macLine = headerLines[headerLines.length - 1]
  if (!macLine.startsWith('--- ')) throw new Error('invalid age header: missing MAC')

  const macB64 = macLine.slice(4)
  const expectedMac = base64Decode(macB64)

  const stanzas: { type: string; args: string[]; body: string }[] = []
  let currentStanza: { type: string; args: string[]; body: string } | null = null

  for (const line of headerLines.slice(0, -1)) {
    if (line.startsWith('-> ')) {
      if (currentStanza) stanzas.push(currentStanza)
      const parts = line.slice(3).split(' ')
      currentStanza = { type: parts[0], args: parts.slice(1), body: '' }
    } else if (currentStanza) {
      const parts = line.split(' ')
      if (parts[0] === currentStanza.type) {
        currentStanza.body = parts.slice(1).join(' ')
      }
    }
  }
  if (currentStanza) stanzas.push(currentStanza)

  let fileKey: Uint8Array | null = null

  for (const stanza of stanzas) {
    if (stanza.type !== 'X25519') continue
    const ephemeralPub = base64Decode(stanza.args[0])
    const encryptedFileKey = base64Decode(stanza.body)

    const sharedSecret = x25519.getSharedSecret(sk, ephemeralPub)
    const wrapKey = hkdf(sha256, sharedSecret, new Uint8Array(0), new TextEncoder().encode(AGE_HKDF_INFO), 32)

    const decryptedFileKey = new Uint8Array(16)
    for (let i = 0; i < 16; i++) {
      decryptedFileKey[i] = encryptedFileKey[i] ^ wrapKey[i]
    }

    const computedMac = hmac(sha256, decryptedFileKey, new TextEncoder().encode(headerText)).slice(0, 16)
    if (computedMac.every((b: number, i: number) => b === expectedMac[i])) {
      fileKey = decryptedFileKey
      break
    }
  }

  if (!fileKey) throw new Error('no matching recipient found')

  const nonce = body.slice(0, 12)
  const ciphertext = body.slice(12)
  const cipher = chacha20poly1305(fileKey, nonce)
  const decrypted = cipher.decrypt(ciphertext)

  return new TextDecoder().decode(decrypted)
}

export async function deriveAESKeyFromPin(pin: string, salt: Uint8Array): Promise<Uint8Array> {
  return pbkdf2Async(sha256, new TextEncoder().encode(pin), salt, { dkLen: 32, c: 600000 })
}

export async function encryptPrivateKey(pk: string, pin: string): Promise<string> {
  const salt = randomBytes(32)
  const key = await deriveAESKeyFromPin(pin, salt)
  const iv = randomBytes(12)
  const encoded = new TextEncoder().encode(pk)
  const cipher = gcm(key, iv)
  const encrypted = cipher.encrypt(encoded)
  return btoa(String.fromCharCode(...salt, ...iv, ...encrypted))
}

export async function decryptPrivateKey(encrypted: string, pin: string): Promise<string> {
  const raw = Uint8Array.from(atob(encrypted), c => c.charCodeAt(0))
  const salt = raw.slice(0, 32)
  const iv = raw.slice(32, 44)
  const ciphertext = raw.slice(44)
  const key = await deriveAESKeyFromPin(pin, salt)
  const cipher = gcm(key, iv)
  const decrypted = cipher.decrypt(ciphertext)
  return new TextDecoder().decode(decrypted)
}
