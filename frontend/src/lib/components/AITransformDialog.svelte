<script lang="ts">
  import { diffLines } from 'diff';
  import { createFocusTrap, type FocusTrap } from 'focus-trap';
  import {
    AlertTriangle,
    Briefcase,
    Check,
    Coffee,
    Expand,
    FileText,
    Languages,
    Loader2,
    Pencil,
    Wand2,
    X,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { AIAction } from '$lib/api';
  import * as api from '$lib/api';
  import * as ui from '$lib/stores/ui.svelte';

  interface Props {
    noteId: string;
    action: AIAction;
    customPrompt?: string;
    originalText: string;
    selectionFrom: number;
    selectionTo: number;
    isFullContent: boolean;
    initialContentHash: string;
    getCurrentContent: () => string;
    onApply: (transformedText: string) => void;
    onClose: () => void;
  }

  const {
    noteId,
    action,
    customPrompt,
    originalText,
    selectionFrom: _selectionFrom,
    selectionTo: _selectionTo,
    isFullContent,
    initialContentHash,
    getCurrentContent,
    onApply,
    onClose,
  }: Props = $props();

  // States
  let loading = $state(true);
  let error = $state<string | null>(null);
  let transformedText = $state('');
  let noChanges = $state(false);
  let conflictDetected = $state(false);
  let applyingAnyway = $state(false);

  // Mobile detection
  const isMobile = $derived(ui.getIsMobile());

  // Icon map for dialog header
  const actionIcons: Record<AIAction, typeof Wand2> = {
    format: Wand2,
    summarize: FileText,
    expand: Expand,
    translate_de: Languages,
    translate_en: Languages,
    formal: Briefcase,
    informal: Coffee,
    custom: Pencil,
  };

  // Get the icon component for current action
  const ActionIcon = $derived(actionIcons[action] || Wand2);

  // Focus trap and focus restoration
  let dialogRef: HTMLDivElement | null = $state(null);
  let focusTrap: FocusTrap | null = null;
  let previousActiveElement: Element | null = null;

  // Compute diff between original and transformed
  const diffResult = $derived.by(() => {
    if (loading || error || !transformedText) return null;
    return diffLines(originalText, transformedText);
  });

  // Client-side hash for conflict detection (first 16 hex chars of SHA-256)
  async function computeContentHash(content: string): Promise<string> {
    const encoder = new TextEncoder();
    const data = encoder.encode(content);
    const hashBuffer = await crypto.subtle.digest('SHA-256', data);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    return hashArray
      .slice(0, 8)
      .map((b) => b.toString(16).padStart(2, '0'))
      .join('');
  }

  // Load transformed content on mount
  $effect(() => {
    transformContent();
  });

  async function transformContent() {
    loading = true;
    error = null;
    noChanges = false;

    try {
      const result = await api.aiTransform(noteId, action, originalText, customPrompt);
      transformedText = result;

      // Check if there are actual changes
      if (result.trim() === originalText.trim()) {
        noChanges = true;
      }
    } catch (e) {
      if (e instanceof api.ApiError) {
        // Map specific error codes to user-friendly messages
        switch (e.status) {
          case 412: // Precondition Failed - no provider
            error = $_('error.ai_transform.no_provider');
            break;
          case 403: // Forbidden - AI disabled
            error = $_('error.ai_transform.ai_disabled');
            break;
          case 413: // Too large
            error = $_('error.ai_transform.too_large');
            break;
          case 400: // Too short, empty, or invalid action
            if (e.message.includes('short')) {
              error = $_('error.ai_transform.too_short');
            } else if (e.message.includes('action')) {
              error = $_('error.ai_transform.invalid_action');
            } else if (e.message.includes('custom_prompt')) {
              error = $_('error.ai_transform.missing_prompt');
            } else {
              error = e.message;
            }
            break;
          case 504: // Gateway timeout
            error = $_('error.ai_transform.timeout');
            break;
          default:
            error = e.message;
        }
      } else {
        error = e instanceof Error ? e.message : $_('error.generic');
      }
    } finally {
      loading = false;
    }
  }

  async function handleApply() {
    if (loading || noChanges || !transformedText) return;

    // Check for conflict (content changed while dialog was open)
    const currentContent = getCurrentContent();
    const currentHash = await computeContentHash(currentContent);

    if (currentHash !== initialContentHash && !applyingAnyway) {
      conflictDetected = true;
      return;
    }

    onApply(transformedText);
    onClose();
  }

  function handleApplyAnyway() {
    applyingAnyway = true;
    conflictDetected = false;
    handleApply();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (conflictDetected) {
        conflictDetected = false;
      } else {
        onClose();
      }
    }
  }

  // Focus trap setup
  $effect(() => {
    if (dialogRef) {
      previousActiveElement = document.activeElement;

      focusTrap = createFocusTrap(dialogRef, {
        escapeDeactivates: false,
        allowOutsideClick: true,
        fallbackFocus: dialogRef,
        returnFocusOnDeactivate: false,
      });

      requestAnimationFrame(() => {
        focusTrap?.activate();
      });

      document.body.style.overflow = 'hidden';

      return () => {
        focusTrap?.deactivate();
        focusTrap = null;
        document.body.style.overflow = '';
        if (previousActiveElement instanceof HTMLElement) {
          previousActiveElement.focus();
        }
      };
    }
  });
</script>

<!-- Backdrop -->
<div
  class="fixed inset-0 bg-black/50 z-50"
  onclick={onClose}
  onkeydown={handleKeydown}
  tabindex="-1"
  role="presentation"
  aria-hidden="true"
></div>

