<script lang="ts">
  import { ArrowLeft, KeyRound, Lock, Shield } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { type AppConfig, getConfig } from '$lib/api';
  import CaptchaIframe from '$lib/components/CaptchaIframe.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import { getServerUrl, isDesktop } from '$lib/config';
  import { authenticateWithSecurityKey, isFIDO2Supported } from '$lib/crypto/fido2';
  import * as auth from '$lib/stores/auth.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as websocket from '$lib/stores/websocket.svelte';

  let usernameOrEmail = $state('');
  let password = $state('');
  let errorMessage = $state<string | null>(null);
  let infoMessage = $state<string | null>(null);
  let isLoading = $state(false);
  let usernameInput = $state<HTMLInputElement | null>(null);

  // CAPTCHA state
  let captchaEnabled = $state(false);
  let captchaSiteKey = $state('');
  let captchaToken = $state<string | null>(null);
  let captchaWidgetId: string | null = null;
  let captchaIframeUrl = $state('');

  // 2FA state
  let requiresTwoFactor = $state(false);
  let totpCode = $state('');
  let backupCode = $state('');
  let activeMethod = $state<'fido2' | 'totp' | 'backup_code'>('totp');
  let availableMethods = $state<string[]>([]);
  let pendingLoginToken = $state<string | null>(null);
  let isFido2Authenticating = $state(false);
  let storedCredentials = $state<{
    username: string;
    password: string;
    captchaToken?: string;
  } | null>(null);
  let totpInput = $state<HTMLInputElement | null>(null);

  // Svelte action: called when the captcha container element is mounted
  function captchaMount(node: HTMLElement) {
    // Try to render immediately if turnstile is already loaded
    tryRenderCaptcha(node);

    return {
      destroy() {
        // Cleanup when element is removed
        if (captchaWidgetId && window.turnstile) {
          try {
            window.turnstile.remove(captchaWidgetId);
            captchaWidgetId = null;
          } catch {
            // Ignore errors during cleanup
          }
        }
      },
    };
  }

  function tryRenderCaptcha(container: HTMLElement) {
    if (!window.turnstile || captchaWidgetId || !captchaSiteKey) {
      return;
    }

    captchaWidgetId = window.turnstile.render(container, {
      sitekey: captchaSiteKey,
      callback: (token: string) => {
        captchaToken = token;
      },
      'expired-callback': () => {
        captchaToken = null;
      },
      'error-callback': () => {
        captchaToken = null;
        errorMessage = $_('page.login.captcha_error');
      },
      theme: 'auto',
    });
  }

  // If already authenticated, redirect to home
  onMount(() => {
    if (auth.isAuthenticated()) {
      goto('/');
    }

    // Check if user was logged out due to encryption being locked
    if (typeof window !== 'undefined') {
      const params = new URLSearchParams(window.location.search);
      if (params.get('reason') === 'encryption_locked') {
        infoMessage = $_('page.login.encryption_locked_info');
      }
    }

    usernameInput?.focus();

    // Load config and setup CAPTCHA if needed
    loadConfig();
  });

  async function loadConfig() {
    try {
      const config: AppConfig = await getConfig();
      captchaEnabled = config.captcha_enabled;
      if (config.captcha_site_key) {
        captchaSiteKey = config.captcha_site_key;
      }
      if (config.captcha_iframe_url) {
        captchaIframeUrl = config.captcha_iframe_url;
      }

      if (captchaEnabled && captchaSiteKey && !isDesktop()) {
        loadTurnstileScript();
      }
    } catch (err) {
      console.error('Failed to load config:', err);
    }
  }

  function loadTurnstileScript() {
    // Check if script already loaded
    if (window.turnstile) {
      // Script already loaded, widget will render via action
      return;
    }

    // Check if script tag already exists
    if (document.querySelector('script[src*="turnstile"]')) {
      return;
    }

    // Load the script with render=explicit so we control when widgets render
    const script = document.createElement('script');
    script.src = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit';
    script.async = true;

    script.onload = () => {
      // Script loaded - find container and render
      const container = document.getElementById('captcha-container');
      if (container && window.turnstile && !captchaWidgetId) {
        tryRenderCaptcha(container);
      }
    };

    document.head.appendChild(script);
  }

  function resetCaptcha() {
    captchaToken = null;
    if (captchaWidgetId && window.turnstile) {
      try {
        window.turnstile.reset(captchaWidgetId);
      } catch {
        // Ignore errors
      }
    }
  }

  async function handleLogin() {
    errorMessage = null;

    if (!usernameOrEmail.trim() || !password) {
      errorMessage = $_('page.login.username_password_required');
      return;
    }

    // CAPTCHA validation (if enabled)
    if (captchaEnabled && !captchaToken) {
      errorMessage = $_('page.login.captcha_required');
      return;
    }

    isLoading = true;
    try {
      const result = await auth.login(usernameOrEmail.trim(), password, captchaToken ?? undefined);

      if (result.requiresTwoFactor) {
        // Store credentials for 2FA verification
        storedCredentials = {
          username: usernameOrEmail.trim(),
          password: password,
          captchaToken: captchaToken ?? undefined,
        };
        pendingLoginToken = result.pendingLoginToken ?? null;
        availableMethods = result.twoFactorMethods ?? ['totp', 'backup_code'];
        requiresTwoFactor = true;

        // Default to FIDO2 if available and supported
        if (availableMethods.includes('fido2') && isFIDO2Supported()) {
          activeMethod = 'fido2';
        } else if (availableMethods.includes('totp')) {
          activeMethod = 'totp';
        } else {
          activeMethod = 'backup_code';
        }

        // Focus TOTP input after render if that method is active
        if (activeMethod === 'totp') {
          setTimeout(() => totpInput?.focus(), 100);
        }
      } else {
        // Login successful - initialize and navigate
        await initializeAndNavigate();
      }
    } catch (err) {
      console.error('Login failed:', err);
      errorMessage = err instanceof Error ? err.message : $_('page.login.login_failed');
      // Reset CAPTCHA on error
      resetCaptcha();
    } finally {
      isLoading = false;
    }
  }

  async function handleTwoFactorVerify() {
    errorMessage = null;

    if (!storedCredentials) {
      errorMessage = $_('page.login.relogin_required');
      resetToLogin();
      return;
    }

    const isBackup = activeMethod === 'backup_code';
    const code = isBackup ? backupCode : totpCode;
    if (!code) {
      errorMessage = isBackup
        ? $_('page.login.enter_backup_code')
        : $_('page.login.enter_six_digit_code');
      return;
    }

    isLoading = true;
    try {
      const result = await auth.login(
        storedCredentials.username,
        storedCredentials.password,
        storedCredentials.captchaToken,
        isBackup ? undefined : totpCode,
        isBackup ? backupCode : undefined
      );

      if (result.requiresTwoFactor) {
        // Code was invalid
        errorMessage = $_('page.login.invalid_code');
        if (isBackup) {
          backupCode = '';
        } else {
          totpCode = '';
        }
      } else {
        // 2FA verification successful - initialize and navigate
        await initializeAndNavigate();
      }
    } catch (err) {
      console.error('2FA verification failed:', err);
      errorMessage = err instanceof Error ? err.message : $_('page.login.twofa_failed');
      if (isBackup) {
        backupCode = '';
      } else {
        totpCode = '';
      }
    } finally {
      isLoading = false;
    }
  }

  function resetToLogin() {
    requiresTwoFactor = false;
    storedCredentials = null;
    totpCode = '';
    backupCode = '';
    activeMethod = 'totp';
    availableMethods = [];
    pendingLoginToken = null;
    isFido2Authenticating = false;
    errorMessage = null;
    // Reset CAPTCHA since we're starting fresh
    resetCaptcha();
  }

  async function handleFido2Authenticate() {
    if (!pendingLoginToken) {
      errorMessage = $_('page.login.relogin_required');
      resetToLogin();
      return;
    }

    errorMessage = null;
    isFido2Authenticating = true;

    try {
      const result = await authenticateWithSecurityKey(pendingLoginToken);
      if (result.access_token && result.refresh_token && result.user) {
        await auth.setAuth(result.access_token, result.refresh_token, result.user);
        await initializeAndNavigate();
      } else {
        errorMessage = $_('page.login.login_failed');
      }
    } catch (err) {
      console.error('FIDO2 auth failed:', err);
      errorMessage = err instanceof Error ? err.message : $_('page.login.twofa_failed');
    } finally {
      isFido2Authenticating = false;
    }
  }

  function handleTotpInput(e: Event) {
    const input = e.target as HTMLInputElement;
    input.value = input.value.replace(/\D/g, '').slice(0, 6);
    totpCode = input.value;
  }

  function handleBackupCodeInput(e: Event) {
    const input = e.target as HTMLInputElement;
    input.value = input.value
      .toUpperCase()
      .replace(/[^A-Z0-9-]/g, '')
      .slice(0, 9);
    backupCode = input.value;
  }

  /**
   * Initialize user data and navigate to home after successful login.
   * This loads preferences, notes, and connects to WebSocket.
   *
   * Uses client-side initialization + goto('/') for all modes (including
   * standalone PWA). The layout's $derived `isPublic` reactively switches
   * from public to protected branch when auth.isAuthenticated() becomes true.
   * The Sidebar's $effect triggers tree.loadTree() on auth change.
   */
  async function initializeAndNavigate() {
    try {
      // Load user preferences (theme + editor mode + security level - all in one call)
      await settings.loadPreferences();

      // Load notes (fire-and-forget, UI updates reactively when data arrives)
      notes.loadNotes();

      // Connect to WebSocket for real-time updates
      websocket.connect();

      // Navigate to home — layout reactively switches to protected branch
      goto('/');
    } catch (err) {
      console.error('[LOGIN] Failed to initialize after login:', err);
      // Still navigate even if preferences fail (user can try again)
      goto('/');
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      if (requiresTwoFactor && activeMethod !== 'fido2') {
        handleTwoFactorVerify();
      } else if (!requiresTwoFactor) {
        handleLogin();
      }
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="login-container">
  <div class="login-card">
    <h1><Logo size="xl" /></h1>

    {#if requiresTwoFactor}
      <!-- 2FA Step -->
      <div class="twofa-header">
        <div class="twofa-icon">
          <Shield size={32} />
        </div>
        <p class="subtitle">{$_('page.login.twofa_title')}</p>
      </div>

      <!-- Method tabs -->
      {#if availableMethods.length > 1}
        <div class="method-tabs">
          {#if availableMethods.includes('fido2') && isFIDO2Supported()}
            <button
              type="button"
              class="method-tab"
              class:active={activeMethod === 'fido2'}
              onclick={() => {
                activeMethod = 'fido2';
                errorMessage = null;
              }}
            >
              <KeyRound size={16} />
              Security Key
            </button>
          {/if}
          {#if availableMethods.includes('totp')}
            <button
              type="button"
              class="method-tab"
              class:active={activeMethod === 'totp'}
              onclick={() => {
                activeMethod = 'totp';
                errorMessage = null;
                setTimeout(() => totpInput?.focus(), 100);
              }}
            >
              <Shield size={16} />
              Authenticator
            </button>
          {/if}
          <button
            type="button"
            class="method-tab"
            class:active={activeMethod === 'backup_code'}
            onclick={() => {
              activeMethod = 'backup_code';
              errorMessage = null;
            }}
          >
            Backup Code
          </button>
        </div>
      {/if}

      {#if activeMethod === 'fido2'}
        <!-- FIDO2 Security Key -->
        <div class="fido2-prompt">
          <div class="fido2-icon">
            <KeyRound size={48} />
          </div>
          <p class="fido2-instruction">
            Stecke deinen Security Key ein und berühre ihn, oder nutze den integrierten
            Authenticator deines Geräts.
          </p>

          {#if errorMessage}
            <div class="error-message" role="alert">{errorMessage}</div>
          {/if}

          <button
            type="button"
            onclick={handleFido2Authenticate}
            disabled={isFido2Authenticating}
            class="login-button"
          >
            {isFido2Authenticating ? 'Warte auf Security Key...' : 'Mit Security Key anmelden'}
          </button>
        </div>
      {:else}
        <form
          onsubmit={(e) => {
            e.preventDefault();
            handleTwoFactorVerify();
          }}
        >
          {#if activeMethod === 'backup_code'}
            <div class="form-group">
              <label for="backup-code">{$_('page.login.backup_code')}</label>
              <input
                id="backup-code"
                type="text"
                value={backupCode}
                oninput={handleBackupCodeInput}
                placeholder="XXXX-XXXX"
                disabled={isLoading}
                class="code-input"
              />
            </div>
          {:else}
            <div class="form-group">
              <label for="totp-code">{$_('page.login.totp_code')}</label>
              <input
                id="totp-code"
                name="totp_code"
                type="text"
                inputmode="numeric"
                maxlength="6"
                value={totpCode}
                oninput={handleTotpInput}
                bind:this={totpInput}
                placeholder="000000"
                disabled={isLoading}
                class="code-input"
              />
            </div>
          {/if}

          {#if errorMessage}
            <div class="error-message" role="alert">{errorMessage}</div>
          {/if}

          <button type="submit" disabled={isLoading} class="login-button">
            {isLoading ? $_('common.verifying') : $_('common.verify')}
          </button>
        </form>
      {/if}

      <button type="button" onclick={resetToLogin} class="back-button">
        <ArrowLeft size={16} />
        {$_('page.login.back_to_login')}
      </button>
    {:else}
      <!-- Login Step -->
      <p class="subtitle">{$_('page.login.title')}</p>

      <form
        onsubmit={(e) => {
          e.preventDefault();
          handleLogin();
        }}
      >
        <div class="form-group">
          <label for="username">{$_('common.username_or_email')}</label>
          <input
            id="username"
            name="username_or_email"
            type="text"
            bind:value={usernameOrEmail}
            bind:this={usernameInput}
            placeholder="username@example.com"
            disabled={isLoading}
            aria-describedby={errorMessage ? 'login-error' : undefined}
          />
        </div>

        <div class="form-group">
          <label for="password">{$_('common.password')}</label>
          <input
            id="password"
            name="password"
            type="password"
            bind:value={password}
            placeholder="••••••••"
            disabled={isLoading}
          />
        </div>

        {#if captchaEnabled}
          <div class="form-group captcha-container">
            {#if isDesktop() && captchaIframeUrl}
              <CaptchaIframe
                iframeUrl="{getServerUrl()}{captchaIframeUrl}"
                onToken={(token) => (captchaToken = token)}
                onExpired={() => (captchaToken = null)}
                onError={() => {
                  captchaToken = null;
                  errorMessage = $_('page.login.captcha_error');
                }}
              />
            {:else if !isDesktop()}
              <div id="captcha-container" use:captchaMount></div>
            {/if}
          </div>
        {/if}

        {#if captchaEnabled && !captchaToken}
          <p class="text-sm text-muted-foreground mb-4">{$_('page.login.captcha_hint')}</p>
        {/if}

        {#if infoMessage}
          <div class="info-message">
            {infoMessage}
          </div>
        {/if}

        {#if errorMessage}
          <div id="login-error" class="error-message" role="alert">{errorMessage}</div>
        {/if}

        <button
          type="submit"
          disabled={isLoading || (captchaEnabled && !captchaToken)}
          class="login-button"
        >
          {isLoading ? $_('common.logging_in') : $_('common.login')}
        </button>
      </form>

      <div class="footer-links">
        <span>{$_('page.login.no_account_yet')}</span>
        <a href="/register">{$_('common.register')}</a>
        <span class="separator">·</span>
        <a href="/about">{$_('page.login.about_app')}</a>
      </div>

      <div class="trust-signal">
        <Lock size={14} />
        <span>{$_('page.login.badge_e2e')}</span>
      </div>
    {/if}
  </div>
</div>

<style>
  .login-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh; /* Fallback for Safari <15.4 */
    min-height: 100dvh; /* Modern browsers */
    padding: 0 1rem;
    background: var(--color-background);
  }

  .login-card {
    width: 100%;
    max-width: 400px;
    padding: 1.5rem;
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-md, 0 4px 6px rgba(0, 0, 0, 0.1));
  }

  h1 {
    margin: 0;
    font-size: 2rem;
    font-weight: 700;
    text-align: center;
    color: var(--color-foreground);
  }

  .subtitle {
    margin: 0.5rem 0 2rem;
    text-align: center;
    color: var(--color-muted-foreground);
  }

  .form-group {
    margin-bottom: 1.5rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-foreground);
  }

  input {
    width: 100%;
    padding: 0.75rem;
    font-size: 1rem;
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    color: var(--color-foreground);
    box-sizing: border-box;
  }

  input:focus {
    outline: none;
    border-color: var(--color-primary);
    box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary), transparent 80%);
  }

  input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .login-button {
    width: 100%;
    padding: 0.75rem;
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-primary-foreground);
    background: var(--color-primary);
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: background var(--duration-base) var(--ease-default);
  }

  .login-button:hover:not(:disabled) {
    background: color-mix(in oklch, var(--color-primary), black 15%);
  }

  .login-button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .error-message {
    margin-bottom: 1rem;
    padding: 0.75rem;
    font-size: 0.875rem;
    color: var(--color-destructive);
    background: color-mix(in oklch, var(--color-destructive), transparent 85%);
    border: 1px solid var(--color-destructive);
    border-radius: var(--radius-sm);
  }

  .captcha-container {
    display: flex;
    justify-content: center;
  }

  .footer-links {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 0.5rem;
    margin-top: 1.5rem;
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
  }

  .footer-links a {
    color: var(--color-primary);
    text-decoration: none;
    font-weight: 600;
  }

  .footer-links a:hover {
    text-decoration: underline;
  }

  .footer-links .separator {
    color: var(--color-muted-foreground);
  }

  .trust-signal {
    display: flex;
    justify-content: center;
    align-items: center;
    gap: 0.375rem;
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--color-border);
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
  }

  .info-message {
    background-color: color-mix(in oklch, var(--color-primary), transparent 85%);
    color: var(--color-primary);
    padding: 12px;
    border-radius: var(--radius-md);
    margin-bottom: 16px;
    border-left: 4px solid var(--color-primary);
  }

  /* Method tabs */
  .method-tabs {
    display: flex;
    gap: 0.25rem;
    margin-bottom: 1.5rem;
    padding: 0.25rem;
    background: var(--color-background);
    border-radius: var(--radius-md);
    border: 1px solid var(--color-border);
  }

  .method-tab {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.375rem;
    padding: 0.5rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--color-muted-foreground);
    background: transparent;
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--duration-base) var(--ease-default);
  }

  .method-tab:hover {
    color: var(--color-foreground);
    background: color-mix(in oklch, var(--color-primary), transparent 90%);
  }

  .method-tab.active {
    color: var(--color-primary);
    background: var(--color-card);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
  }

  /* FIDO2 prompt */
  .fido2-prompt {
    text-align: center;
    margin-bottom: 1rem;
  }

  .fido2-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 80px;
    height: 80px;
    margin: 0 auto 1rem;
    background: color-mix(in oklch, var(--color-primary), transparent 85%);
    border-radius: 50%;
    color: var(--color-primary);
  }

  .fido2-instruction {
    margin-bottom: 1.5rem;
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
    line-height: 1.5;
  }

  /* 2FA Styles */
  .twofa-header {
    text-align: center;
    margin-bottom: 1.5rem;
  }

  .twofa-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 64px;
    height: 64px;
    margin: 0 auto 1rem;
    background: color-mix(in oklch, var(--color-primary), transparent 80%);
    border-radius: 50%;
    color: var(--color-primary);
  }

  .code-input {
    text-align: center;
    font-size: 1.5rem;
    font-family: monospace;
    letter-spacing: 0.25em;
  }

  .back-button {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    width: 100%;
    margin-top: 1rem;
    padding: 0.75rem;
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
    background: transparent;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--duration-base) var(--ease-default);
  }

  .back-button:hover {
    color: var(--color-foreground);
    background: var(--color-background);
  }

  @media (min-width: 640px) {
    .login-card {
      padding: 2rem;
    }
  }
</style>
