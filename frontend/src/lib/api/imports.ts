import { request } from './client';
import type { ImportFile, ImportResult } from './types';

export async function importMarkdown(
  files: ImportFile[],
  preserveStructure = true
): Promise<ImportResult> {
  return request('/import/markdown', {
    method: 'POST',
    body: JSON.stringify({
      files,
      preserve_structure: preserveStructure,
    }),
  });
}
