/**
 * Desktop Bridge Module
 *
 * Provides a unified interface for desktop-specific functionality
 * that works across Tauri, Electron, and Web environments.
 */

export type { AuthTokens, DesktopBridge } from './interface';
export { getDesktopBridge, getDesktopBridgeSync, resetBridge } from './interface';
