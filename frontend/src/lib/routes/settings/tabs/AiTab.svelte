<script lang="ts">
  import { AlertTriangle, Eye, EyeOff, Key, Loader2, Sparkles, Trash2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { ApiKeyFormState } from '$lib/routes/settings/ai-keys';

  export let claudeApiKeyStatus: import('$lib/api').ClaudeAPIKeyStatus | null;
  export let isLoadingClaudeKeyStatus: boolean;
  export let claudeKeyForm: ApiKeyFormState;
  export let handleSaveClaudeApiKey: (e: Event) => void;
  export let handleDeleteClaudeApiKey: () => void;

  export let geminiApiKeyStatus: import('$lib/api').GeminiAPIKeyStatus | null;
  export let isLoadingGeminiKeyStatus: boolean;
  export let geminiKeyForm: ApiKeyFormState;
  export let handleSaveGeminiApiKey: (e: Event) => void;
  export let handleDeleteGeminiApiKey: () => void;
</script>

<div class="space-y-8">
  <!-- Claude API Key (BYOK) -->
  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">{$_('page.settings.ai.claude_title')}</h3>
    <p class="text-sm text-muted-foreground mb-4">{$_('page.settings.ai.claude_description')}</p>

    {#if isLoadingClaudeKeyStatus}
      <div class="p-4 rounded-lg border border-border bg-card flex items-center gap-3">
        <Loader2 size={20} class="animate-spin text-muted-foreground" />
        <span class="text-muted-foreground">{$_('common.loading')}</span>
      </div>
    {:else if claudeApiKeyStatus?.has_key}
      <!-- API Key is configured -->
      <div class="p-4 rounded-lg border border-success/30 bg-success/10">
        <div class="flex items-start gap-3">
          <Key size={20} class="text-success mt-0.5" />
          <div class="flex-1">
            <div class="font-medium text-success mb-1">{$_('page.settings.ai.api_key_configured')}</div>
            <div class="text-sm text-success font-mono">{claudeApiKeyStatus.masked_key}</div>
            {#if claudeApiKeyStatus.updated_at}
              <div class="text-xs text-success mt-2">
                {$_('page.settings.ai.api_key_updated_at', {
                  values: { date: new Date(claudeApiKeyStatus.updated_at).toLocaleDateString() },
                })}
              </div>
            {/if}
          </div>
          <button
            onclick={handleDeleteClaudeApiKey}
            disabled={claudeKeyForm.isDeleting}
            class="p-2 rounded-lg text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors disabled:opacity-50"
            title={$_('page.settings.ai.delete_api_key')}
          >
            {#if claudeKeyForm.isDeleting}
              <Loader2 size={18} class="animate-spin" />
            {:else}
              <Trash2 size={18} />
            {/if}
          </button>
        </div>
      </div>

      <!-- Option to update the key -->
      <details class="mt-4">
        <summary class="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
          {$_('page.settings.ai.update_api_key')}
        </summary>
        <form onsubmit={handleSaveClaudeApiKey} class="mt-4 space-y-4">
          <div>
            <label for="claude-api-key-update" class="block text-sm font-medium text-foreground mb-1">
              {$_('page.settings.ai.new_api_key_label')}
            </label>
            <div class="relative">
              <input
                id="claude-api-key-update"
                type={claudeKeyForm.showKey ? 'text' : 'password'}
                bind:value={claudeKeyForm.apiKey}
                disabled={claudeKeyForm.isSaving}
                class="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-foreground font-mono
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                placeholder="sk-ant-api03-..."
              />
              <button
                type="button"
                onclick={() => (claudeKeyForm.showKey = !claudeKeyForm.showKey)}
                class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
              >
                {#if claudeKeyForm.showKey}
                  <EyeOff size={18} />
                {:else}
                  <Eye size={18} />
                {/if}
              </button>
            </div>
          </div>

          {#if claudeKeyForm.error}
            <div class="text-sm text-red-500">{claudeKeyForm.error}</div>
          {/if}

          <button
            type="submit"
            disabled={claudeKeyForm.isSaving}
            class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
								font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {#if claudeKeyForm.isSaving}
              <Loader2 size={16} class="animate-spin" />
              {$_('common.saving')}
            {:else}
              {$_('page.settings.ai.update_api_key_button')}
            {/if}
          </button>
        </form>
      </details>
    {:else}
      <!-- No API Key configured -->
      <form onsubmit={handleSaveClaudeApiKey} class="space-y-4">
        <div>
          <label for="claude-api-key" class="block text-sm font-medium text-foreground mb-1">
            {$_('page.settings.ai.api_key_label')}
          </label>
          <div class="relative">
            <input
              id="claude-api-key"
              type={claudeKeyForm.showKey ? 'text' : 'password'}
              bind:value={claudeKeyForm.apiKey}
              disabled={claudeKeyForm.isSaving}
              class="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-foreground font-mono
								focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
              placeholder="sk-ant-api03-..."
            />
            <button
              type="button"
              onclick={() => (claudeKeyForm.showKey = !claudeKeyForm.showKey)}
              class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
            >
              {#if claudeKeyForm.showKey}
                <EyeOff size={18} />
              {:else}
                <Eye size={18} />
              {/if}
            </button>
          </div>
          <p class="text-xs text-muted-foreground mt-1">{$_('page.settings.ai.claude_key_hint')}</p>
        </div>

        {#if claudeKeyForm.error}
          <div class="text-sm text-red-500">{claudeKeyForm.error}</div>
        {/if}

        <button
          type="submit"
          disabled={claudeKeyForm.isSaving}
          class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
						font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {#if claudeKeyForm.isSaving}
            <Loader2 size={16} class="animate-spin" />
            {$_('common.saving')}
          {:else}
            <Key size={16} />
            {$_('page.settings.ai.save_api_key_button')}
          {/if}
        </button>
      </form>
    {/if}
  </div>

  <!-- Gemini API Key (BYOK) -->
  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">{$_('page.settings.ai.gemini_title')}</h3>
    <p class="text-sm text-muted-foreground mb-4">{$_('page.settings.ai.gemini_description')}</p>

    {#if isLoadingGeminiKeyStatus}
      <div class="p-4 rounded-lg border border-border bg-card flex items-center gap-3">
        <Loader2 size={20} class="animate-spin text-muted-foreground" />
        <span class="text-muted-foreground">{$_('common.loading')}</span>
      </div>
    {:else if geminiApiKeyStatus?.has_key}
      <!-- API Key is configured -->
      <div class="p-4 rounded-lg border border-success/30 bg-success/10">
        <div class="flex items-start gap-3">
          <Key size={20} class="text-success mt-0.5" />
          <div class="flex-1">
            <div class="font-medium text-success mb-1">{$_('page.settings.ai.api_key_configured')}</div>
            <div class="text-sm text-success font-mono">{geminiApiKeyStatus.masked_key}</div>
            {#if geminiApiKeyStatus.updated_at}
              <div class="text-xs text-success mt-2">
                {$_('page.settings.ai.api_key_updated_at', {
                  values: { date: new Date(geminiApiKeyStatus.updated_at).toLocaleDateString() },
                })}
              </div>
            {/if}
          </div>
          <button
            onclick={handleDeleteGeminiApiKey}
            disabled={geminiKeyForm.isDeleting}
            class="p-2 rounded-lg text-red-600 dark:text-red-400 hover:bg-red-100 dark:hover:bg-red-900/30 transition-colors disabled:opacity-50"
            title={$_('page.settings.ai.delete_api_key')}
          >
            {#if geminiKeyForm.isDeleting}
              <Loader2 size={18} class="animate-spin" />
            {:else}
              <Trash2 size={18} />
            {/if}
          </button>
        </div>
      </div>

      <!-- Option to update the key -->
      <details class="mt-4">
        <summary class="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
          {$_('page.settings.ai.update_api_key')}
        </summary>
        <form onsubmit={handleSaveGeminiApiKey} class="mt-4 space-y-4">
          <div>
            <label for="gemini-api-key-update" class="block text-sm font-medium text-foreground mb-1">
              {$_('page.settings.ai.new_api_key_label')}
            </label>
            <div class="relative">
              <input
                id="gemini-api-key-update"
                type={geminiKeyForm.showKey ? 'text' : 'password'}
                bind:value={geminiKeyForm.apiKey}
                disabled={geminiKeyForm.isSaving}
                class="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-foreground font-mono
									focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
                placeholder="AIzaSy..."
              />
              <button
                type="button"
                onclick={() => (geminiKeyForm.showKey = !geminiKeyForm.showKey)}
                class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
              >
                {#if geminiKeyForm.showKey}
                  <EyeOff size={18} />
                {:else}
                  <Eye size={18} />
                {/if}
              </button>
            </div>
          </div>

          {#if geminiKeyForm.error}
            <div class="text-sm text-red-500">{geminiKeyForm.error}</div>
          {/if}

          <button
            type="submit"
            disabled={geminiKeyForm.isSaving}
            class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
								font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {#if geminiKeyForm.isSaving}
              <Loader2 size={16} class="animate-spin" />
              {$_('common.saving')}
            {:else}
              {$_('page.settings.ai.update_api_key_button')}
            {/if}
          </button>
        </form>
      </details>
    {:else}
      <!-- No API Key configured -->
      <form onsubmit={handleSaveGeminiApiKey} class="space-y-4">
        <div>
          <label for="gemini-api-key" class="block text-sm font-medium text-foreground mb-1">
            {$_('page.settings.ai.api_key_label')}
          </label>
          <div class="relative">
            <input
              id="gemini-api-key"
              type={geminiKeyForm.showKey ? 'text' : 'password'}
              bind:value={geminiKeyForm.apiKey}
              disabled={geminiKeyForm.isSaving}
              class="w-full px-3 py-2 pr-10 rounded-lg border border-border bg-background text-foreground font-mono
								focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
              placeholder="AIzaSy..."
            />
            <button
              type="button"
              onclick={() => (geminiKeyForm.showKey = !geminiKeyForm.showKey)}
              class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground"
            >
              {#if geminiKeyForm.showKey}
                <EyeOff size={18} />
              {:else}
                <Eye size={18} />
              {/if}
            </button>
          </div>
          <p class="text-xs text-muted-foreground mt-1">{$_('page.settings.ai.gemini_key_hint')}</p>
        </div>

        {#if geminiKeyForm.error}
          <div class="text-sm text-red-500">{geminiKeyForm.error}</div>
        {/if}

        <button
          type="submit"
          disabled={geminiKeyForm.isSaving}
          class="flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-primary text-primary-foreground
						font-medium hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {#if geminiKeyForm.isSaving}
            <Loader2 size={16} class="animate-spin" />
            {$_('common.saving')}
          {:else}
            <Key size={16} />
            {$_('page.settings.ai.save_api_key_button')}
          {/if}
        </button>
      </form>
    {/if}
  </div>

  <!-- AI Features Info -->
  <div class="p-4 rounded-lg bg-primary/10 border border-primary/30">
    <div class="flex items-start gap-3">
      <Sparkles size={20} class="text-primary mt-0.5" />
      <div class="flex-1 text-sm text-foreground">
        <div class="font-medium mb-2">{$_('page.settings.ai.features_title')}</div>
        <ul class="list-disc list-inside space-y-1">
          <li>{$_('page.settings.ai.feature_summaries')}</li>
          <li>{$_('page.settings.ai.feature_tagging')}</li>
          <li>{$_('page.settings.ai.feature_linking')}</li>
        </ul>
      </div>
    </div>
  </div>

  <!-- Privacy Notice -->
  <div
    class="p-4 rounded-lg bg-orange-100/80 dark:bg-orange-900/20 border border-orange-400 dark:border-orange-700"
  >
    <div class="flex items-start gap-3">
      <AlertTriangle size={20} class="text-orange-700 dark:text-orange-400 mt-0.5" />
      <div class="flex-1">
        <div class="font-medium text-orange-950 dark:text-orange-200 mb-1">
          {$_('page.settings.ai.privacy_title')}
        </div>
        <div class="text-sm text-orange-900 dark:text-orange-300">
          {$_('page.settings.ai.privacy_notice')}
        </div>
      </div>
    </div>
  </div>
</div>
