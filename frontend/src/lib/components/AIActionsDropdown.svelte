<script lang="ts">
  import {
    ArrowLeft,
    Briefcase,
    Coffee,
    Expand,
    FileText,
    Languages,
    Pencil,
    Send,
    Wand2,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { AIAction } from '$lib/api';

  interface Props {
    onSelectAction: (action: AIAction, customPrompt?: string) => void;
    onClose: () => void;
    triggerRect?: {
      top: number;
      right: number;
      bottom: number;
      left: number;
      width: number;
      height: number;
    } | null;
  }

  const { onSelectAction, onClose, triggerRect = null }: Props = $props();

  // Compute desktop position from trigger button rect
  const desktopStyle = $derived.by(() => {
    if (!triggerRect) return '';
    const top = triggerRect.bottom + 4; // 4px gap below button
    const right = window.innerWidth - triggerRect.right;
    return `top: ${top}px; right: ${right}px;`;
  });

  // State for custom prompt input mode
  let showCustomInput = $state(false);
  let customPromptText = $state('');
  let customInputRef: HTMLInputElement | null = $state(null);

  // Action definitions with icons and i18n keys
  const actionGroups = [
    {
      id: 'structure',
      items: [
        { action: 'format' as AIAction, icon: Wand2 },
        { action: 'summarize' as AIAction, icon: FileText },
        { action: 'expand' as AIAction, icon: Expand },
      ],
    },
    {
      id: 'translate',
      items: [
        { action: 'translate_de' as AIAction, icon: Languages },
        { action: 'translate_en' as AIAction, icon: Languages },
      ],
    },
    {
      id: 'tone',
      items: [
        { action: 'formal' as AIAction, icon: Briefcase },
        { action: 'informal' as AIAction, icon: Coffee },
      ],
    },
  ];

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (showCustomInput) {
        if (customPromptText.trim()) {
          // Clear input first
          customPromptText = '';
        } else {
          // Go back to action list
          showCustomInput = false;
        }
        e.stopPropagation();
      } else {
        onClose();
      }
    }
  }

  function handleBackdropClick() {
    onClose();
  }

  function handleDropdownClick(e: MouseEvent) {
    e.stopPropagation();
  }

  function selectAction(action: AIAction) {
    onSelectAction(action);
  }

  function openCustomInput() {
    showCustomInput = true;
    // Focus input after render
    requestAnimationFrame(() => {
      customInputRef?.focus();
    });
  }

  function handleCustomSubmit() {
    if (customPromptText.trim()) {
      onSelectAction('custom', customPromptText.trim());
    }
  }

  function handleCustomInputKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && customPromptText.trim()) {
      e.preventDefault();
      handleCustomSubmit();
    }
  }

  function goBackToActions() {
    showCustomInput = false;
    customPromptText = '';
  }
</script>

<!-- Backdrop -->
<div
  class="fixed inset-0 z-40 md:bg-transparent bg-black/50"
  onclick={handleBackdropClick}
  onkeydown={handleKeydown}
  tabindex="-1"
  role="presentation"
></div>

<!-- Dropdown: Bottom sheet on mobile, fixed-positioned near button on desktop -->
<div
  class="fixed z-50 bg-background border border-border shadow-lg flex flex-col
		md:w-72 md:rounded-lg md:max-h-[calc(var(--app-viewport-height,100dvh)-6rem)]
		bottom-0 left-0 right-0 max-h-[80vh] rounded-t-2xl animate-slide-up md:animate-none
		md:bottom-auto md:left-auto md:rounded-lg"
  style={desktopStyle}
  onkeydown={handleKeydown}
  onclick={handleDropdownClick}
  role="dialog"
  aria-label={$_('component.editor.ai_actions')}
  tabindex="-1"
>
  <!-- Mobile handle bar -->
  <div class="md:hidden flex justify-center pt-2 pb-1">
    <div class="w-12 h-1 bg-muted-foreground/30 rounded-full"></div>
  </div>

  <!-- Header -->
  <div class="flex items-center gap-2 px-4 py-3 border-b border-border">
    {#if showCustomInput}
      <button
        type="button"
        onclick={goBackToActions}
        class="p-1 hover:bg-accent rounded-md -ml-1"
        aria-label={$_('common.back')}
      >
        <ArrowLeft size={18} />
      </button>
    {/if}
    <Wand2 size={18} class="text-primary" />
    <span class="font-medium text-sm">
      {showCustomInput ? $_('ai_action.custom') : $_('component.editor.ai_actions')}
    </span>
  </div>

  <!-- Content -->
  <div class="flex-1 overflow-y-auto p-2">
    {#if showCustomInput}
      <!-- Custom prompt input -->
      <div class="p-2 space-y-3">
        <input
          bind:this={customInputRef}
          bind:value={customPromptText}
          type="text"
          placeholder={$_('ai_action.custom_placeholder')}
          class="w-full px-3 py-2.5 bg-background border border-border rounded-md text-sm
						focus:outline-none focus:ring-2 focus:ring-ring"
          onkeydown={handleCustomInputKeydown}
        />
        <div class="flex gap-2">
          <button
            type="button"
            onclick={goBackToActions}
            class="flex-1 px-3 py-2 text-sm hover:bg-accent rounded-md"
          >
            {$_('common.cancel')}
          </button>
          <button
            type="button"
            onclick={handleCustomSubmit}
            disabled={!customPromptText.trim()}
            class="flex-1 px-3 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md
							disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
          >
            <Send size={14} />
            {$_('common.send')}
          </button>
        </div>
      </div>
    {:else}
      <!-- Action groups -->
      {#each actionGroups as group, groupIndex (group.id)}
        <div class="py-1">
          {#each group.items as item (item.action)}
            <button
              type="button"
              onclick={() => selectAction(item.action)}
              class="w-full flex items-start gap-3 px-3 py-2.5 hover:bg-accent rounded-md
								text-left transition-colors"
            >
              <item.icon size={18} class="text-muted-foreground mt-0.5 shrink-0" />
              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium">{$_(`ai_action.${item.action}`)}</div>
                <div class="text-xs text-muted-foreground">
                  {$_(`ai_action.${item.action}_desc`)}
                </div>
              </div>
            </button>
          {/each}
        </div>
        {#if groupIndex < actionGroups.length - 1}
          <div class="border-t border-border my-1"></div>
        {/if}
      {/each}

      <!-- Separator before custom -->
      <div class="border-t border-border my-1"></div>

      <!-- Custom prompt option -->
      <div class="py-1">
        <button
          type="button"
          onclick={openCustomInput}
          class="w-full flex items-start gap-3 px-3 py-2.5 hover:bg-accent rounded-md
						text-left transition-colors"
        >
          <Pencil size={18} class="text-muted-foreground mt-0.5 shrink-0" />
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium">{$_('ai_action.custom')}</div>
            <div class="text-xs text-muted-foreground">{$_('ai_action.custom_desc')}</div>
          </div>
        </button>
      </div>
    {/if}
  </div>
</div>
