/**
 * Dialog store for managing confirm and alert dialogs
 * Replaces native confirm() and alert() with accessible custom dialogs
 */

export interface ConfirmOptions {
  title?: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'default' | 'danger';
}

export interface AlertOptions {
  title?: string;
  message: string;
  confirmText?: string;
  variant?: 'default' | 'danger' | 'warning';
}

// Confirm dialog state
let confirmState = $state<ConfirmOptions | null>(null);
let confirmResolve: ((value: boolean) => void) | null = null;

// Alert dialog state
let alertState = $state<AlertOptions | null>(null);
let alertResolve: (() => void) | null = null;

/**
 * Show a confirmation dialog (replaces native confirm())
 * @returns Promise that resolves to true if confirmed, false if cancelled
 */
export function confirm(options: ConfirmOptions): Promise<boolean> {
  return new Promise((resolve) => {
    confirmState = options;
    confirmResolve = resolve;
  });
}

/**
 * Resolve the current confirm dialog
 */
export function resolveConfirm(result: boolean) {
  console.log('[dialog] resolveConfirm called with:', result);
  console.log('[dialog] confirmResolve exists:', !!confirmResolve);
  confirmResolve?.(result);
  confirmState = null;
  confirmResolve = null;
}

/**
 * Show an alert dialog (replaces native alert())
 * @returns Promise that resolves when the dialog is dismissed
 */
export function alert(options: AlertOptions): Promise<void> {
  return new Promise((resolve) => {
    alertState = options;
    alertResolve = resolve;
  });
}

/**
 * Resolve the current alert dialog
 */
export function resolveAlert() {
  alertResolve?.();
  alertState = null;
  alertResolve = null;
}

/**
 * Get the current confirm dialog state
 */
export function getConfirmState(): ConfirmOptions | null {
  return confirmState;
}

/**
 * Get the current alert dialog state
 */
export function getAlertState(): AlertOptions | null {
  return alertState;
}
