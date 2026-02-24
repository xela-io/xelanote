<script lang="ts">
  import { List, X } from 'lucide-svelte';

  import type { TocEntry } from '$lib/editor/markdown';

  interface Props {
    headings: TocEntry[];
    onHeadingClick?: (slug: string) => void;
    activeSlug?: string | null;
  }

  const { headings, onHeadingClick, activeSlug = null }: Props = $props();
  let isOpen = $state(false);
  let dropdownRef: HTMLDivElement | undefined = $state();
  let triggerButtonRef: HTMLButtonElement | undefined = $state();
  let dropdownPanelRef: HTMLElement | undefined = $state();
  let sheetDragOffset = $state(0);
  let isSheetDragging = $state(false);
  let touchStartY = 0;
  let touchStartTime = 0;
  let touchTracking = false;

  const minLevel = $derived(headings.length > 0 ? Math.min(...headings.map((h) => h.level)) : 1);

  function closeToc(options?: { restoreFocus?: boolean }) {
    isOpen = false;
    if (options?.restoreFocus) {
      requestAnimationFrame(() => {
        triggerButtonRef?.focus();
      });
    }
  }

  function handleClick(slug: string) {
    if (onHeadingClick) {
      onHeadingClick(slug);
    }
    closeToc();
  }

  function handleClickOutside(event: MouseEvent) {
    if (dropdownRef && !dropdownRef.contains(event.target as Node)) {
      closeToc();
    }
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      closeToc({ restoreFocus: true });
      return;
    }

    if (event.key !== 'Tab' || !isOpen || !dropdownPanelRef) return;

    const focusable = Array.from(
      dropdownPanelRef.querySelectorAll<HTMLElement>(
        'button:not([disabled]), [href], [tabindex]:not([tabindex="-1"])'
      )
    ).filter((el) => !el.hasAttribute('disabled'));

    if (focusable.length === 0) return;

    const first = focusable[0];
    const last = focusable[focusable.length - 1];
    const active = document.activeElement as HTMLElement | null;

    if (event.shiftKey && (active === first || active === dropdownPanelRef)) {
      event.preventDefault();
      last.focus();
      return;
    }

    if (!event.shiftKey && active === last) {
      event.preventDefault();
      first.focus();
    }
  }

  function isMobileSheetMode(): boolean {
    return typeof window !== 'undefined' && window.matchMedia('(max-width: 640px)').matches;
  }

  function handleSheetTouchStart(event: TouchEvent) {
    if (!isMobileSheetMode() || !dropdownPanelRef || event.touches.length !== 1) return;
    if (dropdownPanelRef.scrollTop > 2) return;
    touchTracking = true;
    isSheetDragging = false;
    sheetDragOffset = 0;
    touchStartY = event.touches[0].clientY;
    touchStartTime = performance.now();
  }

  function handleSheetTouchMove(event: TouchEvent) {
    if (!touchTracking || !dropdownPanelRef || event.touches.length !== 1) return;
    const deltaY = event.touches[0].clientY - touchStartY;
    if (deltaY <= 0) {
      sheetDragOffset = 0;
      isSheetDragging = false;
      return;
    }
    if (dropdownPanelRef.scrollTop > 2) {
      touchTracking = false;
      sheetDragOffset = 0;
      isSheetDragging = false;
      return;
    }
    isSheetDragging = true;
    sheetDragOffset = Math.min(deltaY, 120);
    if (event.cancelable) {
      event.preventDefault();
    }
  }

  function handleSheetTouchEnd() {
    if (!touchTracking) return;
    const elapsed = performance.now() - touchStartTime;
    const shouldClose = sheetDragOffset >= 72 || (sheetDragOffset >= 42 && elapsed < 220);
    touchTracking = false;
    isSheetDragging = false;
    sheetDragOffset = 0;
    if (shouldClose) {
      closeToc({ restoreFocus: true });
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

  $effect(() => {
    if (!isOpen || !activeSlug || !dropdownRef) return;
    const activeEntry = dropdownRef.querySelector<HTMLButtonElement>(
      `.toc-entry[data-slug="${CSS.escape(activeSlug)}"]`
    );
    activeEntry?.scrollIntoView({ block: 'nearest' });
  });

  $effect(() => {
    if (!isOpen || !dropdownPanelRef) return;
    requestAnimationFrame(() => {
      const preferred =
        dropdownPanelRef?.querySelector<HTMLButtonElement>('.toc-entry.active') ??
        dropdownPanelRef?.querySelector<HTMLButtonElement>('.toc-entry');
      preferred?.focus();
    });
  });

  function getIndent(level: number): number {
    const indent = Math.max(0, level - minLevel);
    return Math.min(indent, 4);
  }
</script>

{#if headings.length > 0}
  <div class="toc-floating" bind:this={dropdownRef}>
    <button
      bind:this={triggerButtonRef}
      onclick={() => (isOpen = !isOpen)}
      class="toc-trigger"
      class:open={isOpen}
      title="Inhaltsverzeichnis ({headings.length})"
      aria-label="Inhaltsverzeichnis ({headings.length})"
      aria-expanded={isOpen}
    >
      <span class="toc-trigger-icon" aria-hidden="true">
        {#if isOpen}
          <X size={16} />
        {:else}
          <List size={16} />
        {/if}
      </span>
      <span class="toc-trigger-label">Inhalt</span>
      <span class="toc-badge">{headings.length}</span>
    </button>

    {#if isOpen}
      <div
        class="toc-backdrop"
        aria-hidden="true"
        onclick={() => closeToc({ restoreFocus: true })}
      ></div>
      <nav
        class="toc-dropdown"
        class:sheet-dragging={isSheetDragging}
        bind:this={dropdownPanelRef}
        style={`--toc-sheet-drag-offset: ${sheetDragOffset}px;`}
        aria-label="Inhaltsverzeichnis"
        tabindex="-1"
        ontouchstart={handleSheetTouchStart}
        ontouchmove={handleSheetTouchMove}
        ontouchend={handleSheetTouchEnd}
        ontouchcancel={handleSheetTouchEnd}
      >
        <div class="toc-header">
          <div class="toc-header-title">
            <List size={14} />
            <span>Inhaltsverzeichnis</span>
          </div>
          <span class="toc-header-count">{headings.length}</span>
        </div>
        <ul class="toc-list">
          {#each headings as heading (heading.slug)}
            <li
              class="toc-item"
              data-level={heading.level}
              data-depth={getIndent(heading.level)}
              style={`--toc-indent: ${getIndent(heading.level)};`}
            >
              <button
                onclick={() => handleClick(heading.slug)}
                class="toc-entry"
                class:active={activeSlug === heading.slug}
                data-slug={heading.slug}
                title={heading.text}
                aria-current={activeSlug === heading.slug ? 'location' : undefined}
              >
                <span class="toc-entry-rail" aria-hidden="true"></span>
                <span class="toc-entry-text">{heading.text}</span>
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
    gap: 6px;
    padding: 5px 8px 5px 6px;
    border-radius: var(--radius-md);
    background: color-mix(in oklch, var(--surface-panel-bg, var(--color-card)) 86%, transparent);
    backdrop-filter: blur(10px);
    border: 1px solid color-mix(in oklch, var(--color-border), transparent 26%);
    color: var(--color-muted-foreground);
    cursor: pointer;
    transition: all var(--duration-fast) var(--ease-default);
    font-size: 0.75rem;
    box-shadow:
      0 1px 0 color-mix(in oklch, var(--color-border), transparent 40%),
      0 8px 22px -16px rgb(0 0 0 / 0.45);
  }

  .toc-trigger:hover {
    background: color-mix(
      in oklch,
      var(--color-accent),
      var(--surface-panel-bg, var(--color-card)) 58%
    );
    color: var(--color-foreground);
    border-color: color-mix(in oklch, var(--color-border), transparent 8%);
  }

  .toc-trigger.open {
    color: var(--color-foreground);
    background: color-mix(
      in oklch,
      var(--color-accent),
      var(--surface-panel-bg, var(--color-card)) 52%
    );
    border-color: color-mix(in oklch, var(--color-border), transparent 2%);
  }

  .toc-trigger-icon {
    width: 22px;
    height: 22px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: calc(var(--radius-sm) + 2px);
    background: color-mix(in oklch, var(--color-background), transparent 18%);
    color: inherit;
    flex-shrink: 0;
  }

  .toc-trigger.open .toc-trigger-icon {
    background: color-mix(in oklch, var(--color-accent), transparent 72%);
  }

  .toc-trigger-label {
    font-weight: 600;
    letter-spacing: 0.01em;
    line-height: 1;
  }

  .toc-badge {
    font-size: 0.65rem;
    font-weight: 600;
    min-width: 18px;
    height: 18px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 9999px; /* rounded-full */
    background: color-mix(in oklch, var(--color-muted), var(--color-background) 30%);
    color: var(--color-foreground);
    border: 1px solid color-mix(in oklch, var(--color-border), transparent 35%);
    line-height: 1;
  }

  .toc-dropdown {
    position: absolute;
    top: calc(100% + 8px);
    right: 0;
    width: min(340px, calc(100vw - 24px));
    max-height: min(440px, 62vh);
    overflow-y: auto;
    background: linear-gradient(
      180deg,
      color-mix(in oklch, var(--surface-panel-bg-contrast, var(--color-card)) 88%, transparent) 0%,
      color-mix(in oklch, var(--surface-panel-bg, var(--color-card)) 94%, transparent) 100%
    );
    border: 1px solid color-mix(in oklch, var(--color-border), transparent 18%);
    border-radius: calc(var(--radius-lg) + 2px);
    box-shadow:
      0 18px 40px -24px rgb(0 0 0 / 0.65),
      0 8px 20px -16px rgb(0 0 0 / 0.45),
      inset 0 1px 0 rgb(255 255 255 / 0.05);
    backdrop-filter: blur(14px);
    -webkit-overflow-scrolling: touch;
    --toc-sheet-drag-offset: 0px;
  }

  .toc-backdrop {
    display: none;
    position: fixed;
    inset: 0;
    background: rgb(0 0 0 / 0.22);
    backdrop-filter: blur(2px);
    opacity: 0;
    animation: toc-backdrop-in 140ms ease-out forwards;
  }

  .toc-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 10px 12px 9px;
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--color-foreground);
    border-bottom: 1px solid color-mix(in oklch, var(--color-border), transparent 28%);
    background: color-mix(in oklch, var(--color-background), transparent 78%);
    position: sticky;
    top: 0;
    z-index: 1;
    backdrop-filter: blur(12px);
  }

  .toc-header-title {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .toc-header-count {
    font-size: 0.72rem;
    line-height: 1;
    padding: 4px 7px;
    border-radius: 9999px;
    color: var(--color-muted-foreground);
    background: color-mix(in oklch, var(--color-muted), transparent 35%);
    border: 1px solid color-mix(in oklch, var(--color-border), transparent 34%);
  }

  .toc-list {
    list-style: none;
    padding: 8px;
    margin: 0;
  }

  .toc-list li,
  .toc-item {
    margin: 0;
  }

  .toc-item {
    --toc-indent-step: 12px;
  }

  .toc-entry {
    display: grid;
    grid-template-columns: 10px minmax(0, 1fr);
    align-items: start;
    column-gap: 8px;
    width: 100%;
    text-align: left;
    padding: 7px 10px 7px calc(8px + (var(--toc-indent) * var(--toc-indent-step)));
    border: 1px solid transparent;
    border-radius: calc(var(--radius-sm) + 2px);
    background: transparent;
    color: var(--color-muted-foreground);
    font-size: 0.84rem;
    line-height: 1.25;
    cursor: pointer;
    transition: all var(--duration-fast) var(--ease-default);
  }

  .toc-entry:hover {
    background: color-mix(in oklch, var(--color-accent), transparent 78%);
    color: var(--color-foreground);
    border-color: color-mix(in oklch, var(--color-border), transparent 22%);
  }

  .toc-entry.active {
    background: color-mix(in oklch, var(--color-accent), transparent 70%);
    color: var(--color-foreground);
    border-color: color-mix(in oklch, var(--color-primary), var(--color-border) 68%);
    box-shadow: inset 0 0 0 1px color-mix(in oklch, var(--color-primary), transparent 72%);
  }

  .toc-entry-rail {
    width: 2px;
    min-height: 1.1em;
    margin-top: 0.1em;
    border-radius: 9999px;
    background: color-mix(in oklch, var(--color-border), transparent 24%);
    opacity: calc(1 - (var(--toc-indent) * 0.08));
  }

  .toc-entry-text {
    display: -webkit-box;
    line-clamp: 2;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    text-wrap: balance;
  }

  .toc-item[data-depth='0'] .toc-entry {
    font-weight: 650;
    color: var(--color-foreground);
  }

  .toc-item[data-depth='0'] .toc-entry-rail {
    background: color-mix(in oklch, var(--color-primary), var(--color-border) 65%);
    width: 3px;
  }

  .toc-entry.active .toc-entry-rail {
    width: 3px;
    background: var(--color-primary);
    opacity: 1;
  }

  .toc-item[data-depth='1'] .toc-entry {
    font-weight: 560;
  }

  .toc-item[data-level='3'] .toc-entry,
  .toc-item[data-level='4'] .toc-entry,
  .toc-item[data-level='5'] .toc-entry,
  .toc-item[data-level='6'] .toc-entry {
    font-size: 0.8rem;
  }

  .toc-trigger:focus-visible,
  .toc-entry:focus-visible {
    outline: 2px solid var(--color-ring);
    outline-offset: 2px;
  }

  @media (max-width: 640px) {
    .toc-floating {
      position: relative;
      top: auto;
      height: auto;
      z-index: auto;
      padding: 8px 8px 6px 0;
    }

    .toc-trigger-label {
      display: none;
    }

    .toc-backdrop {
      display: block;
      background: rgb(0 0 0 / 0.34);
      backdrop-filter: blur(4px);
    }

    .toc-dropdown {
      position: fixed;
      left: 8px;
      right: 8px;
      top: auto;
      bottom: max(8px, env(safe-area-inset-bottom));
      width: auto;
      max-height: min(68vh, 520px);
      border-radius: calc(var(--radius-lg) + 6px);
      box-shadow:
        0 28px 60px -28px rgb(0 0 0 / 0.7),
        0 16px 32px -20px rgb(0 0 0 / 0.5),
        inset 0 1px 0 rgb(255 255 255 / 0.06);
      animation: toc-sheet-in 170ms var(--ease-default, ease-out);
      transform: translateY(var(--toc-sheet-drag-offset));
      transition:
        transform 160ms var(--ease-default, ease-out),
        box-shadow 160ms var(--ease-default, ease-out);
      touch-action: pan-y;
    }

    .toc-dropdown.sheet-dragging {
      transition: none;
      box-shadow:
        0 18px 36px -24px rgb(0 0 0 / 0.55),
        0 10px 22px -18px rgb(0 0 0 / 0.4),
        inset 0 1px 0 rgb(255 255 255 / 0.06);
    }

    .toc-header {
      padding-top: 16px;
    }

    .toc-header::before {
      content: '';
      position: absolute;
      top: 6px;
      left: 50%;
      transform: translateX(-50%);
      width: 34px;
      height: 4px;
      border-radius: 9999px;
      background: color-mix(in oklch, var(--color-border), transparent 12%);
    }

    .toc-entry {
      padding-top: 9px;
      padding-bottom: 9px;
    }
  }

  @keyframes toc-sheet-in {
    from {
      opacity: 0;
      transform: translateY(8px) scale(0.985);
    }
    to {
      opacity: 1;
      transform: translateY(0) scale(1);
    }
  }

  @keyframes toc-backdrop-in {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .toc-backdrop,
    .toc-dropdown {
      animation: none !important;
    }

    .toc-dropdown {
      transition: none !important;
    }
  }
</style>
