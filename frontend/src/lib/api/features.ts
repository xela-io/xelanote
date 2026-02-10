import { request } from './client';
import type { UserFeature } from './types';

/**
 * Get a specific feature configuration for the current user.
 */
export async function getFeature(feature: string): Promise<UserFeature> {
  return request(`/features/${feature}`);
}

/**
 * Set a feature configuration for the current user.
 */
export async function setFeature(
  feature: string,
  enabled: boolean,
  settings?: Record<string, unknown>
): Promise<{ success: boolean }> {
  return request(`/features/${feature}`, {
    method: 'PUT',
    body: JSON.stringify({ enabled, settings }),
  });
}

/**
 * List all feature configurations for the current user.
 */
export async function listFeatures(): Promise<UserFeature[]> {
  return request('/features');
}
