<script lang="ts">
  import { Loader2 } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { FEATURE_FLAGS } from '$lib/config';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as settings from '$lib/stores/settings.svelte';
  import * as ui from '$lib/stores/ui.svelte';

  onMount(() => {
    features.loadJournalFeature();
    features.loadRecipeFeature();
    features.loadCanvasFeature();
    features.loadTabsFeature();
    features.loadShoppingFeature();
  });

  const editorModes = $derived([
    {
      id: 'edit' as const,
      label: $_('page.settings.editor.mode_edit_label'),
      description: $_('page.settings.editor.mode_edit_description'),
    },
    {
      id: 'preview' as const,
      label: $_('page.settings.editor.mode_preview_label'),
      description: $_('page.settings.editor.mode_preview_description'),
    },
    {
      id: 'split' as const,
      label: $_('page.settings.editor.mode_split_label'),
      description: $_('page.settings.editor.mode_split_description'),
    },
    ...(FEATURE_FLAGS.livePreview
      ? [
          {
            id: 'live' as const,
            label: $_('page.settings.editor.mode_live_label'),
            description: $_('page.settings.editor.mode_live_description'),
          },
        ]
      : []),
  ]);

  async function handleEditorModeChange(mode: 'edit' | 'preview' | 'split' | 'live') {
    await settings.setEditorModePreference(mode);
  }

  async function handleJournalToggle(enabled: boolean) {
    try {
      await features.toggleJournalFeature(enabled);
    } catch (error) {
      console.error('Failed to toggle journal feature:', error);
    }
  }

  async function handleRecipeToggle(enabled: boolean) {
    try {
      await features.toggleRecipeFeature(enabled);
    } catch (error) {
      console.error('Failed to toggle recipe feature:', error);
    }
  }

  async function handleCanvasToggle(enabled: boolean) {
    try {
      await features.toggleCanvasFeature(enabled);
    } catch (error) {
      console.error('Failed to toggle canvas feature:', error);
    }
  }

  async function handleTabsToggle(enabled: boolean) {
    try {
      await features.toggleTabsFeature(enabled);
    } catch (error) {
      console.error('Failed to toggle tabs feature:', error);
    }
  }

  async function handleShoppingToggle(enabled: boolean) {
    try {
      await features.toggleShoppingFeature(enabled);
    } catch (error) {
      console.error('Failed to toggle shopping feature:', error);
    }
  }
</script>