<!-- Dialog -->
<div
  bind:this={dialogRef}
  class="fixed inset-0 z-50 flex items-center justify-center {isMobile ? 'p-0' : 'p-4'}"
  role="dialog"
  aria-modal="true"
  aria-labelledby="ai-transform-title"
  tabindex="-1"
  onkeydown={handleKeydown}
>
  <div
    class="bg-background border border-border shadow-lg flex flex-col {isMobile
      ? 'h-full w-full rounded-none'
      : 'rounded-lg w-full max-w-3xl max-h-[80vh]'}"
    onclick={(e) => e.stopPropagation()}
    role="presentation"
  >
    <!-- Header -->
    <div class="flex items-center justify-between p-4 border-b border-border flex-shrink-0">
      <h2 id="ai-transform-title" class="text-lg font-semibold flex items-center gap-2">
        <ActionIcon size={20} />
        {$_(`dialog.ai_transform.title.${action}`)}
      </h2>
      <button
        type="button"
        onclick={onClose}
        class="p-1 hover:bg-accent rounded-md"
        aria-label={$_('common.close')}
      >
        <X size={18} />
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-4">
      {#if loading}
        <div class="flex flex-col items-center justify-center py-12 gap-4">
          <Loader2 size={32} class="animate-spin text-primary" />
          <span class="text-muted-foreground">{$_('dialog.ai_transform.loading')}</span>
        </div>
      {:else if error}
        <div class="flex flex-col items-center justify-center py-12 gap-4">
          <AlertTriangle size={32} class="text-destructive" />
          <span class="text-destructive text-center">{error}</span>
        </div>
      {:else if noChanges}
        <div class="flex flex-col items-center justify-center py-12 gap-4">
          <Check size={32} class="text-green-500" />
          <span class="text-muted-foreground">{$_('dialog.ai_transform.no_changes')}</span>
        </div>
      {:else if diffResult}
        <!-- Diff view -->
        <div class="font-mono text-sm bg-muted/30 p-4 rounded-md overflow-x-auto">
          {#each diffResult as change, i (i)}
            {#if change.added}
              <div class="bg-green-500/20 border-l-4 border-green-500 pl-2 -ml-2">
                {#each change.value.split('\n').slice(0, -1) as line, li (li)}
                  <div class="min-h-[1.5em]">+ {line}</div>
                {/each}
                {#if !change.value.endsWith('\n')}
                  <div class="min-h-[1.5em]">+ {change.value.split('\n').slice(-1)[0]}</div>
                {/if}
              </div>
            {:else if change.removed}
              <div class="bg-red-500/20 border-l-4 border-red-500 pl-2 -ml-2">
                {#each change.value.split('\n').slice(0, -1) as line, li (li)}
                  <div class="min-h-[1.5em]">- {line}</div>
                {/each}
                {#if !change.value.endsWith('\n')}
                  <div class="min-h-[1.5em]">- {change.value.split('\n').slice(-1)[0]}</div>
                {/if}
              </div>
            {:else}
              {#each change.value.split('\n').slice(0, -1) as line, li (li)}
                <div class="min-h-[1.5em] text-muted-foreground">{line}</div>
              {/each}
              {#if !change.value.endsWith('\n')}
                <div class="min-h-[1.5em] text-muted-foreground">
                  {change.value.split('\n').slice(-1)[0]}
                </div>
              {/if}
            {/if}
          {/each}
        </div>
      {/if}
    </div>

    <!-- Footer -->
    <div class="flex items-center justify-between p-4 border-t border-border flex-shrink-0 gap-4">
      <div class="text-xs text-muted-foreground">
        {#if !loading && !error && !noChanges}
          {isFullContent
            ? $_('dialog.ai_transform.full_note')
            : $_('dialog.ai_transform.selection_only')}
        {/if}
      </div>
      <div class="flex items-center gap-2">
        <button
          type="button"
          onclick={onClose}
          class="px-4 py-2 text-sm hover:bg-accent rounded-md"
        >
          {$_('common.cancel')}
        </button>
        <button
          type="button"
          onclick={handleApply}
          disabled={loading || !!error || noChanges}
          class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md disabled:opacity-50 disabled:cursor-not-allowed"
        >
          {$_('dialog.ai_transform.apply')}
        </button>
      </div>
    </div>
  </div>
</div>

<!-- Conflict warning dialog -->
{#if conflictDetected}
  <div
    class="fixed inset-0 bg-black/50 z-[60] flex items-center justify-center p-4"
    onclick={() => (conflictDetected = false)}
    onkeydown={(e) => e.key === 'Escape' && (conflictDetected = false)}
    tabindex="-1"
    role="presentation"
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_interactive_supports_focus -->
    <div
      class="bg-background border border-border rounded-lg shadow-lg p-6 max-w-md w-full"
      onclick={(e) => e.stopPropagation()}
      role="alertdialog"
      aria-labelledby="conflict-title"
    >
      <div class="flex items-start gap-4">
        <AlertTriangle size={24} class="text-amber-500 flex-shrink-0 mt-0.5" />
        <div class="flex-1">
          <h3 id="conflict-title" class="font-semibold mb-2">
            {$_('dialog.ai_transform.conflict_title')}
          </h3>
          <p class="text-sm text-muted-foreground mb-4">
            {$_('dialog.ai_transform.conflict_warning')}
          </p>
          <div class="flex justify-end gap-2">
            <button
              type="button"
              onclick={() => (conflictDetected = false)}
              class="px-3 py-1.5 text-sm hover:bg-accent rounded-md"
            >
              {$_('common.cancel')}
            </button>
            <button
              type="button"
              onclick={handleApplyAnyway}
              class="px-3 py-1.5 text-sm bg-amber-500 text-white hover:bg-amber-600 rounded-md"
            >
              {$_('dialog.ai_transform.apply_anyway')}
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}
