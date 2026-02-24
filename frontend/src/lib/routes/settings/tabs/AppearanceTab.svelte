<script lang="ts">
  import { Check } from 'lucide-svelte';
  import { _, locale } from 'svelte-i18n';

  import * as settings from '$lib/stores/settings.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { type ThemeId, THEMES } from '$lib/themes';

  const themeList = Object.values(THEMES);
  const lightThemes = themeList.filter((t) => t.variant === 'light');
  const darkThemes = themeList.filter((t) => t.variant === 'dark');

  async function handleThemeChange(themeId: ThemeId) {
    await settings.setThemePreference(themeId);
  }
</script>

<div class="space-y-8">
  <!-- Language -->
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.appearance.language')}
    </h3>
    <select
      value={$locale}
      onchange={(e) => {
        const newLocale = (e.target as HTMLSelectElement).value;
        locale.set(newLocale);
        window.localStorage.setItem('locale', newLocale);
      }}
      class="ui-select"
    >
      <option value="de">Deutsch</option>
      <option value="en">English</option>
    </select>
  </section>

  <!-- Dark Themes -->
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.appearance.dark_themes')}
    </h3>
    <div class="grid grid-cols-2 md:grid-cols-3 gap-2 sm:gap-3">
      {#each darkThemes as theme (theme.id)}
        <button
          onclick={() => handleThemeChange(theme.id)}
          disabled={settings.getIsSavingPreferences()}
          class={`ui-select-card ui-select-card-primary relative text-left ${
            ui.getCurrentThemeId() === theme.id ? 'is-active' : ''
          }`}
        >
          {#if ui.getCurrentThemeId() === theme.id}
            <div class="absolute top-2 right-2 text-primary">
              <Check size={16} />
            </div>
          {/if}
          <div class="font-medium text-foreground text-sm">{theme.name}</div>
          {#if theme.description}
            <div class="hidden sm:block text-xs text-muted-foreground mt-1">
              {theme.description}
            </div>
          {/if}
        </button>
      {/each}
    </div>
  </section>

  <!-- Light Themes -->
  <section class="ui-form-section">
    <h3 class="ui-form-section-title">
      {$_('page.settings.appearance.light_themes')}
    </h3>
    <div class="grid grid-cols-2 md:grid-cols-3 gap-2 sm:gap-3">
      {#each lightThemes as theme (theme.id)}
        <button
          onclick={() => handleThemeChange(theme.id)}
          disabled={settings.getIsSavingPreferences()}
          class={`ui-select-card ui-select-card-primary relative text-left ${
            ui.getCurrentThemeId() === theme.id ? 'is-active' : ''
          }`}
        >
          {#if ui.getCurrentThemeId() === theme.id}
            <div class="absolute top-2 right-2 text-primary">
              <Check size={16} />
            </div>
          {/if}
          <div class="font-medium text-foreground text-sm">{theme.name}</div>
          {#if theme.description}
            <div class="hidden sm:block text-xs text-muted-foreground mt-1">
              {theme.description}
            </div>
          {/if}
        </button>
      {/each}
    </div>
  </section>
</div>
