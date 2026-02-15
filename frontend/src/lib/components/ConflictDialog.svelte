<script lang="ts">
  import { type DiffLine, diffLines } from '$lib/offline/diff-utils';
  import { getConflicts, resolveConflict } from '$lib/offline/sync-manager.svelte';

  import BaseDialog from './ui/BaseDialog.svelte';

  const conflicts = $derived(getConflicts());
  const isOpen = $derived(conflicts.length > 0);

  let isResolving = $state(false);
  let resolvedCount = $state(0);
  let expandedIndex = $state(0);

  // Reset expanded index when conflicts change
  $effect(() => {
    if (conflicts.length > 0 && expandedIndex >= conflicts.length) {
      expandedIndex = 0;
    }
  });

  const current = $derived(conflicts[expandedIndex] || null);
  const diff = $derived(current ? diffLines(current.remoteContent, current.localContent) : []);

  async function handleResolution(
    operationId: string,
    resolution: 'keep_local' | 'keep_remote' | 'keep_both'
  ) {
    if (isResolving) return;
    isResolving = true;
    try {
      await resolveConflict(operationId, resolution);
    } finally {
      isResolving = false;
    }
  }

  async function handleBulkResolution(resolution: 'keep_local' | 'keep_remote') {
    if (isResolving) return;
    isResolving = true;
    resolvedCount = 0;
    const total = conflicts.length;
    try {
      // Resolve all conflicts sequentially (each resolveConflict mutates the conflicts array)
      for (let i = 0; i < total; i++) {
        const conflict = getConflicts()[0];
        if (!conflict) break;
        await resolveConflict(conflict.operationId, resolution);
        resolvedCount = i + 1;
      }
    } finally {
      isResolving = false;
      resolvedCount = 0;
    }
  }

  function lineClass(type: DiffLine['type']): string {
    switch (type) {
      case 'added':
        return 'bg-green-500/20 text-green-300';
      case 'removed':
        return 'bg-red-500/20 text-red-300';
      default:
        return 'text-muted-foreground';
    }
  }

  function linePrefix(type: DiffLine['type']): string {
    switch (type) {
      case 'added':
        return '+';
      case 'removed':
        return '-';
      default:
        return ' ';
    }
  }
</script>

<BaseDialog
  open={isOpen}
  title={conflicts.length > 1
    ? `Synchronisationskonflikte (${conflicts.length})`
    : current?.isDelete
      ? 'Loeschkonflikt'
      : 'Synchronisationskonflikt'}
  onClose={() => {
    if (current) handleResolution(current.operationId, 'keep_remote');
  }}
  size="xl"
  closeOnBackdrop={false}
  closeOnEscape={false}
  scrollable={true}
