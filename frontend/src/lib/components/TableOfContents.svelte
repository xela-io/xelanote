<script lang="ts">
  import { List, X } from 'lucide-svelte';

  import type { TocEntry } from '$lib/editor/markdown';

  interface Props {
    headings: TocEntry[];
    onHeadingClick?: (slug: string) => void;
  }

  const { headings, onHeadingClick }: Props = $props();
  let isOpen = $state(false);
  let dropdownRef: HTMLDivElement | undefined = $state();

  const minLevel = $derived(headings.length > 0 ? Math.min(...headings.map((h) => h.level)) : 1);

  function handleClick(slug: string) {
    if (onHeadingClick) {
      onHeadingClick(slug);
    }
    isOpen = false;
  }

  function handleClickOutside(event: MouseEvent) {
    if (dropdownRef && !dropdownRef.contains(event.target as Node)) {
      isOpen = false;
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      isOpen = false;
    }
  }

  $effect(() => {
    if (isOpen) {
      document.addEventListener('click', handleClickOutside, true);
      document.addEventListener('keydown', handleKeydown);
      return () => {
        document.removeEventListener('click', handleClickOutside, true);
        document.removeEventListener('keydown', handleKeydown);
      };
    }
  });

  function getIndentClass(level: number): string {
    const indent = level - minLevel;
    switch (indent) {
      case 0:
        return '';
      case 1:
        return 'pl-3';
      case 2:
        return 'pl-6';
      case 3:
        return 'pl-9';
      case 4:
        return 'pl-12';
      default:
        return 'pl-12';
    }
  }
</script>

{#if headings.length > 0}
  <div class="toc-floating" bind:this={dropdownRef}>
    <button
      onclick={() => (isOpen = !isOpen)}
      class="toc-trigger"
      title="Inhaltsverzeichnis ({headings.length})"
      aria-label="Inhaltsverzeichnis ({headings.length})"
      aria-expanded={isOpen}
    >
      {#if isOpen}
        <X size={16} />
      {:else}
        <List size={16} />
      {/if}
      <span class="toc-badge">{headings.length}</span>
    </button>

    {#if isOpen}
      <nav class="toc-dropdown" aria-label="Inhaltsverzeichnis">
        <div class="toc-header">
          <List size={14} />
          <span>Inhaltsverzeichnis</span>
        </div>
        <ul class="toc-list">
          {#each headings as heading (heading.slug)}
            <li class={getIndentClass(heading.level)}>
              <button
                onclick={() => handleClick(heading.slug)}
                class="toc-entry"
                title={heading.text}
              >
                {heading.text}
              </button>
            </li>
          {/each}
        </ul>
      </nav>
    {/if}
  </div>
{/if}

<style>
  .toc-floating {
    position: sticky;
    top: 0;
    z-index: 20;
    height: 0;
    overflow: visible;
    display: flex;
    justify-content: flex-end;
    padding: 8px 8px 0 0;
    pointer-events: none;
  }

  .toc-floating :global(*) {
    pointer-events: auto;
  }

  .toc-trigger {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 8px;
    border-radius: 6px;
    background: color-mix(in oklch, var(--color-background) 85%, transparent);
    backdrop-filter: blur(8px);
    border: 1px solid var(--color-border);
    color: var(--color-muted-foreground);
    cursor: pointer;
    transition: all 0.15s ease;
    font-size: 0.75rem;
  }

  .toc-trigger:hover {
    background: var(--color-accent);
    color: var(--color-foreground);
  }

  .toc-badge {
    font-size: 0.65rem;
    font-weight: 600;
    min-width: 16px;
    height: 16px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 9999px;
    background: var(--color-muted);
    color: var(--color-muted-foreground);
  }

  .toc-dropdown {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    width: min(280px, calc(100vw - 32px));
    max-height: min(400px, 60vh);
    overflow-y: auto;
    background: var(--color-background);
    border: 1px solid var(--color-border);
    border-radius: 8px;
    box-shadow:
      0 4px 6px -1px rgb(0 0 0 / 0.1),
      0 2px 4px -2px rgb(0 0 0 / 0.1);
    -webkit-overflow-scrolling: touch;
  }

  .toc-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 10px 12px;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--color-foreground);
    border-bottom: 1px solid var(--color-border);
  }

  .toc-list {
    list-style: none;
    padding: 6px;
    margin: 0;
  }

  .toc-list li {
    margin: 0;
  }

  .toc-entry {
    display: block;
    width: 100%;
    text-align: left;
    padding: 4px 8px;
    border: none;
    border-radius: 4px;
    background: none;
    color: var(--color-muted-foreground);
    font-size: 0.8rem;
    cursor: pointer;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: all 0.1s ease;
  }

  .toc-entry:hover {
    background: var(--color-accent);
    color: var(--color-foreground);
  }
</style>
