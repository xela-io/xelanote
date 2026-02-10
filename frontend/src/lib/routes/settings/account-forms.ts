export interface EmailFormState {
  newEmail: string;
  password: string;
  error: string;
}

export interface EmailFormDeps {
  form: EmailFormState;
  setForm: (next: Partial<EmailFormState>) => void;
  changeEmail: (newEmail: string, password: string) => Promise<{ success: boolean; error?: string }>;
  reload: () => void;
  validationMessages: {
    emailRequired: string;
    passwordRequired: string;
    changeEmailFailed: string;
  };
}

export async function handleEmailSubmit(e: Event, deps: EmailFormDeps) {
  e.preventDefault();
  deps.setForm({ error: '' });

  if (!deps.form.newEmail.trim()) {
    deps.setForm({ error: deps.validationMessages.emailRequired });
    return;
  }

  if (!deps.form.password) {
    deps.setForm({ error: deps.validationMessages.passwordRequired });
    return;
  }

  const result = await deps.changeEmail(deps.form.newEmail.trim(), deps.form.password);

  if (result.success) {
    deps.setForm({ newEmail: '', password: '' });
    deps.reload();
  } else {
    deps.setForm({ error: result.error || deps.validationMessages.changeEmailFailed });
  }
}
