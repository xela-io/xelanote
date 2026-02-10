export interface NavigationGuardDeps {
  autosaveEnabled: () => boolean;
  isDirty: () => boolean;
  confirm: (message: string) => boolean;
  getUnsavedMessage: () => string;
}

export function shouldBlockNavigation(deps: NavigationGuardDeps) {
  if (!deps.autosaveEnabled() && deps.isDirty()) {
    return !deps.confirm(deps.getUnsavedMessage());
  }
  return false;
}
