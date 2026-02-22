import { writable } from 'svelte/store';

// Toast type definition
export interface Toast {
	id: string;
	message: string;
	type: 'success' | 'error' | 'warning' | 'info';
	duration?: number;
}

// Create a toast store
function createToastStore() {
	const { subscribe, set, update } = writable<Toast[]>([]);

	// Add a new toast
	function add(toast: Omit<Toast, 'id'>) {
		const id = Math.random().toString(36).substring(2, 9);
		const newToast: Toast = { ...toast, id };
		
		update(items => [...items, newToast]);
		
		// Auto-remove toast after duration
		if (toast.duration !== 0) {
			setTimeout(() => remove(id), toast.duration || 5000);
		}
	}

	// Remove a toast by id
	function remove(id: string) {
		update(items => items.filter(item => item.id !== id));
	}

	// Clear all toasts
	function clear() {
		set([]);
	}

	return {
		subscribe,
		add,
		remove,
		clear
	};
}

export const toast = createToastStore();
