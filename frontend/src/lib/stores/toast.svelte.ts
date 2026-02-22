// Toast store using Svelte 5 runes

const MAX_TOASTS = 3;
const DEFAULT_DURATION = 3000;
const UNDO_DURATION = 8000;

export interface ToastAction {
  label: string;
  handler: () => void;
}

export interface Toast {
  id: string;
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
  duration?: number;
  action?: ToastAction;
}

let toasts = $state<Toast[]>([]);

export const toastState = {
  get toasts() {
    return toasts;
  },
};

export function getToasts(): Toast[] {
  return toasts;
}

export function addToast(toast: Omit<Toast, 'id'>): string {
  const id = Math.random().toString(36).substring(2, 9);
  const newToast: Toast = { ...toast, id };

  toasts = [...toasts, newToast].slice(-MAX_TOASTS);

  const duration = toast.duration ?? DEFAULT_DURATION;
  if (duration > 0) {
    setTimeout(() => removeToast(id), duration);
  }

  return id;
}

export function removeToast(id: string) {
  toasts = toasts.filter((t) => t.id !== id);
}

export function success(message: string): string {
  return addToast({ message, type: 'success' });
}

export function error(message: string): string {
  return addToast({ message, type: 'error' });
}

export function warning(message: string, action?: ToastAction): string {
  return addToast({ message, type: 'warning', action });
}

export function info(message: string): string {
  return addToast({ message, type: 'info' });
}

export function undoToast(message: string, handler: () => Promise<void>): string {
  return addToast({
    message,
    type: 'info',
    duration: UNDO_DURATION,
    action: { label: 'Undo', handler: () => void handler() },
  });
}
