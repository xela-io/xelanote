<script lang="ts">
  import { BookOpen, Search } from 'lucide-svelte';

  import * as ui from '$lib/stores/ui.svelte';

  interface Props {
    onClose: () => void;
  }

  const { onClose }: Props = $props();

  let searchQuery = $state('');

  // Quick reference items grouped by category
  const quickReferenceItems = [
    {
      category: 'Überschriften',
      items: [
        { syntax: '# Überschrift 1', description: 'Große Überschrift' },
        { syntax: '## Überschrift 2', description: 'Mittlere Überschrift' },
        { syntax: '### Überschrift 3', description: 'Kleine Überschrift' },
      ],
    },
    {
      category: 'Text-Formatierung',
      items: [
        { syntax: '**fett**', description: 'Fetter Text' },
        { syntax: '*kursiv*', description: 'Kursiver Text' },
        { syntax: '~~durchgestrichen~~', description: 'Durchgestrichener Text' },
      ],
    },
    {
      category: 'Listen',
      items: [
        { syntax: '- Punkt', description: 'Ungeordnete Liste' },
        { syntax: '1. Punkt', description: 'Geordnete Liste' },
        { syntax: '  - Unterpunkt', description: 'Verschachtelte Liste' },
      ],
    },
    {
      category: 'Links',
      items: [
        { syntax: '[Text](url)', description: 'Standard-Link' },
        { syntax: '[[Notiz-Titel]]', description: 'Wikilink' },
        { syntax: '[[Titel|Alias]]', description: 'Wikilink mit Alias' },
      ],
    },
    {
      category: 'Fälligkeitsdaten',
      items: [
        { syntax: '@due(2026-02-10)', description: 'Fälligkeitsdatum (farbiges Badge)' },
        { syntax: '- [ ] Aufgabe @due(2026-02-10)', description: 'Aufgabe mit Fälligkeit' },
      ],
    },
    {
      category: 'Code',
      items: [
        { syntax: '`code`', description: 'Inline-Code' },
        { syntax: '```\ncode\n```', description: 'Code-Block' },
        { syntax: '```js\ncode\n```', description: 'Code-Block mit Sprache' },
      ],
    },
    {
      category: 'Bilder',
      items: [
        { syntax: '![alt](url)', description: 'Bild einfügen' },
        { syntax: 'Drag & Drop', description: 'Bild hochladen' },
      ],
    },
  ];

  // Filter items based on search query
  const filteredItems = $derived.by(() => {
    if (!searchQuery.trim()) {
      return quickReferenceItems;
    }

    const query = searchQuery.toLowerCase();
    return quickReferenceItems
      .map((category) => ({
        ...category,
        items: category.items.filter(
          (item) =>
            item.syntax.toLowerCase().includes(query) ||
            item.description.toLowerCase().includes(query) ||
            category.category.toLowerCase().includes(query)
        ),
      }))
      .filter((category) => category.items.length > 0);
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    }
  }

  function handleBackdropClick(_e: MouseEvent) {
    // Close when clicking outside the dropdown
    onClose();
  }

  function handleDropdownClick(e: MouseEvent) {
    // Prevent clicks inside dropdown from closing it
    e.stopPropagation();
  }

  function openFullGuide() {
    onClose(); // Close dropdown
    ui.setMarkdownGuideOpen(true); // Open modal
  }

  async function copyToClipboard(text: string) {
    try {
      await navigator.clipboard.writeText(text);
    } catch (err) {
      console.error('Failed to copy:', err);
    }
  }
</script>

<!-- Backdrop (semi-transparent on mobile, invisible on desktop) -->
<div
  class="fixed inset-0 z-40 md:bg-transparent bg-black/50"
  onclick={handleBackdropClick}
  onkeydown={handleKeydown}
  tabindex="-1"
  role="presentation"
></div>

<!-- Dropdown: Bottom sheet on mobile, top-right on desktop -->
<div
  class="fixed z-50 bg-background border border-border shadow-lg flex flex-col
		md:top-16 md:right-4 md:w-96 md:rounded-lg md:max-h-[calc(var(--app-viewport-height,100dvh)-6rem)]
		bottom-0 left-0 right-0 max-h-[80vh] rounded-t-2xl animate-slide-up"
  onkeydown={handleKeydown}
  onclick={handleDropdownClick}
  role="dialog"
  aria-label="Markdown-Hilfe"
  tabindex="-1"
>
  <!-- Mobile handle bar -->
  <div class="md:hidden flex justify-center pt-2 pb-1">
    <div class="w-12 h-1 bg-muted-foreground/30 rounded-full"></div>
  </div>

  <!-- Search Field -->
  <div class="p-4 md:p-3 border-b border-border">
    <div class="relative">
      <Search
        size={18}
        class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground md:w-4 md:h-4"
      />
      <input
        type="text"
        bind:value={searchQuery}
        placeholder="Syntax suchen..."
        class="w-full pl-10 md:pl-9 pr-3 py-3 md:py-2 bg-background border border-border rounded-md text-base md:text-sm focus:outline-none focus:ring-2 focus:ring-ring"
      />
    </div>
  </div>

  <!-- Scrollable Content -->
  <div class="flex-1 overflow-y-auto p-4 md:p-3 space-y-5 md:space-y-4">
    {#if filteredItems.length === 0}
      <div class="text-center text-sm text-muted-foreground py-4">
        Keine Ergebnisse für "{searchQuery}"
      </div>
    {:else}
      {#each filteredItems as category (category.category)}
        <div class="space-y-2">
          <h3 class="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
            {category.category}
          </h3>
          <div class="space-y-1">
            {#each category.items as item (item.syntax)}
              <button
                type="button"
                class="w-full flex items-start justify-between gap-2 p-3 md:p-2 hover:bg-accent rounded-md cursor-pointer group active:bg-accent/80 transition-colors text-left"
                onclick={() => copyToClipboard(item.syntax)}
                title="Klicken zum Kopieren"
              >
                <div class="flex-1 min-w-0">
                  <code
                    class="text-sm md:text-xs font-mono block text-foreground break-all whitespace-pre-wrap"
                    >{item.syntax}</code
                  >
                  <span class="text-sm md:text-xs text-muted-foreground mt-1 block"
                    >{item.description}</span
                  >
                </div>
                <div
                  class="text-xs text-muted-foreground opacity-0 md:group-hover:opacity-100 [@media(pointer:coarse)]:opacity-100 transition-opacity shrink-0"
                >
                  Kopieren
                </div>
              </button>
            {/each}
          </div>
        </div>
      {/each}
    {/if}
  </div>

  <!-- Footer with Full Guide Link -->
  <div class="border-t border-border p-4 md:p-3">
    <button
      type="button"
      onclick={openFullGuide}
      class="w-full flex items-center justify-center gap-2 px-4 py-3 md:py-2 text-base md:text-sm font-medium text-primary hover:bg-accent active:bg-accent/80 rounded-md transition-colors"
    >
      <BookOpen size={18} class="md:w-4 md:h-4" />
      Vollständige Anleitung öffnen
    </button>
  </div>
</div>
