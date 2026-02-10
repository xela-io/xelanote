export interface AuthRedirectDeps {
  authInitialized: boolean;
  isAuthenticated: boolean;
  isPublicRoute: boolean;
}

export function shouldRedirectToLogin(deps: AuthRedirectDeps) {
  if (!deps.authInitialized) return false;
  if (!deps.isAuthenticated && !deps.isPublicRoute) return true;
  return false;
}
