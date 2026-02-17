<script lang="ts">
  import { X } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { bottomsheet } from '$lib/actions/bottomsheet';

  const {
    currentStep,
    onSnooze,
    onDismiss,
    onStepChange,
  }: {
    currentStep: number;
    onSnooze: () => void;
    onDismiss: () => void;
    onStepChange: (step: number) => void;
  } = $props();

  let dialogEl = $state<HTMLDivElement | null>(null);

  // Focus trap: focus dialog on mount
  $effect(() => {
    if (dialogEl) {
      dialogEl.focus();
    }
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      e.preventDefault();
      onSnooze();
    }
  }

  function handleBackdropClick(e: MouseEvent) {
    if (e.target === e.currentTarget) {
      onSnooze();
    }
  }

  function handleNext() {
    if (currentStep < 3) {
      onStepChange(currentStep + 1);
    } else {
      onSnooze();
    }
  }

  const TOTAL_STEPS = 3;
</script>

<!-- Backdrop -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="fixed inset-0 z-50 bg-black/40 backdrop-blur-md motion-reduce:backdrop-blur-none motion-reduce:bg-black/50"
  onclick={handleBackdropClick}
  onkeydown={handleKeydown}
>
  <!-- Dialog -->
  <div
    bind:this={dialogEl}
    role="dialog"
    aria-modal="true"
    aria-labelledby="ios-coach-title"
    tabindex="-1"
    class="fixed bottom-0 left-0 right-0 z-50 bg-background border-t border-border rounded-t-2xl shadow-lg pb-safe animate-bottom-sheet focus:outline-none"
    onkeydown={handleKeydown}
    use:bottomsheet={{ onClose: onSnooze }}
  >
    <!-- Handle bar -->
    <div class="flex justify-center pt-3 pb-1">
      <div class="w-10 h-1 rounded-full bg-muted-foreground/30"></div>
    </div>

    <!-- Header -->
    <div class="flex items-center justify-between px-5 pb-2">
      <div>
        <h2 id="ios-coach-title" class="text-lg font-semibold">{$_('pwa.ios_install_title')}</h2>
        <p class="text-sm text-muted-foreground">{$_('pwa.ios_install_subtitle')}</p>
      </div>
      <button
        onclick={onSnooze}
        class="p-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
        aria-label={$_('common.close')}
        style="min-width: 44px; min-height: 44px; display: flex; align-items: center; justify-content: center;"
      >
        <X size={20} />
      </button>
    </div>

    <!-- Step indicator -->
    <div class="flex justify-center gap-2 pb-4" role="tablist" aria-label="Installation steps">
      {#each Array(TOTAL_STEPS) as _, i (i)}
        <button
          role="tab"
          aria-selected={currentStep === i + 1}
          aria-label="Step {i + 1}"
          class="ios-coach-dot {currentStep === i + 1 ? 'ios-coach-dot-active' : ''}"
          onclick={() => onStepChange(i + 1)}
          style="min-width: 44px; min-height: 44px; display: flex; align-items: center; justify-content: center;"
        >
          <span
            class="block w-2 h-2 rounded-full transition-all duration-200 {currentStep === i + 1
              ? 'bg-primary scale-125'
              : 'bg-muted-foreground/40'}"
          ></span>
        </button>
      {/each}
    </div>

    <!-- Step content -->
    <div class="px-5 min-h-[140px] flex flex-col items-center justify-center text-center">
      {#if currentStep === 1}
        <div class="animate-fade-in">
          <!-- Safari Share Icon -->
          <div class="mb-3 flex justify-center">
            <div class="bg-primary/10 p-4 rounded-2xl">
              <svg
                class="w-10 h-10 text-primary"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d="M12 3v12M8 7l4-4 4 4"></path>
                <rect x="4" y="14" width="16" height="7" rx="2"></rect>
              </svg>
            </div>
          </div>
          <h3 class="text-base font-semibold mb-1">{$_('pwa.ios_step1_title')}</h3>
          <p class="text-sm text-muted-foreground max-w-[280px]">
            {$_('pwa.ios_step1_desc', { values: { icon: '' } })}
            <svg
              class="inline-block w-5 h-5 align-text-bottom -mt-0.5"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.5"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <path d="M12 3v12M8 7l4-4 4 4"></path>
              <rect x="4" y="14" width="16" height="7" rx="2"></rect>
            </svg>
          </p>
        </div>
      {:else if currentStep === 2}
        <div class="animate-fade-in">
          <!-- Plus Icon -->
          <div class="mb-3 flex justify-center">
            <div class="bg-primary/10 p-4 rounded-2xl">
              <svg
                class="w-10 h-10 text-primary"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <rect x="3" y="3" width="18" height="18" rx="3"></rect>
                <path d="M12 8v8M8 12h8"></path>
              </svg>
            </div>
          </div>
          <h3 class="text-base font-semibold mb-1">{$_('pwa.ios_step2_title')}</h3>
          <p class="text-sm text-muted-foreground max-w-[280px]">{$_('pwa.ios_step2_desc')}</p>
        </div>
      {:else if currentStep === 3}
        <div class="animate-fade-in">
          <!-- App Icon -->
          <div class="mb-3 flex justify-center">
            <div class="bg-primary/10 p-4 rounded-2xl">
              <svg
                class="w-10 h-10 text-primary"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <rect x="3" y="3" width="18" height="18" rx="4"></rect>
                <path d="M8 12l3 3 5-6"></path>
              </svg>
            </div>
          </div>
          <h3 class="text-base font-semibold mb-1">{$_('pwa.ios_step3_title')}</h3>
          <p class="text-sm text-muted-foreground max-w-[280px]">{$_('pwa.ios_step3_desc')}</p>
        </div>
      {/if}
    </div>

    <!-- Actions -->
    <div class="px-5 pt-4 pb-2">
      <button
        onclick={handleNext}
        class="w-full py-3 bg-primary text-primary-foreground rounded-xl font-medium hover:bg-primary/90 transition-colors"
        style="min-height: 44px;"
      >
        {currentStep < 3 ? $_('pwa.ios_next') : $_('pwa.ios_got_it')}
      </button>
    </div>

    <!-- Footer: Snooze / Dismiss -->
    <div class="flex items-center justify-center gap-4 px-5 pt-1 pb-4">
      <button
        onclick={onSnooze}
        class="text-sm text-muted-foreground hover:text-foreground transition-colors py-2 px-3"
        style="min-height: 44px;"
      >
        {$_('pwa.ios_later')}
      </button>
      <span class="text-muted-foreground/30">|</span>
      <button
        onclick={onDismiss}
        class="text-sm text-muted-foreground hover:text-foreground transition-colors py-2 px-3"
        style="min-height: 44px;"
      >
        {$_('pwa.ios_never')}
      </button>
    </div>
  </div>
</div>
