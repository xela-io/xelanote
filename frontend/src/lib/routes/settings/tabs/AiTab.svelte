<script lang="ts">
  import { AlertTriangle, Eye, EyeOff, Key, Loader2, Sparkles, Trash2 } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type {
    AIAvailableModelItem,
    AIAvailableModelsResponse,
    AIModelPreferences,
    AIProvider,
    ClaudeAPIKeyStatus,
    DietaryPreference,
    GeminiAPIKeyStatus,
    OpenAIAPIKeyStatus,
  } from '$lib/api';
  import type { ApiKeyFormState } from '$lib/routes/settings/ai-keys';

  export let claudeApiKeyStatus: ClaudeAPIKeyStatus | null;
  export let isLoadingClaudeKeyStatus: boolean;
  export let claudeKeyForm: ApiKeyFormState;
  export let handleSaveClaudeApiKey: (e: Event) => void;
  export let handleDeleteClaudeApiKey: () => void;

  export let geminiApiKeyStatus: GeminiAPIKeyStatus | null;
  export let isLoadingGeminiKeyStatus: boolean;
  export let geminiKeyForm: ApiKeyFormState;
  export let handleSaveGeminiApiKey: (e: Event) => void;
  export let handleDeleteGeminiApiKey: () => void;

  export let openAIApiKeyStatus: OpenAIAPIKeyStatus | null;
  export let isLoadingOpenAIKeyStatus: boolean;
  export let openAIKeyForm: ApiKeyFormState;
  export let handleSaveOpenAIApiKey: (e: Event) => void;
  export let handleDeleteOpenAIApiKey: () => void;

  export let activeAIProvider: AIProvider;
  export let isSavingAIProvider: boolean;
  export let handleAIProviderChange: (provider: AIProvider) => void;
  export let dietaryPreference: DietaryPreference;
  export let isSavingDietaryPreference: boolean;
  export let handleDietaryPreferenceChange: (pref: DietaryPreference) => void;
  export let aiModels: AIModelPreferences;
  export let availableAIModels: AIAvailableModelsResponse | null;
  export let isLoadingAvailableAIModels: boolean;
  export let isSavingAIModels: boolean;
  export let handleSaveAIModels: (e: Event) => void;

  function includesModel(options: AIAvailableModelItem[], value: string): boolean {
    return options.some((m) => m.id === value);
  }

  function formatPrice(input: number, output: number): string {
    return `$${input.toFixed(2)} / $${output.toFixed(2)}`;
  }
</script>

