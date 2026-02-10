export interface TwoFactorStatusDeps<TStatus> {
  getStatus: () => Promise<TStatus>;
  setStatus: (status: TStatus | null) => void;
  setLoading: (value: boolean) => void;
}

export async function loadTwoFactorStatus<TStatus>(deps: TwoFactorStatusDeps<TStatus>) {
  deps.setLoading(true);
  try {
    const status = await deps.getStatus();
    deps.setStatus(status);
  } catch (err) {
    console.error('Failed to load 2FA status:', err);
  } finally {
    deps.setLoading(false);
  }
}

export interface TwoFactorSuccessDeps {
  closeDialog: () => void;
  reloadStatus: () => void | Promise<void>;
}

export function handleTwoFactorSetupSuccess(deps: TwoFactorSuccessDeps) {
  deps.closeDialog();
  void deps.reloadStatus();
}

export function handleTwoFactorDisableSuccess(deps: TwoFactorSuccessDeps) {
  deps.closeDialog();
  void deps.reloadStatus();
}

export interface BackupCodesPromptDeps {
  openPrompt: () => void;
}

export function requestBackupCodesRegeneration(deps: BackupCodesPromptDeps) {
  deps.openPrompt();
}

export interface ConfirmBackupCodesDeps {
  password: string;
  setIsRegenerating: (value: boolean) => void;
  regenerate: (password: string) => Promise<{ backup_codes: string[] }>;
  setNewBackupCodes: (codes: string[] | null) => void;
  setShowBackupCodesDialog: (value: boolean) => void;
  setShowPrompt: (value: boolean) => void;
  setPassword: (value: string) => void;
  reloadStatus: () => void | Promise<void>;
}

export async function confirmBackupCodesRegeneration(deps: ConfirmBackupCodesDeps) {
  if (!deps.password) {
    return;
  }

  deps.setIsRegenerating(true);
  try {
    const result = await deps.regenerate(deps.password);
    deps.setNewBackupCodes(result.backup_codes);
    deps.setShowBackupCodesDialog(true);
    deps.setShowPrompt(false);
    deps.setPassword('');
    void deps.reloadStatus();
  } catch (err) {
    console.error('Failed to regenerate backup codes:', err);
  } finally {
    deps.setIsRegenerating(false);
  }
}
