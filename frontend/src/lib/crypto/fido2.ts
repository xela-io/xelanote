/**
 * FIDO2/WebAuthn 2FA Module
 *
 * Separate from webauthn.ts (biometric KEK unlock).
 * This module handles security key registration and authentication for 2FA login.
 */

import {
  type AuthResponse,
  beginFIDO2Auth,
  beginFIDO2Registration,
  finishFIDO2Auth,
  finishFIDO2Registration,
} from '$lib/api';

// Server sends WebAuthn options as JSON with base64url strings instead of ArrayBuffers.
// These types represent the server's JSON format before client-side conversion.
interface ServerCredentialDescriptor {
  id: string;
  type: PublicKeyCredentialType;
  transports?: AuthenticatorTransport[];
}

interface ServerPublicKeyOptions {
  challenge: string;
  user?: { id: string; name: string; displayName: string };
  rp?: { name: string; id?: string };
  pubKeyCredParams?: Array<{ type: string; alg: number }>;
  excludeCredentials?: ServerCredentialDescriptor[];
  allowCredentials?: ServerCredentialDescriptor[];
  timeout?: number;
  authenticatorSelection?: AuthenticatorSelectionCriteria;
  attestation?: AttestationConveyancePreference;
}

type ServerCreationOptions = ServerPublicKeyOptions | { publicKey: ServerPublicKeyOptions };
type ServerRequestOptions = ServerPublicKeyOptions | { publicKey: ServerPublicKeyOptions };

function unwrapServerPublicKeyOptions(
  options: ServerCreationOptions | ServerRequestOptions
): ServerPublicKeyOptions {
  if ('publicKey' in options) {
    return options.publicKey;
  }
  return options;
}

function isServerPublicKeyOptions(value: unknown): value is ServerPublicKeyOptions {
  if (!value || typeof value !== 'object') return false;
  const options = value as { challenge?: unknown; user?: unknown };
  if (typeof options.challenge !== 'string') return false;

  if (options.user !== undefined) {
    if (!options.user || typeof options.user !== 'object') return false;
    const user = options.user as { id?: unknown; name?: unknown; displayName?: unknown };
    if (
      typeof user.id !== 'string' ||
      typeof user.name !== 'string' ||
      typeof user.displayName !== 'string'
    ) {
      return false;
    }
  }

  return true;
}

function parseServerCreationOptions(value: unknown): ServerCreationOptions {
  if (isServerPublicKeyOptions(value)) {
    return value;
  }

  if (value && typeof value === 'object') {
    const wrapped = value as { publicKey?: unknown };
    if (isServerPublicKeyOptions(wrapped.publicKey)) {
      return { publicKey: wrapped.publicKey };
    }
  }

  throw new Error('Ungueltige FIDO2-Registrierungsoptionen vom Server.');
}

function parseServerRequestOptions(value: unknown): ServerRequestOptions {
  if (isServerPublicKeyOptions(value)) {
    return value;
  }

  if (value && typeof value === 'object') {
    const wrapped = value as { publicKey?: unknown };
    if (isServerPublicKeyOptions(wrapped.publicKey)) {
      return { publicKey: wrapped.publicKey };
    }
  }

  throw new Error('Ungueltige FIDO2-Authentifizierungsoptionen vom Server.');
}

/**
 * Check if WebAuthn/FIDO2 is supported in the current browser
 */
export function isFIDO2Supported(): boolean {
  return (
    typeof window !== 'undefined' &&
    window.PublicKeyCredential !== undefined &&
    navigator.credentials !== undefined
  );
}

/**
 * Convert a base64url string to an ArrayBuffer
 */
