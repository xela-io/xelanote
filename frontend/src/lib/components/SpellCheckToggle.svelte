<script lang="ts">
  import { SpellCheck, ChevronDown, Loader2 } from 'lucide-svelte';
  import type { EditorView } from '@codemirror/view';
  import {
    toggleSpellCheck,
    setSpellCheckLanguage,
    isSpellCheckEnabled,
    getSpellCheckLanguage,
  } from '$lib/editor/codemirror';
  import { _ } from 'svelte-i18n';

  interface Props {
    editorView: EditorView | undefined;
  }

  const { editorView }: Props = $props();

  let enabled = $state(false);
  let language = $state<'de' | 'en'>('en');
  let showDropdown = $state(false);
  let checking = $state(false);

  // Sync state when editorView changes
  $effect(() => {
    if (editorView) {
      enabled = isSpellCheckEnabled(editorView);
      language = getSpellCheckLanguage(editorView);
    }
  });

  function handleToggle() {
    if (!editorView) return;

    enabled = !enabled;
    toggleSpellCheck(editorView, enabled);

    // Show brief checking indicator when enabling
    if (enabled) {
      checking = true;
      setTimeout(() => {
        checking = false;
      }, 2500);
    }
  }

  function handleLanguageChange(lang: 'de' | 'en') {
    if (!editorView) return;

    language = lang;
    setSpellCheckLanguage(editorView, lang);
    showDropdown = false;

    // Trigger recheck
    if (enabled) {
      checking = true;
      setTimeout(() => {
        checking = false;
      }, 2500);
    }
  }

  function handleClickOutside(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (!target.closest('.spell-check-dropdown')) {
      showDropdown = false;
    }
  }

  $effect(() => {
    if (showDropdown) {
      document.addEventListener('click', handleClickOutside);
      return () => document.removeEventListener('click', handleClickOutside);
    }
  });
</script>

<div class="spell-check-dropdown relative flex items-center">
  <!-- Main toggle button -->
  <button
    type="button"
    onclick={handleToggle}
    disabled={!editorView}
    class="p-2 hover:bg-accent rounded-l-md disabled:opacity-50 transition-colors"
    class:bg-accent={enabled}
    aria-label={enabled ? $_('spellCheck.disable') : $_('spellCheck.enable')}
    aria-pressed={enabled}
  >
    {#if checking}
      <Loader2 size={16} class="animate-spin" />
    {:else}
      <SpellCheck size={16} class={enabled ? 'text-primary' : ''} />
    {/if}
  </button>

  <!-- Language dropdown trigger -->
  <button
    type="button"
    onclick={() => (showDropdown = !showDropdown)}
    disabled={!editorView}
    class="p-2 hover:bg-accent rounded-r-md border-l border-border disabled:opacity-50 transition-colors flex items-center gap-0.5"
    aria-label={$_('spellCheck.language')}
    aria-expanded={showDropdown}
  >
    <span class="text-xs uppercase font-medium">{language}</span>
    <ChevronDown size={12} />
  </button>

  <!-- Dropdown menu -->
  {#if showDropdown}
    <div
      class="absolute top-full right-0 mt-1 bg-popover border border-border rounded-md shadow-lg z-50 min-w-[120px]"
    >
      <button
        type="button"
        onclick={() => handleLanguageChange('en')}
        class="w-full px-3 py-2 text-sm text-left hover:bg-accent flex items-center gap-2"
        class:selected-language={language === 'en'}
      >
        <span class="w-4 text-center">{language === 'en' ? '✓' : ''}</span>
        {$_('spellCheck.english')}
      </button>
      <button
        type="button"
        onclick={() => handleLanguageChange('de')}
        class="w-full px-3 py-2 text-sm text-left hover:bg-accent flex items-center gap-2"
        class:selected-language={language === 'de'}
      >
        <span class="w-4 text-center">{language === 'de' ? '✓' : ''}</span>
        {$_('spellCheck.german')}
      </button>
    </div>
  {/if}
</div>

<style>
  .selected-language {
    background-color: color-mix(in oklch, var(--color-accent), transparent 50%);
  }
</style>
