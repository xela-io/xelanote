import type { HandleClientError } from '@sveltejs/kit';

export const handleError: HandleClientError = async ({ error, status, message }) => {
  const errorId = crypto.randomUUID().slice(0, 8);

  // Log with context for debugging (visible in browser console)
  console.error(`[client-error] id=${errorId} status=${status}`, error);

  return {
    message,
    errorId,
  };
};
