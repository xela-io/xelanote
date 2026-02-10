// Admin store for managing admin panel state
import type {
  ActivityLog,
  ActivityLogsOptions,
  AdminStats,
  AdminUser,
  DetailedStats,
  SystemSettings,
} from '$lib/api';
import * as api from '$lib/api';

interface AdminState {
  stats: AdminStats | null;
  detailedStats: DetailedStats | null;
  users: AdminUser[];
  activityLogs: ActivityLog[];
  activityTotal: number;
  settings: SystemSettings | null;
  isLoading: boolean;
  error: string | null;
}

const adminState = $state<AdminState>({
  stats: null,
  detailedStats: null,
  users: [],
  activityLogs: [],
  activityTotal: 0,
  settings: null,
  isLoading: false,
  error: null,
});

// Getters
export function getAdminState() {
  return adminState;
}

export function getStats() {
  return adminState.stats;
}

export function getDetailedStats() {
  return adminState.detailedStats;
}

export function getUsers() {
  return adminState.users;
}

export function getActivityLogs() {
  return adminState.activityLogs;
}

export function getActivityTotal() {
  return adminState.activityTotal;
}

export function getSettings() {
  return adminState.settings;
}

export function isLoading() {
  return adminState.isLoading;
}

export function getError() {
  return adminState.error;
}

// Actions
export async function loadStats(): Promise<void> {
  adminState.isLoading = true;
  adminState.error = null;
  try {
    adminState.stats = await api.getAdminStats();
  } catch (err) {
    adminState.error = err instanceof Error ? err.message : 'Failed to load stats';
    throw err;
  } finally {
    adminState.isLoading = false;
  }
}

export async function loadDetailedStats(): Promise<void> {
  adminState.isLoading = true;
  adminState.error = null;
  try {
    adminState.detailedStats = await api.getDetailedStats();
  } catch (err) {
    adminState.error = err instanceof Error ? err.message : 'Failed to load detailed stats';
    throw err;
  } finally {
    adminState.isLoading = false;
  }
}

export async function loadUsers(): Promise<void> {
  adminState.isLoading = true;
  adminState.error = null;
  try {
    adminState.users = await api.getAdminUsers();
  } catch (err) {
    adminState.error = err instanceof Error ? err.message : 'Failed to load users';
    throw err;
  } finally {
    adminState.isLoading = false;
  }
}

export async function loadActivityLogs(options: ActivityLogsOptions = {}): Promise<void> {
  adminState.isLoading = true;
  adminState.error = null;
  try {
    const response = await api.getActivityLogs(options);
    adminState.activityLogs = response.logs || [];
    adminState.activityTotal = response.total;
  } catch (err) {
    adminState.error = err instanceof Error ? err.message : 'Failed to load activity logs';
    throw err;
  } finally {
    adminState.isLoading = false;
  }
}

export async function loadSettings(): Promise<void> {
  adminState.isLoading = true;
  adminState.error = null;
  try {
    adminState.settings = await api.getSystemSettings();
  } catch (err) {
    adminState.error = err instanceof Error ? err.message : 'Failed to load settings';
    throw err;
  } finally {
    adminState.isLoading = false;
  }
}

export async function updateSettings(settings: Partial<SystemSettings>): Promise<void> {
  adminState.isLoading = true;
  adminState.error = null;
  try {
    adminState.settings = await api.updateSystemSettings(settings);
  } catch (err) {
    adminState.error = err instanceof Error ? err.message : 'Failed to update settings';
    throw err;
  } finally {
    adminState.isLoading = false;
  }
}

export async function toggleUserAdmin(userId: number, isAdmin: boolean): Promise<void> {
  adminState.error = null;
  try {
    await api.toggleUserAdmin(userId, isAdmin);
    // Update local state
    const user = adminState.users.find((u) => u.id === userId);
    if (user) {
      user.is_admin = isAdmin;
    }
  } catch (err) {
    adminState.error = err instanceof Error ? err.message : 'Failed to update admin status';
    throw err;
  }
}

export async function deleteUser(userId: number): Promise<void> {
  adminState.error = null;
  try {
    await api.deleteUserAdmin(userId);
    // Remove from local state
    adminState.users = adminState.users.filter((u) => u.id !== userId);
  } catch (err) {
    adminState.error = err instanceof Error ? err.message : 'Failed to delete user';
    throw err;
  }
}

// Reset state (call when leaving admin panel)
export function resetAdminState(): void {
  adminState.stats = null;
  adminState.detailedStats = null;
  adminState.users = [];
  adminState.activityLogs = [];
  adminState.activityTotal = 0;
  adminState.settings = null;
  adminState.isLoading = false;
  adminState.error = null;
}
