/**
 * WebAuthn Integration for Biometric Unlock
 *
 * Strategy: Client-Challenge (no backend challenge needed)
 * - Offline-first: KEK is stored in IndexedDB (from Phase 1)
 * - WebAuthn is used as an alternative unlock method (instead of password)
 * - No additional encryption of KEK with WebAuthn (keeps it simple)
 */

import * as api from '$lib/api';

export interface WebAuthnCredential {
  id: number;
  credential_id: string;
  device_name: string;
  created_at: string;
  last_used_at?: string;
}

/**
 * Check if WebAuthn is supported in the current browser
 */
export function isWebAuthnSupported(): boolean {
  return (
    typeof window !== 'undefined' &&
    window.PublicKeyCredential !== undefined &&
    navigator.credentials !== undefined
  );
}

/**
 * Check if platform authenticator (Touch ID, Face ID) is available
 */
export async function isPlatformAuthenticatorAvailable(): Promise<boolean> {
  if (!isWebAuthnSupported()) {
    return false;
  }

  try {
    return await PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable();
  } catch {
    return false;
  }
}

/**
 * Register a new WebAuthn credential for biometric unlock
 *
 * @param userId - User ID for credential storage
 * @param username - Username for display in biometric prompts
 * @param deviceName - Optional device name (e.g., "MacBook Pro", "iPhone 13")
 * @returns The created credential info
 */
export async function registerWebAuthnCredential(
  userId: number,
  username: string,
  deviceName?: string
): Promise<WebAuthnCredential> {
  if (!isWebAuthnSupported()) {
    throw new Error('WebAuthn is not supported in this browser');
  }

  // Generate a random challenge (client-side challenge, no backend needed)
  const challenge = new Uint8Array(32);
  crypto.getRandomValues(challenge);

  // Convert userId to buffer (required by WebAuthn)
  const userIdBuffer = new TextEncoder().encode(userId.toString());

  // Create credential options
  const publicKeyCredentialCreationOptions: PublicKeyCredentialCreationOptions = {
    challenge,
    rp: {
      name: 'XelaNote',
      id: window.location.hostname,
    },
    user: {
      id: userIdBuffer,
      name: username,
      displayName: username,
    },
    pubKeyCredParams: [
      // Prefer ES256 (ECDSA with SHA-256)
      { alg: -7, type: 'public-key' },
      // Fallback to RS256 (RSA with SHA-256)
      { alg: -257, type: 'public-key' },
    ],
    authenticatorSelection: {
      // Require platform authenticator (Touch ID, Face ID, Windows Hello)
      authenticatorAttachment: 'platform',
      // Require user verification (biometric or PIN)
      userVerification: 'required',
      // Don't require resident key (simpler, works on more devices)
      residentKey: 'discouraged',
    },
    timeout: 60000, // 60 seconds
    attestation: 'none', // We don't need attestation (simpler, more privacy-friendly)
  };

  try {
    // Create the credential
    const credential = (await navigator.credentials.create({
      publicKey: publicKeyCredentialCreationOptions,
    })) as PublicKeyCredential | null;

    if (!credential) {
      throw new Error('Failed to create credential');
    }

    // Extract credential ID (base64url encoded)
    const credentialId = bufferToBase64Url(credential.rawId);

    // Auto-generate device name if not provided
    const finalDeviceName =
      deviceName ||
      `${getBrowserName()} on ${getDeviceType()} (${new Date().toLocaleDateString()})`;

    // Store credential in backend
    const result = await api.addWebAuthnCredential(credentialId, finalDeviceName);

    return result;
  } catch (error) {
    if (error instanceof Error) {
      // Handle common WebAuthn errors
      if (error.name === 'NotAllowedError') {
        throw new Error('Biometric registration was cancelled or not allowed');
      } else if (error.name === 'NotSupportedError') {
        throw new Error('This device does not support biometric authentication');
      } else if (error.name === 'InvalidStateError') {
        throw new Error('This biometric credential is already registered');
      }
    }
    throw error;
  }
}

