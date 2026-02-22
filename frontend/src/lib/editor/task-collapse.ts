import { writable } from 'svelte/store';

// Store for tracking collapsed state of tasks
export const collapsedTasks = writable<Set<string>>(new Set());

// Toggle the collapsed state of a task
export function toggleTaskCollapse(taskId: string) {
	collapsedTasks.update(current => {
		const newSet = new Set(current);
		if (newSet.has(taskId)) {
			newSet.delete(taskId);
		} else {
			newSet.add(taskId);
		}
		return newSet;
	});
}

// Collapse a task
export function collapseTask(taskId: string) {
	collapsedTasks.update(current => {
		const newSet = new Set(current);
		newSet.add(taskId);
		return newSet;
	});
}

// Expand a task
export function expandTask(taskId: string) {
	collapsedTasks.update(current => {
		const newSet = new Set(current);
		newSet.delete(taskId);
		return newSet;
	});
}

// Check if a task is collapsed
export function isTaskCollapsed(taskId: string): boolean {
	let result = false;
	collapsedTasks.subscribe((tasks) => {
		result = tasks.has(taskId);
	})();
	return result;
}
