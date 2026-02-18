import { request } from './client';
import { withQuery } from './query';
import type { GraphData } from './types';

export async function getGlobalGraph(
  options: {
    folder?: string;
    max_nodes?: number;
  } = {}
): Promise<GraphData> {
  return request(
    withQuery('/graph', (params) => {
      if (options.folder) params.set('folder', options.folder);
      if (options.max_nodes) params.set('max_nodes', options.max_nodes.toString());
    })
  );
}