function base64UrlToBuffer(base64url: string): ArrayBuffer {
  // Add padding if needed
  let base64 = base64url.replace(/-/g, '+').replace(/_/g, '/');
  while (base64.length % 4 !== 0) {
    base64 += '=';
  }
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * Convert an ArrayBuffer to a base64url string
 */
function bufferToBase64Url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.length; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/**
 * Prepare CredentialCreationOptions from server response.
 * The server sends base64url-encoded buffers that need to be converted to ArrayBuffers.
 */
function prepareCreationOptions(options: ServerCreationOptions): CredentialCreationOptions {
  const publicKey = unwrapServerPublicKeyOptions(options);

  return {
    publicKey: {
      ...publicKey,
      challenge: base64UrlToBuffer(publicKey.challenge),
      user: {
        ...publicKey.user!,
        id: base64UrlToBuffer(publicKey.user!.id),
      },
      excludeCredentials: (publicKey.excludeCredentials || []).map(
        (cred: ServerCredentialDescriptor) => ({
          ...cred,
          id: base64UrlToBuffer(cred.id),
        })
      ),
    } as PublicKeyCredentialCreationOptions,
  };
}

/**
 * Prepare CredentialRequestOptions from server response.
 */
function prepareRequestOptions(options: ServerRequestOptions): CredentialRequestOptions {
  const publicKey = unwrapServerPublicKeyOptions(options);

  return {
    publicKey: {
      ...publicKey,
      challenge: base64UrlToBuffer(publicKey.challenge),
      allowCredentials: (publicKey.allowCredentials || []).map(
        (cred: ServerCredentialDescriptor) => ({
          ...cred,
          id: base64UrlToBuffer(cred.id),
        })
      ),
    } as PublicKeyCredentialRequestOptions,
  };
}

/**
 * Serialize a PublicKeyCredential (registration response) to a JSON-safe object.
 */
function serializeRegistrationCredential(credential: PublicKeyCredential): Record<string, unknown> {
  const response = credential.response as AuthenticatorAttestationResponse;
  return {
    id: credential.id,
    rawId: bufferToBase64Url(credential.rawId),
    type: credential.type,
    response: {
      attestationObject: bufferToBase64Url(response.attestationObject),
      clientDataJSON: bufferToBase64Url(response.clientDataJSON),
    },
  };
}

/**
 * Serialize a PublicKeyCredential (authentication response) to a JSON-safe object.
 */
function serializeAuthenticationCredential(
  credential: PublicKeyCredential
): Record<string, unknown> {
  const response = credential.response as AuthenticatorAssertionResponse;
  return {
    id: credential.id,
    rawId: bufferToBase64Url(credential.rawId),
    type: credential.type,
    response: {
      authenticatorData: bufferToBase64Url(response.authenticatorData),
      clientDataJSON: bufferToBase64Url(response.clientDataJSON),
      signature: bufferToBase64Url(response.signature),
      userHandle: response.userHandle ? bufferToBase64Url(response.userHandle) : null,
    },
  };
}

/**
 * Register a new security key for 2FA.
 * @param deviceName - Name for the security key (e.g., "YubiKey")
 * @returns Registration result with optional backup codes
 */
export async function registerSecurityKey(
  deviceName: string = 'Security Key'
): Promise<{ credential_id: number; backup_codes?: string[] }> {
  if (!isFIDO2Supported()) {
    throw new Error('Dein Browser unterstützt keine Security Keys.');
  }

  // Step 1: Get creation options from server (API returns JSON with base64url strings, not ArrayBuffers)
  const serverOptions = parseServerCreationOptions(await beginFIDO2Registration());
  const options = prepareCreationOptions(serverOptions);

  // Step 2: Create credential via browser API
  let credential: Credential | null;
  try {
    credential = await navigator.credentials.create(options);
  } catch (err: unknown) {
    if (err instanceof DOMException) {
      if (err.name === 'NotAllowedError') {
        throw new Error('Vorgang abgebrochen. Bitte versuche es erneut.');
      }
      if (err.name === 'NotSupportedError') {
        throw new Error('Dieses Gerät unterstützt keine Security Keys.');
      }
    }
    throw err;
  }

  if (!credential) {
    throw new Error('Keine Anmeldeinformationen erstellt.');
  }

  // Step 3: Send credential to server
  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error('Ungueltige Registrierungsantwort vom Browser.');
  }
  const serialized = serializeRegistrationCredential(credential);
  return finishFIDO2Registration(deviceName, serialized);
}

/**
 * Authenticate with a security key during login.
 * @param pendingLoginToken - Token from the initial login step
 * @returns Auth response with tokens
 */
export async function authenticateWithSecurityKey(
  pendingLoginToken: string
): Promise<AuthResponse> {
  if (!isFIDO2Supported()) {
    throw new Error('Dein Browser unterstützt keine Security Keys.');
  }

  // Step 1: Get assertion options from server (API returns JSON with base64url strings, not ArrayBuffers)
  const serverOptions = parseServerRequestOptions(await beginFIDO2Auth(pendingLoginToken));
  const options = prepareRequestOptions(serverOptions);

  // Step 2: Get assertion via browser API
  let credential: Credential | null;
  try {
    credential = await navigator.credentials.get(options);
  } catch (err: unknown) {
    if (err instanceof DOMException) {
      if (err.name === 'NotAllowedError') {
        throw new Error('Vorgang abgebrochen. Bitte versuche es erneut.');
      }
      if (err.name === 'NotSupportedError') {
        throw new Error(
          'Dieses Gerät unterstützt keine Security Keys. Verwende stattdessen die Authenticator App.'
        );
      }
    }
    throw err;
  }

  if (!credential) {
    throw new Error('Keine Anmeldeinformationen erhalten.');
  }

  // Step 3: Send assertion to server
  if (!(credential instanceof PublicKeyCredential)) {
    throw new Error('Ungueltige Authentifizierungsantwort vom Browser.');
  }
  const serialized = serializeAuthenticationCredential(credential);
  return finishFIDO2Auth(pendingLoginToken, serialized);
}
