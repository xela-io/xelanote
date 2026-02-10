import { request } from './client';
import type {
  ActivityLogsOptions,
  ActivityLogsResponse,
  AdminStats,
  AdminUser,
  DetailedStats,
  SystemSettings,
} from './types';

export async function getAdminStats(): Promise<AdminStats> {
  return request('/admin/stats');
}

export async function getDetailedStats(): Promise<DetailedStats> {
  return request('/admin/stats/detailed');
}

export async function getAdminUsers(): Promise<AdminUser[]> {
  return request('/admin/users');
}

export async function getAdminUserDetails(id: number): Promise<AdminUser> {
  return request(`/admin/users/${id}`);
}

export async function toggleUserAdmin(id: number, isAdmin: boolean): Promise<void> {
  return request(`/admin/users/${id}/admin`, {
    method: 'PUT',
    body: JSON.stringify({ is_admin: isAdmin }),
  });
}

export async function deleteUserAdmin(id: number): Promise<void> {
  return request(`/admin/users/${id}`, {
    method: 'DELETE',
  });
}

export async function getActivityLogs(
  options: ActivityLogsOptions = {}
): Promise<ActivityLogsResponse> {
  const params = new URLSearchParams();
  if (options.limit) params.set('limit', options.limit.toString());
  if (options.page) params.set('page', options.page.toString());
  if (options.action) params.set('action', options.action);
  if (options.user_id) params.set('user_id', options.user_id.toString());
  if (options.target_type) params.set('target_type', options.target_type);
  if (options.date_from) params.set('date_from', options.date_from);
  if (options.date_to) params.set('date_to', options.date_to);

  const query = params.toString();
  return request(`/admin/activity${query ? '?' + query : ''}`);
}

export async function getSystemSettings(): Promise<SystemSettings> {
  return request('/admin/settings');
}

export async function updateSystemSettings(
  settings: Partial<SystemSettings>
): Promise<SystemSettings> {
  return request('/admin/settings', {
    method: 'PUT',
    body: JSON.stringify(settings),
  });
}
