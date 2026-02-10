export type SecurityLevel = 'paranoid' | 'balanced' | 'convenient';

export interface SecurityPreferencesDeps<TPrefs> {
  getPreferences: () => Promise<TPrefs>;
  setSecurityLevel: (level: SecurityLevel) => void;
  setAutoLockTimeout: (timeout: number) => void;
  setWebAuthnCredentials: (credentials: unknown[]) => void;
}

export async function loadSecurityPreferences<TPrefs>(deps: SecurityPreferencesDeps<TPrefs>) {
  try {
    const prefs = await deps.getPreferences();
    deps.setSecurityLevel(
      (prefs as { security_level?: SecurityLevel }).security_level ?? 'balanced'
    );
    deps.setAutoLockTimeout((prefs as { auto_lock_timeout?: number }).auto_lock_timeout ?? 0);
    deps.setWebAuthnCredentials(
      (prefs as { webauthn_credentials?: unknown[] }).webauthn_credentials ?? []
    );
  } catch (err) {
    console.error('Failed to load security preferences:', err);
  }
}

export interface SecurityLevelChangeDeps {
  getIsSaving: () => boolean;
  setIsSaving: (value: boolean) => void;
  getSecurityLevel: () => SecurityLevel;
  getAutoLockTimeout: () => number;
  setSecurityLevel: (level: SecurityLevel) => void;
  confirm: (options: {
    title: string;
    message: string;
    confirmText: string;
    cancelText: string;
    variant: 'danger' | 'default';
  }) => Promise<boolean>;
  updateSecurityLevel: (level: SecurityLevel) => Promise<void>;
  updateSecurityPreferences: (prefs: { security_level: SecurityLevel }) => Promise<boolean>;
  stopAutoLock: () => void;
  initAutoLock: (timeout: number) => void;
  texts: {
    confirmTitle: string;
    confirmCancel: string;
    confirmLabel: string;
    confirmParanoid: string;
    confirmBalanced: string;
  };
}

export async function handleSecurityLevelChange(
  newLevel: SecurityLevel,
  deps: SecurityLevelChangeDeps
) {
  if (deps.getIsSaving()) return;

  let confirmMessage = '';
  const currentLevel = deps.getSecurityLevel();
  if (newLevel === 'paranoid' && currentLevel !== 'paranoid') {
    confirmMessage = deps.texts.confirmParanoid;
  } else if (newLevel !== 'paranoid' && currentLevel === 'paranoid') {
    confirmMessage = deps.texts.confirmBalanced;
  }

  if (confirmMessage) {
    const confirmed = await deps.confirm({
      title: deps.texts.confirmTitle,
      message: confirmMessage,
      confirmText: deps.texts.confirmLabel,
      cancelText: deps.texts.confirmCancel,
      variant: newLevel === 'paranoid' ? 'danger' : 'default',
    });

    if (!confirmed) return;
  }

  const oldSecurityLevel = currentLevel;
  const oldAutoLockRunning = deps.getAutoLockTimeout() > 0;

  deps.setIsSaving(true);
  try {
    await deps.updateSecurityLevel(newLevel);

    const success = await deps.updateSecurityPreferences({ security_level: newLevel });
    if (!success) {
      throw new Error('Backend save failed');
    }

    deps.setSecurityLevel(newLevel);

    if (newLevel === 'paranoid') {
      deps.stopAutoLock();
    } else if (oldAutoLockRunning) {
      deps.stopAutoLock();
      deps.initAutoLock(deps.getAutoLockTimeout());
    }
  } catch (err) {
    console.error('Failed to change security level:', err);
    deps.setSecurityLevel(oldSecurityLevel);
    try {
      await deps.updateSecurityLevel(oldSecurityLevel);
    } catch (rollbackErr) {
      console.error('Rollback failed:', rollbackErr);
    }
  } finally {
    deps.setIsSaving(false);
  }
}

export interface AutoLockTimeoutDeps {
  getIsSaving: () => boolean;
  setIsSaving: (value: boolean) => void;
  getAutoLockTimeout: () => number;
  setAutoLockTimeout: (timeout: number) => void;
  getSecurityLevel: () => SecurityLevel;
  updateSecurityPreferences: (prefs: { auto_lock_timeout: number }) => Promise<boolean>;
  stopAutoLock: () => void;
  initAutoLock: (timeout: number) => void;
}

export async function handleAutoLockTimeoutChange(deps: AutoLockTimeoutDeps) {
  if (deps.getIsSaving()) return;

  const oldTimeout = deps.getAutoLockTimeout();
  deps.setIsSaving(true);

  try {
    const success = await deps.updateSecurityPreferences({
      auto_lock_timeout: deps.getAutoLockTimeout(),
    });

    if (!success) {
      throw new Error('Backend save failed');
    }

    deps.stopAutoLock();
    if (deps.getAutoLockTimeout() > 0 && deps.getSecurityLevel() !== 'paranoid') {
      deps.initAutoLock(deps.getAutoLockTimeout());
    }
  } catch (err) {
    console.error('Failed to change auto-lock timeout:', err);
    deps.setAutoLockTimeout(oldTimeout);
  } finally {
    deps.setIsSaving(false);
  }
}
