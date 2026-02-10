import { request } from './client';
import type { CreateTemplateRequest, Template, UpdateTemplateRequest } from './types';

export async function listTemplates(): Promise<{ templates: Template[] }> {
  return request('/templates');
}

export async function getTemplate(id: number): Promise<Template> {
  return request(`/templates/${id}`);
}

export async function createTemplate(data: CreateTemplateRequest): Promise<Template> {
  return request('/templates', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function updateTemplate(id: number, data: UpdateTemplateRequest): Promise<Template> {
  return request(`/templates/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  });
}

export async function deleteTemplate(id: number): Promise<void> {
  return request(`/templates/${id}`, { method: 'DELETE' });
}
