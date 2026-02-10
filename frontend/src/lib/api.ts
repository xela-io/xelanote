// API client for xelanote backend

export type { OfflineNoteContext } from './offline/types';
export * from './api/types';
export * from './api/notes';
export * from './api/search';
export * from './api/folders';
export * from './api/tags';
export * from './api/auth';
export * from './api/uploads';
export * from './api/imports';
export * from './api/trash';
export * from './api/versions';
export * from './api/preferences';
export * from './api/admin';
export * from './api/ai';
export * from './api/sharing';
export * from './api/recipes';
export * from './api/journal';
export * from './api/features';
export * from './api/encryption';
export * from './api/config';
export * from './api/due-dates';
export * from './api/graph';
export * from './api/templates';
export * from './api/snippets';
export { initApiAuth, setOnOfflineEnqueue, refreshWithMutex, ApiError } from './api/client';