<div class="space-y-6">
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.editor.default_editor_mode')}
    </h3>
    <div class="space-y-2">
      {#each editorModes as mode (mode.id)}
        <button
          onclick={() => handleEditorModeChange(mode.id)}
          disabled={settings.getIsSavingPreferences()}
          class={`ui-select-card ui-select-card-primary w-full text-left ${
            ui.getEditorMode() === mode.id ? 'is-active' : ''
          }`}
        >
          <div
            class="w-5 h-5 rounded-full border-2 flex items-center justify-center
								{ui.getEditorMode() === mode.id ? 'border-primary' : 'border-muted-foreground'}"
          >
            {#if ui.getEditorMode() === mode.id}
              <div class="w-2.5 h-2.5 rounded-full bg-primary"></div>
            {/if}
          </div>
          <div class="flex-1">
            <div class="font-medium text-foreground">{mode.label}</div>
            <div class="text-sm text-muted-foreground">{mode.description}</div>
          </div>
        </button>
      {/each}
    </div>
  </section>

  <!-- Performance Settings (Experimental) -->
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.editor.performance_experimental_title')}
    </h3>
    <div class="space-y-4">
      <label class="ui-panel-soft flex items-start gap-3 p-4 cursor-pointer transition-colors">
        <input
          type="checkbox"
          checked={settings.getVirtualTreeEnabled()}
          onchange={(e) => settings.setVirtualTreeEnabled(e.currentTarget.checked)}
          class="mt-1"
        />
        <div class="flex-1">
          <div class="font-medium text-foreground">Virtual Tree Scrolling</div>
          <div class="text-sm text-muted-foreground mt-1">
            Verbessert die Performance bei 500+ Notizen durch virtuelles Scrollen (nur sichtbare
            Items werden gerendert).
            <strong class="text-orange-600 dark:text-orange-400">Hinweis:</strong> Drag-and-Drop ist auf
            sichtbare Items beschränkt.
          </div>
        </div>
      </label>
    </div>
  </section>

  <!-- Feature Toggles -->
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.editor.features_title')}
    </h3>
    <div class="space-y-4">
      <label class="ui-panel-soft flex items-start gap-3 p-4 cursor-pointer transition-colors">
        <input
          type="checkbox"
          checked={features.getJournalFeatureEnabled()}
          disabled={features.getJournalFeatureLoading()}
          onchange={(e) => handleJournalToggle(e.currentTarget.checked)}
          class="mt-1"
        />
        <div class="flex-1">
          <div class="font-medium text-foreground">
            {$_('page.settings.editor.journal_feature_title')}
          </div>
          <div class="text-sm text-muted-foreground mt-1">
            {$_('page.settings.editor.journal_feature_description')}
          </div>
        </div>
        {#if features.getJournalFeatureLoading()}
          <Loader2 size={16} class="animate-spin text-muted-foreground" />
        {/if}
      </label>

      <label class="ui-panel-soft flex items-start gap-3 p-4 cursor-pointer transition-colors">
        <input
          type="checkbox"
          checked={features.getRecipeFeatureEnabled()}
          disabled={features.getRecipeFeatureLoading()}
          onchange={(e) => handleRecipeToggle(e.currentTarget.checked)}
          class="mt-1"
        />
        <div class="flex-1">
          <div class="font-medium text-foreground">
            {$_('page.settings.editor.recipe_feature_title')}
          </div>
          <div class="text-sm text-muted-foreground mt-1">
            {$_('page.settings.editor.recipe_feature_description')}
          </div>
        </div>
        {#if features.getRecipeFeatureLoading()}
          <Loader2 size={16} class="animate-spin text-muted-foreground" />
        {/if}
      </label>

      <label class="ui-panel-soft flex items-start gap-3 p-4 cursor-pointer transition-colors">
        <input
          type="checkbox"
          checked={features.getCanvasFeatureEnabled()}
          disabled={features.getCanvasFeatureLoading()}
          onchange={(e) => handleCanvasToggle(e.currentTarget.checked)}
          class="mt-1"
        />
        <div class="flex-1">
          <div class="font-medium text-foreground">
            {$_('page.settings.editor.canvas_feature_title')}
          </div>
          <div class="text-sm text-muted-foreground mt-1">
            {$_('page.settings.editor.canvas_feature_description')}
          </div>
        </div>
        {#if features.getCanvasFeatureLoading()}
          <Loader2 size={16} class="animate-spin text-muted-foreground" />
        {/if}
      </label>

      <label class="ui-panel-soft flex items-start gap-3 p-4 cursor-pointer transition-colors">
        <input
          type="checkbox"
          checked={features.getTabsFeatureEnabled()}
          disabled={features.getTabsFeatureLoading()}
          onchange={(e) => handleTabsToggle(e.currentTarget.checked)}
          class="mt-1"
        />
        <div class="flex-1">
          <div class="font-medium text-foreground">
            {$_('page.settings.editor.tabs_feature_title')}
          </div>
          <div class="text-sm text-muted-foreground mt-1">
            {$_('page.settings.editor.tabs_feature_description')}
          </div>
        </div>
        {#if features.getTabsFeatureLoading()}
          <Loader2 size={16} class="animate-spin text-muted-foreground" />
        {/if}
      </label>

      <label class="ui-panel-soft flex items-start gap-3 p-4 cursor-pointer transition-colors">
        <input
          type="checkbox"
          checked={features.getShoppingFeatureEnabled()}
          disabled={features.getShoppingFeatureLoading()}
          onchange={(e) => handleShoppingToggle(e.currentTarget.checked)}
          class="mt-1"
        />
        <div class="flex-1">
          <div class="font-medium text-foreground">
            {$_('page.settings.editor.shopping_feature_title')}
          </div>
          <div class="text-sm text-muted-foreground mt-1">
            {$_('page.settings.editor.shopping_feature_description')}
          </div>
        </div>
        {#if features.getShoppingFeatureLoading()}
          <Loader2 size={16} class="animate-spin text-muted-foreground" />
        {/if}
      </label>

      {#if errorReporter.getServiceAvailable()}
        <label class="ui-panel-soft flex items-start gap-3 p-4 cursor-pointer transition-colors">
          <input
            type="checkbox"
            checked={errorReporter.isEnabled()}
            onchange={(e) => errorReporter.setEnabled(e.currentTarget.checked)}
            class="mt-1"
          />
          <div class="flex-1">
            <div class="font-medium text-foreground">{$_('feedback.settings_title')}</div>
            <div class="text-sm text-muted-foreground mt-1">
              {$_('feedback.settings_description')}
            </div>
          </div>
        </label>
      {/if}
    </div>
  </section>
</div>
