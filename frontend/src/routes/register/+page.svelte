<script lang="ts">
  import { Lock } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { type AppConfig,getConfig } from '$lib/api';
  import CaptchaIframe from '$lib/components/CaptchaIframe.svelte';
  import Logo from '$lib/components/Logo.svelte';
  import { getServerUrl,isDesktop } from '$lib/config';
  import * as auth from '$lib/stores/auth.svelte';

  let username = $state('');
  let email = $state('');
  let password = $state('');
  let confirmPassword = $state('');
  let errorMessage = $state<string | null>(null);
  let isLoading = $state(false);
  let usernameInput = $state<HTMLInputElement | null>(null);

  // CAPTCHA state
  let captchaEnabled = $state(false);
  let captchaSiteKey = $state('');
  let captchaToken = $state<string | null>(null);
  let captchaWidgetId: string | null = null;
  let captchaIframeUrl = $state('');

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
        errorMessage = $_('page.login.captcha_error'); // Re-use from login page
      },
      theme: 'auto',
    });
  }

  // If already authenticated, redirect to home
  onMount(() => {
    if (auth.isAuthenticated()) {
      goto('/');
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
      const container = document.getElementById('register-captcha-container');
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

  async function handleRegister() {
    errorMessage = null;

    // Validation
    if (!username.trim() || !email.trim() || !password) {
      errorMessage = $_('page.register.all_fields_required');
      return;
    }

    if (password.length < 8) {
      errorMessage = $_('page.register.password_min_length');
      return;
    }

    if (password !== confirmPassword) {
      errorMessage = $_('page.register.passwords_do_not_match');
      return;
    }

    // Email validation (basic)
    if (!email.includes('@') || !email.includes('.')) {
      errorMessage = $_('page.register.invalid_email');
      return;
    }

    // CAPTCHA validation (if enabled)
    if (captchaEnabled && !captchaToken) {
      errorMessage = $_('page.login.captcha_required');
      return;
    }

    isLoading = true;
    try {
      await auth.register(username.trim(), email.trim(), password, captchaToken ?? undefined);
      goto('/login');
    } catch (err) {
      console.error('Registration failed:', err);
      errorMessage = err instanceof Error ? err.message : $_('page.register.registration_failed');
      // Reset CAPTCHA on error
      resetCaptcha();
    } finally {
      isLoading = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter') {
      handleRegister();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="register-container">
  <div class="register-card">
    <h1><Logo size="xl" /></h1>
    <p class="subtitle">{$_('page.register.title')}</p>

    <form
      onsubmit={(e) => {
        e.preventDefault();
        handleRegister();
      }}
    >
      <div class="form-group">
        <label for="username">{$_('common.username')}</label>
        <input
          id="username"
          name="username"
          type="text"
          bind:value={username}
          bind:this={usernameInput}
          placeholder="username"
          disabled={isLoading}
          aria-describedby={errorMessage ? 'register-error' : undefined}
        />
      </div>

      <div class="form-group">
        <label for="email">{$_('common.email')}</label>
        <input
          id="email"
          name="email"
          type="email"
          bind:value={email}
          placeholder="user@example.com"
          disabled={isLoading}
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
        <p class="hint">{$_('page.register.password_hint')}</p>

        {#if password.length > 0}
          {@const strength =
            password.length >= 16 &&
            /[!@#$%^&*(),.?":{}|<>]/.test(password) &&
            /[A-Z]/.test(password) &&
            /[a-z]/.test(password) &&
            /\d/.test(password)
              ? 4
              : password.length >= 12 &&
                  /[A-Z]/.test(password) &&
                  /[a-z]/.test(password) &&
                  /\d/.test(password)
                ? 3
                : password.length >= 10 &&
                    ((/[A-Z]/.test(password) && /[a-z]/.test(password)) || /\d/.test(password))
                  ? 2
                  : 1}
          {@const strengthLabel =
            strength === 4
              ? $_('page.register.password_strength.very_strong')
              : strength === 3
                ? $_('page.register.password_strength.strong')
                : strength === 2
                  ? $_('page.register.password_strength.medium')
                  : $_('page.register.password_strength.weak')}
          {@const strengthColor =
            strength === 4
              ? 'var(--color-success)'
              : strength === 3
                ? 'var(--color-success)'
                : strength === 2
                  ? 'var(--color-warning)'
                  : 'var(--color-destructive)'}
          <div class="password-strength" aria-live="polite">
            <div class="strength-bar">
              {#each [1, 2, 3, 4] as segment (segment)}
                <div
                  class="strength-segment"
                  style="background-color: {segment <= strength
                    ? strengthColor
                    : 'var(--color-border)'};"
                ></div>
              {/each}
            </div>
            <span class="strength-label" style="color: {strengthColor};">{strengthLabel}</span>
          </div>
        {/if}
      </div>

      <div class="form-group">
        <label for="confirm-password">{$_('page.register.confirm_password')}</label>
        <input
          id="confirm-password"
          name="confirmPassword"
          type="password"
          bind:value={confirmPassword}
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
            <div id="register-captcha-container" use:captchaMount></div>
          {/if}
        </div>
      {/if}

      {#if captchaEnabled && !captchaToken}
        <p class="captcha-hint">{$_('page.register.captcha_hint')}</p>
      {/if}

      {#if errorMessage}
        <div id="register-error" class="error-message" role="alert">{errorMessage}</div>
      {/if}

      <button
        type="submit"
        disabled={isLoading || (captchaEnabled && !captchaToken)}
        class="register-button"
      >
        {isLoading ? $_('common.registering') : $_('common.register')}
      </button>
    </form>

    <div class="footer-links">
      <span>{$_('page.register.already_have_account')}</span>
      <a href="/login">{$_('common.login')}</a>
      <span class="separator">·</span>
      <a href="/about">{$_('page.login.about_app')}</a>
    </div>

    <div class="trust-signal">
      <Lock size={14} />
      <span>{$_('page.login.badge_e2e')}</span>
    </div>
  </div>
</div>

<style>
  .register-container {
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh; /* Fallback for Safari <15.4 */
    min-height: 100dvh; /* Modern browsers */
    background: var(--color-background);
  }

  .register-card {
    width: 100%;
    max-width: 400px;
    padding: 2rem;
    background: var(--color-card);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
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
    border-radius: 4px;
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

  .hint {
    margin: 0.25rem 0 0 0;
    font-size: 0.75rem;
    color: var(--color-muted-foreground);
  }

  .register-button {
    width: 100%;
    padding: 0.75rem;
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-primary-foreground);
    background: var(--color-primary);
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s;
  }

  .register-button:hover:not(:disabled) {
    background: color-mix(in oklch, var(--color-primary), black 15%);
  }

  .register-button:disabled {
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
    border-radius: 4px;
  }

  .captcha-container {
    display: flex;
    justify-content: center;
  }

  .captcha-hint {
    font-size: 0.875rem;
    color: var(--color-muted-foreground);
    margin-bottom: 1rem;
    text-align: center;
  }

  .password-strength {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-top: 0.5rem;
  }

  .strength-bar {
    display: flex;
    gap: 3px;
    flex: 1;
  }

  .strength-segment {
    height: 4px;
    flex: 1;
    border-radius: 2px;
    transition: background-color 0.2s;
  }

  .strength-label {
    font-size: 0.75rem;
    font-weight: 500;
    white-space: nowrap;
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
</style>
