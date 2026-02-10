import { request } from './client';
import type { GraphData } from './types';

export async function getGlobalGraph(
  options: {
    folder?: string;
    max_nodes?: number;
  } = {}
): Promise<GraphData> {
  const params = new URLSearchParams();
  if (options.folder) params.set('folder', options.folder);
  if (options.max_nodes) params.set('max_nodes', options.max_nodes.toString());

  const query = params.toString();
  return request(`/graph${query ? '?' + query : ''}`);
}
