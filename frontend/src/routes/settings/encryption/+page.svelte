<script lang="ts">
  import DOMPurify from 'isomorphic-dompurify';
  import { AlertTriangle, Eye, Key, Lock, Shield } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';

  const settings = $derived(encryption.getSettings());
  let showKeywordWarning = $state(false);
  let showTitleWarning = $state(false);
  const sanitize = (value: string) => DOMPurify.sanitize(value);

  function toggleKeywords() {
    if (!settings.extractKeywords) {
      // User wants to enable - show warning first
      showKeywordWarning = true;
    } else {
      // User wants to disable - do it immediately
      encryption.updateSettings({ extractKeywords: false });
    }
  }

  function confirmKeywords() {
    encryption.updateSettings({ extractKeywords: true });
    showKeywordWarning = false;
  }

  function toggleTitles() {
    if (!settings.encryptTitles) {
      // User wants to enable - show info
      showTitleWarning = true;
    } else {
      // User wants to disable - do it immediately
      encryption.updateSettings({ encryptTitles: false });
    }
  }

  function confirmTitles() {
    encryption.updateSettings({ encryptTitles: true });
    showTitleWarning = false;
  }

  const isUnlocked = $derived(encryption.isEncryptionUnlocked());
</script>

<svelte:head>
  <title>{$_('page.settings.encryption.page_title')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto p-6">
  <!-- Header -->
  <div class="mb-8">
    <div class="flex items-center gap-2 sm:gap-3 mb-2">
      <MobileSidebarInlineToggle />
      <Shield class="w-8 h-8 text-primary" />
      <h1 class="text-3xl font-bold">{$_('page.settings.encryption.title')}</h1>
    </div>
    <p class="text-muted-foreground">
      {$_('page.settings.encryption.subtitle')}
    </p>
  </div>

  <!-- Encryption Status -->
  <div
    class="mb-6 p-4 rounded-lg border {isUnlocked
      ? 'bg-success/10 border-success/30'
      : 'bg-yellow-50 border-yellow-200 dark:bg-yellow-900/20 dark:border-yellow-800'}"
  >
    <div class="flex items-center gap-2 mb-2">
      {#if isUnlocked}
        <Lock class="w-5 h-5 text-success" />
        <span class="font-semibold text-success"
          >{$_('page.settings.encryption.status_active')}</span
        >
      {:else}
        <AlertTriangle class="w-5 h-5 text-yellow-600 dark:text-yellow-400" />
        <span class="font-semibold text-yellow-800 dark:text-yellow-300"
          >{$_('page.settings.encryption.status_locked')}</span
        >
      {/if}
    </div>
    <p class="text-sm text-muted-foreground">
      {#if isUnlocked}
        {$_('page.settings.encryption.status_active_description')}
      {:else}
        {$_('page.settings.encryption.status_locked_description')}
      {/if}
    </p>
  </div>

  <!-- Settings Sections -->
  <div class="space-y-6">
    <!-- Title Encryption -->
    <div class="bg-card p-6 rounded-lg border border-border">
      <div class="flex items-start justify-between mb-4">
        <div class="flex-1">
          <div class="flex items-center gap-2 mb-2">
            <Eye class="w-5 h-5 text-muted-foreground" />
            <h3 class="text-lg font-semibold">
              {$_('page.settings.encryption.title_encryption_heading')}
            </h3>
          </div>
          <p class="text-sm text-muted-foreground mb-3">
            {$_('page.settings.encryption.title_encryption_description')}
          </p>
          <div class="flex items-start gap-2 p-3 bg-primary/10 rounded-md">
            <AlertTriangle class="w-4 h-4 text-primary mt-0.5 flex-shrink-0" />
            <p class="text-xs text-foreground">
              {@html sanitize($_('page.settings.encryption.title_encryption_info'))}
            </p>
          </div>
        </div>
        <label class="relative inline-flex items-center cursor-pointer ml-4">
          <input
            type="checkbox"
            class="sr-only peer"
            checked={settings.encryptTitles}
            disabled={!isUnlocked}
            onchange={toggleTitles}
          />
          <div
            class="w-11 h-6 bg-muted peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-primary/30 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary peer-disabled:opacity-50 peer-disabled:cursor-not-allowed"
          ></div>
        </label>
      </div>
    </div>

    <!-- Keyword Extraction -->
    <div class="bg-card p-6 rounded-lg border border-border">
      <div class="flex items-start justify-between mb-4">
        <div class="flex-1">
          <div class="flex items-center gap-2 mb-2">
            <Key class="w-5 h-5 text-muted-foreground" />
            <h3 class="text-lg font-semibold">{$_('page.settings.encryption.keyword_heading')}</h3>
          </div>
          <p class="text-sm text-muted-foreground mb-3">
            {$_('page.settings.encryption.keyword_description')}
          </p>
          <div class="flex items-start gap-2 p-3 bg-red-50 dark:bg-red-900/20 rounded-md">
            <AlertTriangle class="w-4 h-4 text-red-600 dark:text-red-400 mt-0.5 flex-shrink-0" />
            <div class="text-xs text-red-800 dark:text-red-300">
              <p class="font-semibold mb-1">
                {$_('page.settings.encryption.keyword_warning_title')}
              </p>
              <p class="mb-2">
                {@html sanitize($_('page.settings.encryption.keyword_warning_text'))}
              </p>
              <p class="mb-1">
                {@html sanitize($_('page.settings.encryption.keyword_warning_example'))}
              </p>
              <p class="text-red-900 dark:text-red-200 font-semibold">
                {$_('page.settings.encryption.keyword_warning_final')}
              </p>
            </div>
          </div>
        </div>
        <label class="relative inline-flex items-center cursor-pointer ml-4">
          <input
            type="checkbox"
            class="sr-only peer"
            checked={settings.extractKeywords}
            disabled={!isUnlocked}
            onchange={toggleKeywords}
          />
          <div
            class="w-11 h-6 bg-muted peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-destructive/30 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-destructive peer-disabled:opacity-50 peer-disabled:cursor-not-allowed"
          ></div>
        </label>
      </div>
    </div>

    <!-- What is Encrypted -->
    <div class="bg-card p-6 rounded-lg border border-border">
      <h3 class="text-lg font-semibold mb-4">
        {$_('page.settings.encryption.what_encrypted_heading')}
      </h3>

      <div class="space-y-4">
        <div>
          <h4 class="font-semibold text-success mb-2">
            {$_('page.settings.encryption.protected_heading')}
          </h4>
          <ul class="list-disc list-inside space-y-1 text-sm text-muted-foreground">
            <li>
              {@html sanitize($_('page.settings.encryption.protected_content'))}
            </li>
            <li>
              {@html sanitize($_('page.settings.encryption.protected_title'))}
            </li>
          </ul>
        </div>

        <div>
          <h4 class="font-semibold text-yellow-700 dark:text-yellow-400 mb-2">
            {$_('page.settings.encryption.visible_heading')}
          </h4>
          <ul class="list-disc list-inside space-y-1 text-sm text-muted-foreground">
            <li>{@html sanitize($_('page.settings.encryption.visible_title'))}</li>
            <li>{@html sanitize($_('page.settings.encryption.visible_folders'))}</li>
            <li>
              {@html sanitize($_('page.settings.encryption.visible_metadata'))}
            </li>
            <li>{@html sanitize($_('page.settings.encryption.visible_keywords'))}</li>
          </ul>
        </div>
      </div>
    </div>

    <!-- Recommendations -->
    <div class="bg-primary/10 p-6 rounded-lg border border-primary/30">
      <h3 class="text-lg font-semibold mb-3 flex items-center gap-2">
        <Shield class="w-5 h-5 text-primary" />
        {$_('page.settings.encryption.recommendations_heading')}
      </h3>
      <ul class="space-y-2 text-sm text-muted-foreground">
        <li class="flex items-start gap-2">
          <span class="text-success font-bold mt-0.5">&#10003;</span>
          <span>{$_('page.settings.encryption.recommendation_titles')}</span>
        </li>
        <li class="flex items-start gap-2">
          <span class="text-success font-bold mt-0.5">&#10003;</span>
          <span>{$_('page.settings.encryption.recommendation_keywords')}</span>
        </li>
        <li class="flex items-start gap-2">
          <span class="text-success font-bold mt-0.5">&#10003;</span>
          <span>{$_('page.settings.encryption.recommendation_folders')}</span>
        </li>
        <li class="flex items-start gap-2">
          <span class="text-success font-bold mt-0.5">&#10003;</span>
          <span>{$_('page.settings.encryption.recommendation_recovery')}</span>
        </li>
      </ul>
    </div>

    <!-- Important Warning -->
    <div
      class="bg-red-50 dark:bg-red-900/20 p-6 rounded-lg border border-red-200 dark:border-red-800"
    >
      <div class="flex items-start gap-3">
        <AlertTriangle class="w-6 h-6 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
        <div>
          <h3 class="text-lg font-semibold mb-2 text-red-800 dark:text-red-300">
            {$_('page.settings.encryption.password_loss_heading')}
          </h3>
          <p class="text-sm text-red-800 dark:text-red-300">
            {@html sanitize($_('page.settings.encryption.password_loss_text'))}
          </p>
        </div>
      </div>
    </div>
  </div>
</div>

<!-- Title Encryption Confirmation Modal -->
{#if showTitleWarning}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="bg-card rounded-lg p-6 max-w-md w-full">
      <h3 class="text-xl font-semibold mb-3 flex items-center gap-2">
        <Eye class="w-6 h-6 text-primary" />
        {$_('page.settings.encryption.modal_title_encryption_title')}
      </h3>
      <div class="space-y-3 mb-6">
        <p class="text-sm text-muted-foreground">
          {$_('page.settings.encryption.modal_title_encryption_intro')}
        </p>
        <ul class="list-disc list-inside space-y-1 text-sm text-muted-foreground ml-2">
          <li>{$_('page.settings.encryption.modal_title_encryption_item1')}</li>
          <li>{$_('page.settings.encryption.modal_title_encryption_item2')}</li>
          <li>{$_('page.settings.encryption.modal_title_encryption_item3')}</li>
        </ul>
        <p class="text-sm text-muted-foreground">
          {$_('page.settings.encryption.modal_title_encryption_note')}
        </p>
      </div>
      <div class="flex gap-3">
        <button
          class="flex-1 px-4 py-2 bg-muted rounded-md hover:bg-muted/80"
          onclick={() => (showTitleWarning = false)}
        >
          {$_('common.cancel')}
        </button>
        <button
          class="flex-1 px-4 py-2 bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
          onclick={confirmTitles}
        >
          {$_('page.settings.encryption.modal_title_encryption_confirm')}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Keyword Extraction Warning Modal -->
{#if showKeywordWarning}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50">
    <div class="bg-card rounded-lg p-6 max-w-md w-full">
      <h3 class="text-xl font-semibold mb-3 flex items-center gap-2 text-red-700 dark:text-red-400">
        <AlertTriangle class="w-6 h-6" />
        {$_('page.settings.encryption.modal_keyword_title')}
      </h3>
      <div class="space-y-3 mb-6">
        <p class="text-sm text-muted-foreground">
          {@html sanitize($_('page.settings.encryption.modal_keyword_warning'))}
        </p>
        <div class="p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-md">
          <p class="text-xs text-yellow-900 dark:text-yellow-200">
            {@html sanitize($_('page.settings.encryption.modal_keyword_example'))}
          </p>
        </div>
        <p class="text-sm font-semibold text-red-700 dark:text-red-400">
          {$_('page.settings.encryption.modal_keyword_final_warning')}
        </p>
      </div>
      <div class="flex gap-3">
        <button
          class="flex-1 px-4 py-2 bg-muted rounded-md hover:bg-muted/80"
          onclick={() => (showKeywordWarning = false)}
        >
          {$_('common.cancel')}
        </button>
        <button
          class="flex-1 px-4 py-2 bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90"
          onclick={confirmKeywords}
        >
          {$_('page.settings.encryption.modal_keyword_confirm')}
        </button>
      </div>
    </div>
  </div>
{/if}