<div class="space-y-8">
  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">
      {$_('page.settings.ai.provider_title')}
    </h3>
    <p class="text-sm text-muted-foreground mb-4">{$_('page.settings.ai.provider_description')}</p>
    <select
      value={activeAIProvider}
      disabled={isSavingAIProvider}
      onchange={(e) => handleAIProviderChange((e.target as HTMLSelectElement).value as AIProvider)}
      class="ui-select"
    >
      <option value="auto">{$_('page.settings.ai.provider_auto')}</option>
      <option value="claude">{$_('page.settings.ai.provider_claude')}</option>
      <option value="gemini">{$_('page.settings.ai.provider_gemini')}</option>
      <option value="chatgpt">{$_('page.settings.ai.provider_chatgpt')}</option>
    </select>
    {#if isSavingAIProvider}
      <div class="text-xs text-muted-foreground mt-2">{$_('common.saving')}</div>
    {/if}
  </div>

  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">
      {$_('page.settings.ai.dietary_title')}
    </h3>
    <p class="text-sm text-muted-foreground mb-4">{$_('page.settings.ai.dietary_description')}</p>
    <select
      value={dietaryPreference}
      disabled={isSavingDietaryPreference}
      onchange={(e) =>
        handleDietaryPreferenceChange((e.target as HTMLSelectElement).value as DietaryPreference)}
      class="ui-select"
    >
      <option value="none">{$_('page.settings.ai.dietary_none')}</option>
      <option value="vegetarian">{$_('page.settings.ai.dietary_vegetarian')}</option>
      <option value="vegan">{$_('page.settings.ai.dietary_vegan')}</option>
      <option value="pescatarian">{$_('page.settings.ai.dietary_pescatarian')}</option>
      <option value="flexitarian">{$_('page.settings.ai.dietary_flexitarian')}</option>
    </select>
    {#if isSavingDietaryPreference}
      <div class="text-xs text-muted-foreground mt-2">{$_('common.saving')}</div>
    {/if}
  </div>

  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">{$_('page.settings.ai.models_title')}</h3>
    <p class="text-sm text-muted-foreground mb-4">{$_('page.settings.ai.models_description')}</p>

    <form onsubmit={handleSaveAIModels} class="ui-fieldset">
      <div class="ui-form-row">
        <label for="claude-model" class="ui-label mb-0">
          {$_('page.settings.ai.provider_claude')}
        </label>
        <select
          id="claude-model"
          bind:value={aiModels.claude_model}
          disabled={isSavingAIModels || isLoadingAvailableAIModels}
          class="ui-select font-mono"
        >
          <option value="">{$_('page.settings.ai.model_default')}</option>
          {#if aiModels.claude_model && availableAIModels && !includesModel(availableAIModels.claude_models, aiModels.claude_model)}
            <option value={aiModels.claude_model}>
              {$_('page.settings.ai.model_custom_current', {
                values: { model: aiModels.claude_model },
              })}
            </option>
          {/if}
          {#if availableAIModels}
            {#each availableAIModels.claude_models as model (model.id)}
              <option value={model.id}>
                {model.id} ({formatPrice(model.input_cost_per_1m, model.output_cost_per_1m)} USD)
              </option>
            {/each}
          {/if}
        </select>
      </div>

      <div class="ui-form-row">
        <label for="gemini-model" class="ui-label mb-0">
          {$_('page.settings.ai.provider_gemini')}
        </label>
        <select
          id="gemini-model"
          bind:value={aiModels.gemini_model}
          disabled={isSavingAIModels || isLoadingAvailableAIModels}
          class="ui-select font-mono"
        >
          <option value="">{$_('page.settings.ai.model_default')}</option>
          {#if aiModels.gemini_model && availableAIModels && !includesModel(availableAIModels.gemini_models, aiModels.gemini_model)}
            <option value={aiModels.gemini_model}>
              {$_('page.settings.ai.model_custom_current', {
                values: { model: aiModels.gemini_model },
              })}
            </option>
          {/if}
          {#if availableAIModels}
            {#each availableAIModels.gemini_models as model (model.id)}
              <option value={model.id}>
                {model.id} ({formatPrice(model.input_cost_per_1m, model.output_cost_per_1m)} USD)
              </option>
            {/each}
          {/if}
        </select>
      </div>

      <div class="ui-form-row">
        <label for="chatgpt-model" class="ui-label mb-0">
          {$_('page.settings.ai.provider_chatgpt')}
        </label>
        <select
          id="chatgpt-model"
          bind:value={aiModels.chatgpt_model}
          disabled={isSavingAIModels || isLoadingAvailableAIModels}
          class="ui-select font-mono"
        >
          <option value="">{$_('page.settings.ai.model_default')}</option>
          {#if aiModels.chatgpt_model && availableAIModels && !includesModel(availableAIModels.chatgpt_models, aiModels.chatgpt_model)}
            <option value={aiModels.chatgpt_model}>
              {$_('page.settings.ai.model_custom_current', {
                values: { model: aiModels.chatgpt_model },
              })}
            </option>
          {/if}
          {#if availableAIModels}
            {#each availableAIModels.chatgpt_models as model (model.id)}
              <option value={model.id}>
                {model.id} ({formatPrice(model.input_cost_per_1m, model.output_cost_per_1m)} USD)
              </option>
            {/each}
          {/if}
        </select>
      </div>

      {#if availableAIModels}
        <p class="text-xs text-muted-foreground">
          {$_('page.settings.ai.models_hint')} ({$_('page.settings.ai.model_price_estimate', {
            values: { date: availableAIModels.catalog_version },
          })})
        </p>
      {:else}
        <p class="text-xs text-muted-foreground">{$_('page.settings.ai.models_hint')}</p>
      {/if}

      <button type="submit" disabled={isSavingAIModels} class="ui-button ui-button-primary">
        {#if isSavingAIModels}
          <Loader2 size={16} class="animate-spin" />
          {$_('common.saving')}
        {:else}
          {$_('page.settings.ai.save_models_button')}
        {/if}
      </button>
    </form>
  </div>

  <!-- Claude API Key (BYOK) -->
  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">{$_('page.settings.ai.claude_title')}</h3>
    <p class="text-sm text-muted-foreground mb-4">{$_('page.settings.ai.claude_description')}</p>

    {#if isLoadingClaudeKeyStatus}
      <div class="ui-panel p-4 flex items-center gap-3">
        <Loader2 size={20} class="animate-spin text-muted-foreground" />
        <span class="text-muted-foreground">{$_('common.loading')}</span>
      </div>
    {:else if claudeApiKeyStatus?.has_key}
      <div class="ui-alert ui-alert-success">
        <div class="flex items-start gap-3">
          <Key size={20} class="text-success mt-0.5" />
          <div class="flex-1">
            <div class="font-medium text-success mb-1">
              {$_('page.settings.ai.api_key_configured')}
            </div>
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
            class="ui-icon-button p-2 text-destructive hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
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

      <details class="mt-4">
        <summary class="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
          {$_('page.settings.ai.update_api_key')}
        </summary>
        <form onsubmit={handleSaveClaudeApiKey} class="mt-4 ui-fieldset">
          <div class="ui-form-row">
            <label for="claude-api-key-update" class="ui-label mb-0">
              {$_('page.settings.ai.new_api_key_label')}
            </label>
            <div class="relative">
              <input
                id="claude-api-key-update"
                type={claudeKeyForm.showKey ? 'text' : 'password'}
                bind:value={claudeKeyForm.apiKey}
                disabled={claudeKeyForm.isSaving}
                class="ui-input w-full pr-10 font-mono"
                placeholder="sk-ant-api03-..."
              />
              <button
                type="button"
                onclick={() => (claudeKeyForm.showKey = !claudeKeyForm.showKey)}
                class="ui-icon-button absolute right-2 top-1/2 -translate-y-1/2 p-1 border-0 bg-transparent"
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
            class="ui-button ui-button-primary"
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
      <form onsubmit={handleSaveClaudeApiKey} class="ui-fieldset">
        <div class="ui-form-row">
          <label for="claude-api-key" class="ui-label mb-0">
            {$_('page.settings.ai.api_key_label')}
          </label>
          <div class="relative">
            <input
              id="claude-api-key"
              type={claudeKeyForm.showKey ? 'text' : 'password'}
              bind:value={claudeKeyForm.apiKey}
              disabled={claudeKeyForm.isSaving}
              class="ui-input w-full pr-10 font-mono"
              placeholder="sk-ant-api03-..."
            />
            <button
              type="button"
              onclick={() => (claudeKeyForm.showKey = !claudeKeyForm.showKey)}
              class="ui-icon-button absolute right-2 top-1/2 -translate-y-1/2 p-1 border-0 bg-transparent"
            >
              {#if claudeKeyForm.showKey}
                <EyeOff size={18} />
              {:else}
                <Eye size={18} />
              {/if}
            </button>
          </div>
          <p class="ui-form-help mt-1">{$_('page.settings.ai.claude_key_hint')}</p>
        </div>

        {#if claudeKeyForm.error}
          <div class="text-sm text-red-500">{claudeKeyForm.error}</div>
        {/if}

        <button type="submit" disabled={claudeKeyForm.isSaving} class="ui-button ui-button-primary">
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
      <div class="ui-panel p-4 flex items-center gap-3">
        <Loader2 size={20} class="animate-spin text-muted-foreground" />
        <span class="text-muted-foreground">{$_('common.loading')}</span>
      </div>
    {:else if geminiApiKeyStatus?.has_key}
      <div class="ui-alert ui-alert-success">
        <div class="flex items-start gap-3">
          <Key size={20} class="text-success mt-0.5" />
          <div class="flex-1">
            <div class="font-medium text-success mb-1">
              {$_('page.settings.ai.api_key_configured')}
            </div>
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
            class="ui-icon-button p-2 text-destructive hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
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

      <details class="mt-4">
        <summary class="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
          {$_('page.settings.ai.update_api_key')}
        </summary>
        <form onsubmit={handleSaveGeminiApiKey} class="mt-4 ui-fieldset">
          <div class="ui-form-row">
            <label for="gemini-api-key-update" class="ui-label mb-0">
              {$_('page.settings.ai.new_api_key_label')}
            </label>
            <div class="relative">
              <input
                id="gemini-api-key-update"
                type={geminiKeyForm.showKey ? 'text' : 'password'}
                bind:value={geminiKeyForm.apiKey}
                disabled={geminiKeyForm.isSaving}
                class="ui-input w-full pr-10 font-mono"
                placeholder="AIzaSy..."
              />
              <button
                type="button"
                onclick={() => (geminiKeyForm.showKey = !geminiKeyForm.showKey)}
                class="ui-icon-button absolute right-2 top-1/2 -translate-y-1/2 p-1 border-0 bg-transparent"
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
            class="ui-button ui-button-primary"
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
      <form onsubmit={handleSaveGeminiApiKey} class="ui-fieldset">
        <div class="ui-form-row">
          <label for="gemini-api-key" class="ui-label mb-0">
            {$_('page.settings.ai.api_key_label')}
          </label>
          <div class="relative">
            <input
              id="gemini-api-key"
              type={geminiKeyForm.showKey ? 'text' : 'password'}
              bind:value={geminiKeyForm.apiKey}
              disabled={geminiKeyForm.isSaving}
              class="ui-input w-full pr-10 font-mono"
              placeholder="AIzaSy..."
            />
            <button
              type="button"
              onclick={() => (geminiKeyForm.showKey = !geminiKeyForm.showKey)}
              class="ui-icon-button absolute right-2 top-1/2 -translate-y-1/2 p-1 border-0 bg-transparent"
            >
              {#if geminiKeyForm.showKey}
                <EyeOff size={18} />
              {:else}
                <Eye size={18} />
              {/if}
            </button>
          </div>
          <p class="ui-form-help mt-1">{$_('page.settings.ai.gemini_key_hint')}</p>
        </div>

        {#if geminiKeyForm.error}
          <div class="text-sm text-red-500">{geminiKeyForm.error}</div>
        {/if}

        <button type="submit" disabled={geminiKeyForm.isSaving} class="ui-button ui-button-primary">
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

  <!-- ChatGPT API Key (BYOK) -->
  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">{$_('page.settings.ai.openai_title')}</h3>
    <p class="text-sm text-muted-foreground mb-4">{$_('page.settings.ai.openai_description')}</p>

    {#if isLoadingOpenAIKeyStatus}
      <div class="ui-panel p-4 flex items-center gap-3">
        <Loader2 size={20} class="animate-spin text-muted-foreground" />
        <span class="text-muted-foreground">{$_('common.loading')}</span>
      </div>
    {:else if openAIApiKeyStatus?.has_key}
      <div class="ui-alert ui-alert-success">
        <div class="flex items-start gap-3">
          <Key size={20} class="text-success mt-0.5" />
          <div class="flex-1">
            <div class="font-medium text-success mb-1">
              {$_('page.settings.ai.api_key_configured')}
            </div>
            <div class="text-sm text-success font-mono">{openAIApiKeyStatus.masked_key}</div>
            {#if openAIApiKeyStatus.updated_at}
              <div class="text-xs text-success mt-2">
                {$_('page.settings.ai.api_key_updated_at', {
                  values: { date: new Date(openAIApiKeyStatus.updated_at).toLocaleDateString() },
                })}
              </div>
            {/if}
          </div>
          <button
            onclick={handleDeleteOpenAIApiKey}
            disabled={openAIKeyForm.isDeleting}
            class="ui-icon-button p-2 text-destructive hover:bg-destructive/10 hover:text-destructive disabled:opacity-50"
            title={$_('page.settings.ai.delete_api_key')}
          >
            {#if openAIKeyForm.isDeleting}
              <Loader2 size={18} class="animate-spin" />
            {:else}
              <Trash2 size={18} />
            {/if}
          </button>
        </div>
      </div>

      <details class="mt-4">
        <summary class="text-sm text-muted-foreground cursor-pointer hover:text-foreground">
          {$_('page.settings.ai.update_api_key')}
        </summary>
        <form onsubmit={handleSaveOpenAIApiKey} class="mt-4 ui-fieldset">
          <div class="ui-form-row">
            <label for="openai-api-key-update" class="ui-label mb-0">
              {$_('page.settings.ai.new_api_key_label')}
            </label>
            <div class="relative">
              <input
                id="openai-api-key-update"
                type={openAIKeyForm.showKey ? 'text' : 'password'}
                bind:value={openAIKeyForm.apiKey}
                disabled={openAIKeyForm.isSaving}
                class="ui-input w-full pr-10 font-mono"
                placeholder="sk-proj-..."
              />
              <button
                type="button"
                onclick={() => (openAIKeyForm.showKey = !openAIKeyForm.showKey)}
                class="ui-icon-button absolute right-2 top-1/2 -translate-y-1/2 p-1 border-0 bg-transparent"
              >
                {#if openAIKeyForm.showKey}
                  <EyeOff size={18} />
                {:else}
                  <Eye size={18} />
                {/if}
              </button>
            </div>
          </div>

          {#if openAIKeyForm.error}
            <div class="text-sm text-red-500">{openAIKeyForm.error}</div>
          {/if}

          <button
            type="submit"
            disabled={openAIKeyForm.isSaving}
            class="ui-button ui-button-primary"
          >
            {#if openAIKeyForm.isSaving}
              <Loader2 size={16} class="animate-spin" />
              {$_('common.saving')}
            {:else}
              {$_('page.settings.ai.update_api_key_button')}
            {/if}
          </button>
        </form>
      </details>
    {:else}
      <form onsubmit={handleSaveOpenAIApiKey} class="ui-fieldset">
        <div class="ui-form-row">
          <label for="openai-api-key" class="ui-label mb-0">
            {$_('page.settings.ai.api_key_label')}
          </label>
          <div class="relative">
            <input
              id="openai-api-key"
              type={openAIKeyForm.showKey ? 'text' : 'password'}
              bind:value={openAIKeyForm.apiKey}
              disabled={openAIKeyForm.isSaving}
              class="ui-input w-full pr-10 font-mono"
              placeholder="sk-proj-..."
            />
            <button
              type="button"
              onclick={() => (openAIKeyForm.showKey = !openAIKeyForm.showKey)}
              class="ui-icon-button absolute right-2 top-1/2 -translate-y-1/2 p-1 border-0 bg-transparent"
            >
              {#if openAIKeyForm.showKey}
                <EyeOff size={18} />
              {:else}
                <Eye size={18} />
              {/if}
            </button>
          </div>
          <p class="ui-form-help mt-1">{$_('page.settings.ai.openai_key_hint')}</p>
        </div>

        {#if openAIKeyForm.error}
          <div class="text-sm text-red-500">{openAIKeyForm.error}</div>
        {/if}

        <button type="submit" disabled={openAIKeyForm.isSaving} class="ui-button ui-button-primary">
          {#if openAIKeyForm.isSaving}
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

  <div class="ui-panel p-4 bg-primary/10 border-primary/30">
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

  <div class="ui-alert ui-alert-warning">
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