>
  {#snippet content()}
    {#if conflicts.length > 0}
      <div class="space-y-4">
        <!-- Bulk resolution progress -->
        {#if isResolving && resolvedCount > 0}
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <div
              class="animate-spin w-4 h-4 border-2 border-primary border-t-transparent rounded-full"
            ></div>
            <span
              >Konflikte werden aufgeloest... ({resolvedCount}/{conflicts.length +
                resolvedCount})</span
            >
          </div>
        {/if}

        <!-- Bulk actions (when multiple conflicts) -->
        {#if conflicts.length > 1 && !isResolving}
          <div class="flex gap-2 p-3 rounded-md bg-muted/50 border border-border">
            <span class="text-sm text-muted-foreground self-center mr-auto">
              {conflicts.length} Konflikte
            </span>
            <button
              class="px-3 py-1.5 rounded-md text-sm bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
              onclick={() => handleBulkResolution('keep_local')}
            >
              Alle: Lokal behalten
            </button>
            <button
              class="px-3 py-1.5 rounded-md text-sm bg-secondary text-secondary-foreground hover:bg-secondary/80 transition-colors"
              onclick={() => handleBulkResolution('keep_remote')}
            >
              Alle: Server behalten
            </button>
          </div>
        {/if}

        <!-- Conflict list (tabs for multiple) -->
        {#if conflicts.length > 1}
          <div class="flex gap-1 overflow-x-auto pb-1">
            {#each conflicts as conflict, i (conflict.operationId)}
              <button
                class="px-3 py-1.5 rounded-md text-sm whitespace-nowrap transition-colors {expandedIndex ===
                i
                  ? 'bg-accent text-foreground'
                  : 'text-muted-foreground hover:bg-accent/50'}"
                onclick={() => (expandedIndex = i)}
              >
                {conflict.localTitle || conflict.remoteTitle || `Konflikt ${i + 1}`}
              </button>
            {/each}
          </div>
        {/if}

        <!-- Current conflict detail -->
        {#if current}
          {#if current.isDelete}
            <p class="text-foreground">
              Diese Notiz wurde lokal geloescht, aber auf dem Server bearbeitet.
            </p>
            <div class="rounded-md border border-border p-3">
              <h4 class="text-sm font-medium text-muted-foreground mb-2">Server-Version</h4>
              <p class="font-medium text-foreground">{current.remoteTitle}</p>
              <pre
                class="mt-2 text-sm text-muted-foreground whitespace-pre-wrap max-h-48 overflow-y-auto">{current.remoteContent.slice(
                  0,
                  500
                )}{current.remoteContent.length > 500 ? '...' : ''}</pre>
            </div>
          {:else}
            <p class="text-foreground text-sm">
              Die Notiz <strong>{current.localTitle || current.remoteTitle}</strong> wurde lokal und auf
              dem Server gleichzeitig bearbeitet.
            </p>

            <!-- Side-by-side comparison -->
            <div class="grid grid-cols-2 gap-3">
              <div>
                <h4 class="text-sm font-medium text-green-400 mb-1">Lokale Version</h4>
                <div
                  class="rounded-md border border-border bg-muted/50 p-2 max-h-64 overflow-y-auto"
                >
                  <pre class="text-xs whitespace-pre-wrap font-mono">{current.localContent.slice(
                      0,
                      1000
                    )}{current.localContent.length > 1000 ? '\n...' : ''}</pre>
                </div>
              </div>
              <div>
                <h4 class="text-sm font-medium text-blue-400 mb-1">Server-Version</h4>
                <div
                  class="rounded-md border border-border bg-muted/50 p-2 max-h-64 overflow-y-auto"
                >
                  <pre class="text-xs whitespace-pre-wrap font-mono">{current.remoteContent.slice(
                      0,
                      1000
                    )}{current.remoteContent.length > 1000 ? '\n...' : ''}</pre>
                </div>
              </div>
            </div>

            <!-- Diff view -->
            {#if diff.length > 0}
              <details class="text-sm">
                <summary class="cursor-pointer text-muted-foreground hover:text-foreground">
                  Diff anzeigen ({diff.filter((d) => d.type !== 'same').length} Aenderungen)
                </summary>
                <div
                  class="mt-2 rounded-md border border-border bg-muted/30 p-2 max-h-48 overflow-y-auto font-mono text-xs"
                >
                  {#each diff as line, i (i)}
                    <div class={lineClass(line.type)}>
                      <span class="select-none opacity-50 mr-1">{linePrefix(line.type)}</span
                      >{line.content}
                    </div>
                  {/each}
                </div>
              </details>
            {/if}
          {/if}
        {/if}
      </div>
    {/if}
  {/snippet}

  {#snippet footer()}
    {#if current}
      <div class="flex gap-2 justify-end w-full">
        {#if current.isDelete}
          <button
            class="px-4 py-2 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 transition-colors"
            onclick={() => handleResolution(current.operationId, 'keep_remote')}
            disabled={isResolving}
          >
            Server behalten
          </button>
          <button
            class="px-4 py-2 rounded-md bg-destructive text-destructive-foreground hover:bg-destructive/90 transition-colors"
            onclick={() => handleResolution(current.operationId, 'keep_local')}
            disabled={isResolving}
          >
            Trotzdem loeschen
          </button>
        {:else}
          <button
            class="px-4 py-2 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 transition-colors"
            onclick={() => handleResolution(current.operationId, 'keep_remote')}
            disabled={isResolving}
          >
            Server behalten
          </button>
          <button
            class="px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            onclick={() => handleResolution(current.operationId, 'keep_local')}
            disabled={isResolving}
          >
            Lokal behalten
          </button>
          <button
            class="px-4 py-2 rounded-md bg-accent text-accent-foreground hover:bg-accent/80 transition-colors"
            onclick={() => handleResolution(current.operationId, 'keep_both')}
            disabled={isResolving}
          >
            Beide behalten (Kopie)
          </button>
        {/if}
      </div>
    {/if}
  {/snippet}
</BaseDialog>
