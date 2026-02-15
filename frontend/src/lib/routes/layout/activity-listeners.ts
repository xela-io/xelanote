/**
 * Manages DOM activity listeners (mousemove, keydown, click, touchstart)
 * for auto-lock and token refresh activity tracking.
 */
export interface ActivityListenersDeps {
  handleActivity: () => void;
}

export function createActivityListeners(deps: ActivityListenersDeps) {
  let registered = false;

  function register() {
    if (registered) return;
    document.addEventListener('mousemove', deps.handleActivity);
    document.addEventListener('keydown', deps.handleActivity);
    document.addEventListener('click', deps.handleActivity);
    document.addEventListener('touchstart', deps.handleActivity);
    registered = true;
    console.log('[Layout] Activity listeners registered');
  }

  function unregister() {
    if (!registered) return;
    document.removeEventListener('mousemove', deps.handleActivity);
    document.removeEventListener('keydown', deps.handleActivity);
    document.removeEventListener('click', deps.handleActivity);
    document.removeEventListener('touchstart', deps.handleActivity);
    registered = false;
    console.log('[Layout] Activity listeners unregistered');
  }

  function isRegistered() {
    return registered;
  }

  return { register, unregister, isRegistered };
}
