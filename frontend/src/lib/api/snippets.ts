import { request } from './client';
import type { CreateSnippetRequest, Snippet, UpdateSnippetRequest } from './types';

export async function listSnippets(): Promise<{ snippets: Snippet[] }> {
  return request('/snippets');
}

export async function getSnippet(id: number): Promise<Snippet> {
  return request(`/snippets/${id}`);
}

export async function createSnippet(data: CreateSnippetRequest): Promise<Snippet> {
  return request('/snippets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function updateSnippet(id: number, data: UpdateSnippetRequest): Promise<Snippet> {
  return request(`/snippets/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function deleteSnippet(id: number): Promise<void> {
  return request(`/snippets/${id}`, { method: 'DELETE' });
}
