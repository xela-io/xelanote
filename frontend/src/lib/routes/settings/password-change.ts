export interface PasswordFormState {
  currentPassword: string;
  newPassword: string;
  confirmPassword: string;
  error: string;
  isChanging: boolean;
  reWrappingProgress: string;
}

export interface PasswordChangeDeps<TNote, TVersion> {
  form: PasswordFormState;
  setForm: (next: Partial<PasswordFormState>) => void;
  validationMessages: {
    currentPasswordRequired: string;
    newPasswordRequired: string;
    newPasswordMinLength: string;
    passwordsDoNotMatch: string;
  };
  getEncryptionEnabled: () => boolean;
  getAllEncryptedNotes: () => Promise<TNote[]>;
  getAllEncryptedVersions: () => Promise<TVersion[]>;
  getCurrentUser: () => { encryption_salt?: string } | null;
  reWrapAllDEKs: (
    notes: TNote[],
    versions: TVersion[],
    currentPassword: string,
    newPassword: string,
    saltBytes: Uint8Array
  ) => Promise<{ notes: Map<string, string>; versions: Map<string, string> }>;
  changePassword: (
    currentPassword: string,
    newPassword: string,
    reWrappedNoteDEKs?: Record<string, string>,
    reWrappedVersionDEKs?: Record<string, string>
  ) => Promise<{ recovery_key_invalidated?: string }>;
  setupKEK: (newPassword: string, saltBytes: Uint8Array) => Promise<void>;
  alert: (options: { title: string; message: string }) => void;
}

export async function handlePasswordChange<TNote, TVersion>(
  e: Event,
  deps: PasswordChangeDeps<TNote, TVersion>
) {
  e.preventDefault();
  deps.setForm({ error: '' });

  if (!deps.form.currentPassword) {
    deps.setForm({ error: deps.validationMessages.currentPasswordRequired });
    return;
  }

  if (!deps.form.newPassword) {
    deps.setForm({ error: deps.validationMessages.newPasswordRequired });
    return;
  }

  if (deps.form.newPassword.length < 8) {
    deps.setForm({ error: deps.validationMessages.newPasswordMinLength });
    return;
  }

  if (deps.form.newPassword !== deps.form.confirmPassword) {
    deps.setForm({ error: deps.validationMessages.passwordsDoNotMatch });
    return;
  }

  deps.setForm({ isChanging: true, reWrappingProgress: '' });

  try {
    const isEncryptionEnabled = deps.getEncryptionEnabled();
    let reWrappedNoteDEKs: Record<string, string> | undefined = undefined;
    let reWrappedVersionDEKs: Record<string, string> | undefined = undefined;

    if (isEncryptionEnabled) {
      try {
        deps.setForm({ reWrappingProgress: 'Fetching encrypted notes...' });
        const encryptedNotes = await deps.getAllEncryptedNotes();
        const encryptedVersions = await deps.getAllEncryptedVersions();

        if (encryptedNotes.length > 0 || encryptedVersions.length > 0) {
          deps.setForm({
            reWrappingProgress: `Re-wrapping ${encryptedNotes.length} notes and ${encryptedVersions.length} versions...`,
          });

          const currentUser = deps.getCurrentUser();
          if (!currentUser?.encryption_salt) {
            throw new Error('Encryption salt not available');
          }

          const saltBytes = Uint8Array.from(atob(currentUser.encryption_salt), (c) =>
            c.charCodeAt(0)
          );

          const result = await deps.reWrapAllDEKs(
            encryptedNotes,
            encryptedVersions,
            deps.form.currentPassword,
            deps.form.newPassword,
            saltBytes
          );

          reWrappedNoteDEKs = Object.fromEntries(result.notes);
          reWrappedVersionDEKs = Object.fromEntries(result.versions);

          deps.setForm({ reWrappingProgress: 'Validating re-wrapped keys...' });
        }
      } catch (err) {
        console.error('DEK re-wrapping failed:', err);
        deps.setForm({
          error: `Failed to re-wrap encryption keys: ${err instanceof Error ? err.message : 'Unknown error'}`,
          isChanging: false,
          reWrappingProgress: '',
        });
        return;
      }
    }

    deps.setForm({ reWrappingProgress: 'Updating password...' });
    const response = await deps.changePassword(
      deps.form.currentPassword,
      deps.form.newPassword,
      reWrappedNoteDEKs,
      reWrappedVersionDEKs
    );

    deps.setForm({
      currentPassword: '',
      newPassword: '',
      confirmPassword: '',
      reWrappingProgress: '',
    });

    if (response.recovery_key_invalidated === 'true') {
      deps.alert({
        title: 'Password Changed',
        message:
          'Your password has been changed successfully.\n\nIMPORTANT: Your recovery key has been invalidated. Please generate a new recovery key if you want to be able to recover your account.',
      });
    } else {
      deps.alert({
        title: 'Password Changed',
        message: 'Your password has been changed successfully.',
      });
    }

    const encryptionSalt = deps.getCurrentUser()?.encryption_salt;
    if (isEncryptionEnabled && encryptionSalt) {
      const saltBytes = Uint8Array.from(atob(encryptionSalt), (c) => c.charCodeAt(0));
      await deps.setupKEK(deps.form.newPassword, saltBytes);
    }
  } catch (err) {
    console.error('Password change failed:', err);
    let errorMsg = 'Failed to change password';

    if (err instanceof Error) {
      if (err.message.includes('DEK re-wrapping required')) {
        errorMsg = 'You have encrypted notes. Please try again.';
      } else if (err.message.includes('incorrect password')) {
        errorMsg = 'Incorrect current password';
      } else if (err.message.includes('Unauthorized')) {
        errorMsg = 'Incorrect current password';
      } else {
        errorMsg = err.message;
      }
    }

    deps.setForm({ error: errorMsg });
  } finally {
    deps.setForm({ isChanging: false, reWrappingProgress: '' });
  }
}
