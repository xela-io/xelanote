/**
 * Environment-aware logger utility.
 *
 * In development: all methods (debug, info, warn, error) forward to console.
 * In production:  only warn and error forward; debug and info are no-ops.
 *
 * Supports tagged logging:
 *   log.debug('[AUTH]', 'Session restored from cookie');
 *   log.error('[WS]', 'Connection failed', error);
 *
 * Usage:
 *   import { log } from '$lib/utils/logger';
 *   log.info('[SYNC]', 'Delta sync completed', { count: 42 });
 */

const isDev: boolean = import.meta.env.DEV;

// eslint-disable-next-line @typescript-eslint/no-empty-function
const noop = (..._args: unknown[]): void => {};

export const log = {
	/** Verbose debugging output. Silenced in production. */
	debug: isDev ? console.debug.bind(console) : noop,

	/** Informational messages. Silenced in production. */
	info: isDev ? console.info.bind(console) : noop,

	/** Warnings — always logged in all environments. */
	warn: console.warn.bind(console),

	/** Errors — always logged in all environments. */
	error: console.error.bind(console)
} as const;
