import { generateX25519Identity, identityToRecipient, Encrypter, Decrypter } from 'age-encryption'
import { pbkdf2Async } from '@noble/hashes/pbkdf2.js'
import { sha256 } from '@noble/hashes/sha2.js'
import { gcm } from '@noble/ciphers/aes.js'
import { randomBytes } from '@noble/ciphers/utils.js'

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

export async function generateIdentity(): Promise<{ publicKey: string; privateKey: string }> {
  const pk = await generateX25519Identity()
  const pub = await identityToRecipient(pk)
  return { publicKey: pub, privateKey: pk }
}

export async function encrypt(plaintext: string, recipients: string[]): Promise<string> {
  const e = new Encrypter()
  for (const r of recipients) {
    e.addRecipient(r)
  }
  const result = await e.encrypt(plaintext)
  return new TextDecoder().decode(result)
}

export async function decrypt(ciphertext: string): Promise<string> {
  if (!privateKey) throw new Error('private key not available')
  const d = new Decrypter()
  d.addIdentity(privateKey)
  const result = await d.decrypt(new TextEncoder().encode(ciphertext), 'text')
  return result
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
