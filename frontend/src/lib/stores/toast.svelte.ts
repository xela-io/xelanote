/**
 * Toast Store - Manages toast notifications
 *
 * Provides success, error, info, and warning toasts with optional actions.
 * Toasts auto-close after a specified duration.
 */

export interface Toast {
  id: string;
  type: 'success' | 'error' | 'info' | 'warning';
  message: string;
  duration: number;
  action?: {
    label: string;
    handler: () => void;
  };
}

// State
let toasts = $state<Toast[]>([]);
let nextId = 1;

/**
 * Adds a toast notification.
 */
function addToast(toast: Omit<Toast, 'id'>): string {
  const id = `toast-${nextId++}`;
  const newToast: Toast = { ...toast, id };

  toasts = [...toasts, newToast];

  // Auto-remove after duration
  if (toast.duration > 0) {
    setTimeout(() => {
      removeToast(id);
    }, toast.duration);
  }

  // Limit to max 3 toasts at once
  if (toasts.length > 3) {
    toasts = toasts.slice(-3);
  }

  return id;
}

/**
 * Removes a toast by ID.
 */
export function removeToast(id: string): void {
  toasts = toasts.filter((t) => t.id !== id);
}

/**
 * Shows a success toast.
 */
export function success(message: string, action?: { label: string; handler: () => void }): string {
  return addToast({
    type: 'success',
    message,
    duration: 3000,
    action,
  });
}

/**
 * Shows an error toast.
 */
export function error(message: string, action?: { label: string; handler: () => void }): string {
  return addToast({
    type: 'error',
    message,
    duration: 5000,
    action,
  });
}

/**
 * Shows an info toast.
 */
export function info(message: string, action?: { label: string; handler: () => void }): string {
  return addToast({
    type: 'info',
    message,
    duration: 3000,
    action,
  });
}

/**
 * Shows a warning toast.
 */
export function warning(message: string, action?: { label: string; handler: () => void }): string {
  return addToast({
    type: 'warning',
    message,
    duration: 4000,
    action,
  });
}

/**
 * Shows a special undo toast with longer duration.
 */
export function undoToast(message: string, onUndo: () => void): string {
  return addToast({
    type: 'success',
    message,
    duration: 10000,
    action: {
      label: 'Undo',
      handler: onUndo,
    },
  });
}

// Getter for reactive access
export function getToasts(): Toast[] {
  return toasts;
}

// Derived reactive state
export const toastState = {
  get toasts() {
    return toasts;
  },
};
