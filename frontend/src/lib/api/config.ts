import { getApiBaseUrl } from '../config';
import { request } from './client';
import type { AppConfig } from './types';

export function getExportUrl(): string {
  return `${getApiBaseUrl()}/export/markdown`;
}

export async function getConfig(): Promise<AppConfig> {
  return request('/config');
}

export async function getChangelog(): Promise<string> {
  const response = await fetch(`${getApiBaseUrl()}/changelog`, {
    credentials: 'include',
  });
  if (!response.ok) {
    throw new Error('Failed to fetch changelog');
  }
  return response.text();
}
