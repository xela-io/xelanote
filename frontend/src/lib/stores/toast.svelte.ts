// Toast store using Svelte 5 runes

const MAX_TOASTS = 3;
const DEFAULT_DURATION = 3000;

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

type ToastOptions = {
  duration?: number;
  action?: ToastAction;
};

let toasts = $state<Toast[]>([]);

export const toastState = {
  get toasts() {
    return toasts;
  },
};

export function getToasts(): Toast[] {
  return toasts;
}

export function addToast(toast: Omit<Toast, 'id'>) {
  const id = Math.random().toString(36).substring(2, 9);
  const newToast: Toast = { ...toast, id };

  toasts = [...toasts, newToast].slice(-MAX_TOASTS);

  const duration = toast.duration ?? DEFAULT_DURATION;
  if (duration > 0) {
    setTimeout(() => removeToast(id), duration);
  }
}

export function removeToast(id: string) {
  toasts = toasts.filter((t) => t.id !== id);
}

export function success(message: string, options?: ToastOptions) {
  addToast({ message, type: 'success', ...options });
}

export function error(message: string, options?: ToastOptions) {
  addToast({ message, type: 'error', ...options });
}

export function warning(message: string, options?: ToastOptions) {
  addToast({ message, type: 'warning', ...options });
}

export function info(message: string, options?: ToastOptions) {
  addToast({ message, type: 'info', ...options });
}