/**
 * Authenticate with an existing WebAuthn credential
 *
 * @param credentialId - The credential ID to use (optional, will use any available if not specified)
 * @returns True if authentication succeeded
 */
export async function authenticateWithWebAuthn(credentialId?: string): Promise<boolean> {
  if (!isWebAuthnSupported()) {
    throw new Error('WebAuthn is not supported in this browser');
  }

  // Generate a random challenge
  const challenge = new Uint8Array(32);
  crypto.getRandomValues(challenge);

  // Build credential request options
  const publicKeyCredentialRequestOptions: PublicKeyCredentialRequestOptions = {
    challenge,
    timeout: 60000, // 60 seconds
    userVerification: 'required', // Require biometric verification
    rpId: window.location.hostname,
  };

  // If specific credential ID is provided, use it
  if (credentialId) {
    publicKeyCredentialRequestOptions.allowCredentials = [
      {
        id: base64UrlToBuffer(credentialId),
        type: 'public-key',
        transports: ['internal'], // Platform authenticator
      },
    ];
  }

  try {
    // Request authentication
    const credential = (await navigator.credentials.get({
      publicKey: publicKeyCredentialRequestOptions,
    })) as PublicKeyCredential | null;

    if (!credential) {
      return false;
    }

    // Extract credential ID from response
    const usedCredentialId = bufferToBase64Url(credential.rawId);

    // Update last_used_at in backend (fire-and-forget, non-blocking)
    touchCredential(usedCredentialId).catch((err) => {
      console.warn('Failed to update credential last_used_at:', err);
    });

    return true;
  } catch (error) {
    if (error instanceof Error) {
      // Handle common WebAuthn errors
      if (error.name === 'NotAllowedError') {
        // User cancelled or timeout
        return false;
      } else if (error.name === 'NotSupportedError') {
        throw new Error('This device does not support biometric authentication');
      }
    }
    throw error;
  }
}

/**
 * Delete a WebAuthn credential
 *
 * @param credentialId - The credential ID to delete
 */
export async function deleteWebAuthnCredential(credentialId: string): Promise<void> {
  await api.deleteWebAuthnCredential(credentialId);
}

/**
 * Update last_used_at timestamp for a credential (internal)
 */
async function touchCredential(credentialId: string): Promise<void> {
  try {
    await api.touchWebAuthnCredential(credentialId);
  } catch (err) {
    console.warn('Failed to touch WebAuthn credential:', err);
    // Non-fatal error, continue
  }
}

// ============================================================
// Helper Functions
// ============================================================

/**
 * Convert ArrayBuffer to Base64URL string
 */
function bufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  const base64 = btoa(binary);
  // Convert base64 to base64url
  return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

/**
 * Convert Base64URL string to ArrayBuffer
 */
function base64UrlToBuffer(base64url: string): ArrayBuffer {
  // Convert base64url to base64
  const base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
  // Add padding if needed
  const padding = '='.repeat((4 - (base64.length % 4)) % 4);
  const paddedBase64 = base64 + padding;
  // Decode base64 to binary string
  const binary = atob(paddedBase64);
  // Convert binary string to Uint8Array
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * Get browser name for device naming
 */
function getBrowserName(): string {
  const ua = navigator.userAgent;
  if (ua.includes('Chrome') && !ua.includes('Edg')) return 'Chrome';
  if (ua.includes('Safari') && !ua.includes('Chrome')) return 'Safari';
  if (ua.includes('Firefox')) return 'Firefox';
  if (ua.includes('Edg')) return 'Edge';
  return 'Browser';
}

/**
 * Get device type for device naming
 */
function getDeviceType(): string {
  const ua = navigator.userAgent;
  if (ua.includes('iPhone')) return 'iPhone';
  if (ua.includes('iPad')) return 'iPad';
  if (ua.includes('Android')) return 'Android';
  if (ua.includes('Mac')) return 'Mac';
  if (ua.includes('Windows')) return 'Windows';
  if (ua.includes('Linux')) return 'Linux';
  return 'Device';
}
