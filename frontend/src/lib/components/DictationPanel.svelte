<script lang="ts">
  import type { EditorView } from '@codemirror/view';
  import { Loader2, Mic, MicOff, Send, Sparkles, X } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import {
    type BrowserDictation,
    createBrowserDictation,
    createServerDictation,
    type DictationState,
    getBestAudioMime,
    isBrowserSpeechSupported,
    isMediaRecorderSupported,
    type ServerDictation,
  } from '$lib/editor/dictation';

  interface Props {
    editorView?: EditorView;
    noteId: string;
    aiEnabled: boolean;
    hasOpenAIKey: boolean;
    onInsert: (text: string, withAICleanup: boolean) => void;
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

  const {
    editorView: _editorView,
    noteId: _noteId,
    aiEnabled,
    hasOpenAIKey,
    onInsert,
    onClose,
    triggerRect = null,
  }: Props = $props();

  // ── State ──────────────────────────────────────────────────────────

  const browserSupported = isBrowserSpeechSupported();
  const serverSupported = isMediaRecorderSupported() && getBestAudioMime() !== null;

  // Dictation mode: browser (Web Speech API) or server (Whisper)
  let mode = $state<'browser' | 'server'>(
    (() => {
      const stored = localStorage.getItem('xelanote_dictation_mode');
      if (stored === 'server' && hasOpenAIKey && serverSupported) return 'server';
      if (browserSupported) return 'browser';
      if (hasOpenAIKey && serverSupported) return 'server';
      return 'browser';
    })()
  );

  let aiCleanup = $state(
    (() => {
      const stored = localStorage.getItem('xelanote_dictation_ai_cleanup');
      return stored === 'true';
    })()
  );

  let dictationState = $state<DictationState>('idle');
  let interimText = $state('');
  let finalText = $state('');
  let errorKey = $state<string | null>(null);
  let recordingSeconds = $state(0);

  let browserDictation: BrowserDictation | null = null;
  let serverDictation: ServerDictation | null = null;
  let timerInterval: ReturnType<typeof setInterval> | null = null;

  // ── Derived ────────────────────────────────────────────────────────

  const displayText = $derived(finalText + (interimText ? interimText : ''));
  const canInsert = $derived(finalText.trim().length > 0);
  const isActive = $derived(dictationState === 'listening' || dictationState === 'recording');
  const showAICleanupToggle = $derived(aiEnabled && hasOpenAIKey);
  const canSwitchToServer = $derived(hasOpenAIKey && serverSupported);

  // Desktop position from trigger rect
  const desktopStyle = $derived.by(() => {
    if (!triggerRect) return '';
    const top = triggerRect.bottom + 4;
    const right = window.innerWidth - triggerRect.right;
    return `top: ${top}px; right: ${right}px;`;
  });

  // ── Persistence ────────────────────────────────────────────────────

  $effect(() => {
    localStorage.setItem('xelanote_dictation_mode', mode);
  });

  $effect(() => {
    localStorage.setItem('xelanote_dictation_ai_cleanup', String(aiCleanup));
  });

  // ── Actions ────────────────────────────────────────────────────────

  function startDictation() {
    errorKey = null;
    interimText = '';

    if (mode === 'browser') {
      if (!browserSupported) {
        errorKey = 'component.editor.dictation.error_not_supported';
        return;
      }
      browserDictation = createBrowserDictation(
        (text, isFinal) => {
          if (isFinal) {
            finalText += text;
            interimText = '';
          } else {
            interimText = text;
          }
        },
        (state) => {
          dictationState = state;
        },
        (err) => {
          if (err === 'microphone_denied') {
            errorKey = 'component.editor.dictation.error_mic_denied';
          } else {
            errorKey = 'component.editor.dictation.error_generic';
          }
        }
      );
      browserDictation.start();
    } else {
      serverDictation = createServerDictation(
        (state) => {
          dictationState = state;
        },
        (err) => {
          if (err === 'microphone_denied') {
            errorKey = 'component.editor.dictation.error_mic_denied';
          } else if (err === 'mediarecorder_unsupported') {
            errorKey = 'component.editor.dictation.error_not_supported';
          } else {
            errorKey = 'component.editor.dictation.error_generic';
          }
        }
      );
      serverDictation.start();
      recordingSeconds = 0;
      timerInterval = setInterval(() => {
        recordingSeconds += 1;
      }, 1000);
    }
  }

  async function stopDictation() {
    if (mode === 'browser') {
      browserDictation?.stop();
      // Finalize any interim
      if (interimText) {
        finalText += interimText;
        interimText = '';
      }
    } else {
      if (!serverDictation) return;
      if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
      }
      dictationState = 'processing';
      const blob = await serverDictation.stop();
      if (blob.size === 0) {
        dictationState = 'idle';
        return;
      }
      try {
        const { transcribeAudio } = await import('$lib/api/ai');
        const result = await transcribeAudio(blob);
        finalText += result.text;
      } catch (_err) {
        errorKey = 'component.editor.dictation.error_transcribe';
      }
      dictationState = 'idle';
    }
  }

  function toggleDictation() {
    if (isActive) {
      stopDictation();
    } else {
      startDictation();
    }
  }

  function handleInsert() {
    if (!canInsert) return;
    const text = finalText.trim();
    const shouldCleanup = aiCleanup && showAICleanupToggle && text.length >= 10;
    onInsert(text, shouldCleanup);
  }

  function clearText() {
    finalText = '';
    interimText = '';
  }

  function formatTime(seconds: number): string {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `${m}:${s.toString().padStart(2, '0')}`;
  }

  // ── Keyboard ───────────────────────────────────────────────────────

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (isActive) {
        stopDictation();
      } else {
        onClose();
      }
    }
  }

  // ── Cleanup ────────────────────────────────────────────────────────

  onMount(() => {
    return () => {
      browserDictation?.destroy();
      serverDictation?.destroy();
      if (timerInterval) clearInterval(timerInterval);
    };
  });
