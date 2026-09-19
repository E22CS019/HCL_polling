/**
 * Generates (or retrieves from localStorage) a stable browser fingerprint.
 * This is used as an anonymous voter identifier so the same browser can't
 * vote twice on the same poll, even without an account.
 *
 * It is NOT cryptographically secure and can be cleared by the user — it
 * exists purely as a UX convenience to prevent accidental double-votes.
 */

const KEY = 'pollster_voter_fp'

export function getFingerprint(): string {
  let fp = localStorage.getItem(KEY)
  if (!fp) {
    // Generate a random 32-char hex string.
    const buf = new Uint8Array(16)
    crypto.getRandomValues(buf)
    fp = Array.from(buf, b => b.toString(16).padStart(2, '0')).join('')
    localStorage.setItem(KEY, fp)
  }
  return fp
}