</script>

<!-- Backdrop -->
<div
  class="fixed inset-0 z-40 md:bg-transparent bg-black/50"
  onclick={onClose}
  onkeydown={handleKeydown}
  tabindex="-1"
  role="presentation"
></div>

<!-- Panel -->
<div
  class="fixed z-50 bg-background border border-border shadow-lg flex flex-col
    md:w-80 md:rounded-lg md:max-h-[calc(var(--app-viewport-height,100dvh)-6rem)]
    bottom-0 left-0 right-0 max-h-[80vh] rounded-t-2xl animate-slide-up md:animate-none
    md:bottom-auto md:left-auto md:rounded-lg"
  style={desktopStyle}
  onkeydown={handleKeydown}
  onclick={(e) => e.stopPropagation()}
  role="dialog"
  aria-label={$_('component.editor.dictation.title')}
  tabindex="-1"
>
  <!-- Mobile handle -->
  <div class="md:hidden flex justify-center pt-2 pb-1">
    <div class="w-12 h-1 bg-muted-foreground/30 rounded-full"></div>
  </div>

  <!-- Header -->
  <div class="flex items-center gap-2 px-4 py-3 border-b border-border">
    <Mic size={18} class={isActive ? 'text-destructive' : 'text-primary'} />
    <span class="font-medium text-sm flex-1">
      {$_('component.editor.dictation.title')}
    </span>
    {#if dictationState === 'recording'}
      <span class="text-xs text-muted-foreground tabular-nums">
        {formatTime(recordingSeconds)}
      </span>
    {/if}
    <button
      type="button"
      onclick={onClose}
      class="p-1 hover:bg-accent rounded-md -mr-1"
      aria-label={$_('common.close')}
    >
      <X size={16} />
    </button>
  </div>

  <!-- Transcript area -->
  <div class="flex-1 overflow-y-auto px-4 py-3 min-h-[100px] max-h-[40vh]">
    {#if displayText}
      <p class="text-sm whitespace-pre-wrap leading-relaxed">
        {finalText}<span class="text-muted-foreground/60">{interimText}</span>
      </p>
    {:else if dictationState === 'processing'}
      <div class="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 size={16} class="animate-spin" />
        {$_('component.editor.dictation.processing')}
      </div>
    {:else if isActive}
      <p class="text-sm text-muted-foreground">
        {$_('component.editor.dictation.listening_hint')}
      </p>
    {:else}
      <p class="text-sm text-muted-foreground">
        {$_('component.editor.dictation.idle_hint')}
      </p>
    {/if}
    {#if errorKey}
      <p class="text-sm text-destructive mt-2">{$_(errorKey)}</p>
    {/if}
  </div>

  <!-- Controls -->
  <div class="px-4 py-3 border-t border-border space-y-3">
    <!-- Mode toggle (only if both modes available) -->
    {#if browserSupported && canSwitchToServer}
      <div class="flex items-center gap-2 text-xs">
        <button
          type="button"
          onclick={() => {
            mode = 'browser';
          }}
          class="px-2 py-1 rounded-md transition-colors"
          class:bg-primary={mode === 'browser'}
          class:text-primary-foreground={mode === 'browser'}
          class:text-muted-foreground={mode !== 'browser'}
          disabled={isActive}
        >
          {$_('component.editor.dictation.mode_browser')}
        </button>
        <button
          type="button"
          onclick={() => {
            mode = 'server';
          }}
          class="px-2 py-1 rounded-md transition-colors"
          class:bg-primary={mode === 'server'}
          class:text-primary-foreground={mode === 'server'}
          class:text-muted-foreground={mode !== 'server'}
          disabled={isActive}
        >
          {$_('component.editor.dictation.mode_server')}
        </button>
      </div>
    {/if}

    <!-- AI Cleanup toggle -->
    {#if showAICleanupToggle}
      <label class="flex items-center gap-2 cursor-pointer">
        <input type="checkbox" bind:checked={aiCleanup} class="rounded border-border" />
        <Sparkles size={14} class="text-primary" />
        <span class="text-xs">{$_('component.editor.dictation.ai_cleanup')}</span>
      </label>
    {/if}

    <!-- Action buttons -->
    <div class="flex items-center gap-2">
      <!-- Record / Stop button -->
      <button
        type="button"
        onclick={toggleDictation}
        disabled={dictationState === 'processing'}
        class="flex-shrink-0 p-3 rounded-full transition-colors
          {isActive
          ? 'bg-destructive text-destructive-foreground hover:bg-destructive/90'
          : 'bg-primary text-primary-foreground hover:bg-primary/90'}
          disabled:opacity-50"
        aria-label={isActive
          ? $_('component.editor.dictation.stop')
          : $_('component.editor.dictation.start')}
      >
        {#if dictationState === 'processing'}
          <Loader2 size={20} class="animate-spin" />
        {:else if isActive}
          <MicOff size={20} />
        {:else}
          <Mic size={20} />
        {/if}
        {#if isActive}
          <!-- pulsing dot -->
          <span class="absolute -top-0.5 -right-0.5 flex h-3 w-3">
            <span
              class="animate-ping absolute inline-flex h-full w-full rounded-full bg-destructive opacity-75"
            ></span>
            <span class="relative inline-flex rounded-full h-3 w-3 bg-destructive"></span>
          </span>
        {/if}
      </button>

      <!-- Clear -->
      {#if finalText}
        <button
          type="button"
          onclick={clearText}
          class="px-3 py-2 text-sm hover:bg-accent rounded-md text-muted-foreground"
        >
          {$_('component.editor.dictation.clear')}
        </button>
      {/if}

      <div class="flex-1"></div>

      <!-- Insert -->
      <button
        type="button"
        onclick={handleInsert}
        disabled={!canInsert || isActive}
        class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90
          rounded-md disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
      >
        <Send size={14} />
        {$_('component.editor.dictation.insert')}
      </button>
    </div>
  </div>
</div>

<style>
  .animate-slide-up {
    animation: slideUp 0.2s ease-out;
  }

  @keyframes slideUp {
    from {
      transform: translateY(100%);
    }
    to {
      transform: translateY(0);
    }
  }
</style>
