<script lang="ts">
  import { diffLines } from 'diff';
  import { createFocusTrap, type FocusTrap } from 'focus-trap';
  import {
    ChevronLeft,
    ChevronRight,
    Eye,
    GitCompare,
    History,
    Loader2,
    Lock,
    RotateCcw,
    X,
  } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { NoteVersion } from '$lib/api';
  import * as api from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import type { EncryptedPayload } from '$lib/crypto/e2e';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as toast from '$lib/stores/toast.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { formatRelativeTime } from '$lib/utils/time';

  interface Props {
    noteId: string;
    noteTitle: string;
    currentVersion: number;
    currentContent: string;
    onClose: () => void;
    onRestored: () => void;
  }

  const { noteId, noteTitle, currentVersion, currentContent, onClose, onRestored }: Props =
    $props();

  // Extended version type that includes "current" pseudo-version
  type VersionItem =
    | NoteVersion
    | {
        id: 'current';
        version: number;
        title: string;
        content: string;
        snapshot_at: string;
        isCurrent: true;
      };

  let versions = $state<NoteVersion[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let encryptionLocked = $state(false);
  let total = $state(0);
  let nextCursor = $state<string | null>(null);
  let loadingMore = $state(false);

  /**
   * Decrypt a single version if it's encrypted.
   * Modifies content and title in place.
   */
  function decryptVersion(version: NoteVersion): void {
    if (!version.content_encrypted || !version.encrypted_content) {
      return; // Not encrypted, nothing to do
    }

    if (!encryption.isEncryptionUnlocked()) {
      encryptionLocked = true;
      return;
    }

    try {
      const encryptedPayload: EncryptedPayload = {
        ciphertext: version.encrypted_content,
        metadata: {
          version: (version.encryption_version as 2) || 2,
          algorithm: 'XChaCha20-Poly1305',
          kdf: 'Argon2id',
          kdf_strength: 'interactive',
          nonce_bytes: 24,
          wrapped_dek: version.wrapped_dek || '',
        },
      };

      const { title, content } = encryption.decryptNote(
        version.encrypted_title || null,
        encryptedPayload
      );

      version.content = content;
      if (title) {
        version.title = title;
      }
    } catch (e) {
      console.error('[VERSION] Failed to decrypt version:', e);
      version.content = $_('component.version_history.decrypt_failed');
    }
  }

  let selectedVersion = $state<VersionItem | null>(null);
  let compareVersion = $state<VersionItem | null>(null);
  let mode = $state<'preview' | 'compare'>('preview');
  let restoring = $state(false);
  let showRestoreConfirm = $state(false);

  // Mobile-specific state
  let mobileTab = $state<'versions' | 'content'>('versions');
  const isMobile = $derived(ui.getIsMobile());

  // Current state as a virtual version
  const currentAsVersion: VersionItem = $derived({
    id: 'current' as const,
    version: currentVersion,
    title: noteTitle,
    content: currentContent,
    snapshot_at: new Date().toISOString(),
    isCurrent: true as const,
  });

  // All items including current
  const allItems = $derived<VersionItem[]>([currentAsVersion, ...versions]);

  // Load versions on mount
  $effect(() => {
    loadVersions();
  });

  async function loadVersions() {
    loading = true;
    error = null;
    encryptionLocked = false;
    try {
      const response = await api.listVersions(noteId, { limit: 50 });
      versions = response.versions;
      total = response.total;
      nextCursor = response.next_cursor || null;

      // Decrypt all loaded versions
      for (const v of versions) {
        decryptVersion(v);
      }

      // Select current by default
      selectedVersion = currentAsVersion;

      // If there are historical versions, set compare to first one
      if (versions.length > 0) {
        compareVersion = versions[0];
      }
    } catch (e) {
      error = e instanceof Error ? e.message : $_('component.version_history.load_error');
    } finally {
      loading = false;
    }
  }

  async function loadMore() {
    if (!nextCursor || loadingMore) return;

    loadingMore = true;
    try {
      const response = await api.listVersions(noteId, { limit: 50, cursor: nextCursor });
      // Decrypt newly loaded versions
      for (const v of response.versions) {
        decryptVersion(v);
      }
      versions = [...versions, ...response.versions];
      nextCursor = response.next_cursor || null;
    } catch (e) {
      console.error('Failed to load more versions:', e);
      toast.error($_('component.version_history.load_error'));
    } finally {
      loadingMore = false;
    }
  }

  function selectVersion(v: VersionItem) {
    if (mode === 'compare' && selectedVersion) {
      // Don't compare with itself
      if (v.id === selectedVersion.id) return;
      compareVersion = v;
      // On mobile, switch to content tab after selecting second version
      if (isMobile) {
        mobileTab = 'content';
      }
    } else {
      selectedVersion = v;
      compareVersion = null;
      // On mobile in preview mode, switch to content tab
      if (isMobile && mode === 'preview') {
        mobileTab = 'content';
      }
    }
  }

  function toggleMode() {
    if (mode === 'preview') {
      mode = 'compare';
      // Set compare version to first historical version if available
      if (versions.length > 0 && selectedVersion?.id === 'current') {
        compareVersion = versions[0];
      } else if (selectedVersion && selectedVersion.id !== 'current') {
        // If a historical version is selected, compare with current
        compareVersion = currentAsVersion;
      }
    } else {
      mode = 'preview';
      compareVersion = null;
    }
  }

  async function handleRestore() {
    if (!selectedVersion || selectedVersion.id === 'current' || restoring) return;

    restoring = true;
    try {
      await api.restoreVersion(noteId, selectedVersion.version, currentVersion);
      onRestored();
      onClose();
    } catch (e) {
      error = e instanceof Error ? e.message : $_('component.version_history.restore_error');
      showRestoreConfirm = false;
    } finally {
      restoring = false;
    }
  }

  function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleDateString(undefined, {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  function isCurrent(v: VersionItem): v is {
    id: 'current';
    version: number;
    title: string;
    content: string;
    snapshot_at: string;
    isCurrent: true;
  } {
    return 'isCurrent' in v && v.isCurrent === true;
  }

  // Compute diff for compare mode
  const diffResult = $derived.by(() => {
    if (mode !== 'compare' || !selectedVersion || !compareVersion) {
      return null;
    }

    // Don't compare with itself
    if (selectedVersion.id === compareVersion.id) {
      return null;
    }

    // Order by version: older first, newer second
    const [older, newer] =
      selectedVersion.version < compareVersion.version
        ? [selectedVersion, compareVersion]
        : [compareVersion, selectedVersion];

    const changes = diffLines(older.content, newer.content);
    return { changes, older, newer };
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      if (showRestoreConfirm) {
        showRestoreConfirm = false;
      } else {
        onClose();
      }
    }
  }

  // Navigate versions with arrow keys
  function navigateVersion(direction: 'prev' | 'next') {
    if (!selectedVersion || allItems.length === 0) return;

    const currentIdx = allItems.findIndex((v) => v.id === selectedVersion!.id);
    if (currentIdx === -1) return;

    if (direction === 'prev' && currentIdx > 0) {
      selectedVersion = allItems[currentIdx - 1];
    } else if (direction === 'next' && currentIdx < allItems.length - 1) {
      selectedVersion = allItems[currentIdx + 1];
    }
  }

  function getVersionLabel(v: VersionItem): string {
    if (isCurrent(v)) {
      return $_('component.version_history.current');
    }
    return `v${v.version}`;
  }

  // Focus trap and focus restoration
  let dialogRef: HTMLDivElement | null = $state(null);
  let focusTrap: FocusTrap | null = null;
  let previousActiveElement: Element | null = null;

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
  aria-labelledby="version-history-title"
  tabindex="-1"
  onkeydown={handleKeydown}
>
  <div
    class="bg-background border border-border shadow-lg flex flex-col {isMobile
      ? 'h-full w-full rounded-none'
      : 'ui-panel rounded-2xl w-full max-w-5xl h-[80vh]'}"
    onclick={(e) => e.stopPropagation()}
    role="presentation"
  >
    <!-- Header -->
    <div class="ui-page-header flex items-center justify-between p-4 flex-shrink-0">
      <div class="ui-page-title-group min-w-0">
        <div class="ui-panel-soft flex h-9 w-9 items-center justify-center rounded-full">
          <History size={18} class="flex-shrink-0 text-primary" />
        </div>
        <div class="ui-page-title-stack">
          <h2 id="version-history-title" class="ui-page-title truncate">
            {isMobile
              ? $_('component.version_history.versions_tab')
              : `${$_('component.version_history.title')}: ${noteTitle}`}
          </h2>
          {#if !isMobile}
            <div class="ui-page-subtitle">
              {$_('component.version_history.preview')} / {$_('component.version_history.compare')}
            </div>
          {/if}
        </div>
      </div>
      <div class="flex items-center gap-2 flex-shrink-0">
        <!-- Mode toggle -->
        <button
          type="button"
          onclick={toggleMode}
          class="ui-tab {mode === 'compare' ? 'is-active' : ''}"
          title={mode === 'preview'
            ? $_('component.version_history.compare')
            : $_('component.version_history.preview')}
        >
          {#if mode === 'preview'}
            <GitCompare size={16} />
            {#if !isMobile}{$_('component.version_history.compare')}{/if}
          {:else}
            <Eye size={16} />
            {#if !isMobile}{$_('component.version_history.preview')}{/if}
          {/if}
        </button>
        <button
          type="button"
          onclick={onClose}
          class="ui-icon-button ui-icon-button-sm"
          aria-label={$_('common.close')}
        >
          <X size={18} />
        </button>
      </div>
    </div>

    <!-- Mobile Tab Navigation -->
    {#if isMobile}
      <div class="flex border-b border-border/80 flex-shrink-0 p-2">
        <button
          onclick={() => (mobileTab = 'versions')}
          class="ui-tab flex-1 justify-center {mobileTab === 'versions' ? 'is-active' : ''}"
        >
          {$_('component.version_history.versions_tab')}
        </button>
        <button
          onclick={() => (mobileTab = 'content')}
          class="ui-tab flex-1 justify-center {mobileTab === 'content' ? 'is-active' : ''}"
        >
          {mode === 'compare'
            ? $_('component.version_history.compare')
            : $_('component.version_history.preview')}
        </button>
      </div>
    {/if}

    <!-- Content -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Sidebar: Version list -->
      <div
        class="{isMobile
          ? 'w-full'
          : 'w-64'} border-r border-border flex flex-col flex-shrink-0 {isMobile &&
        mobileTab !== 'versions'
          ? 'hidden'
          : ''}"
      >
        <div class="ui-panel-section text-sm text-muted-foreground border-b border-border/70">
          {total + 1}
          {total !== 0
            ? $_('component.version_history.versions_tab')
            : $_('component.version_history.version_singular')} ({$_(
            'component.version_history.current'
          )})
        </div>

        <!-- Compare mode hint on mobile -->
        {#if isMobile && mode === 'compare'}
          <div class="ui-panel-section text-xs bg-primary/10 border-b border-border/70">
            {#if selectedVersion && !compareVersion}
              <span class="font-medium">{getVersionLabel(selectedVersion)}</span>
            {:else if selectedVersion && compareVersion}
              <span class="font-medium">{getVersionLabel(selectedVersion)}</span> ↔
              <span class="font-medium">{getVersionLabel(compareVersion)}</span>
            {/if}
          </div>
        {/if}

        {#if loading}
          <div class="ui-empty-state flex-1">
            <Loader2 size={24} class="animate-spin text-muted-foreground" />
          </div>
        {:else if error}
          <div class="ui-empty-state flex-1 p-4">
            <div class="ui-alert ui-alert-danger w-full max-w-sm">{error}</div>
          </div>
        {:else if encryptionLocked}
          <div class="ui-empty-state flex-1 gap-3 text-sm">
            <Lock size={32} class="text-amber-500" />
            <div class="text-center">
              <p class="font-medium">{$_('component.version_history.encryption_locked')}</p>
              <p class="text-xs mt-1">{$_('component.version_history.encryption_locked_hint')}</p>
            </div>
          </div>
        {:else}
          <div class="flex-1 overflow-y-auto">
            <!-- Current version -->
            <button
              onclick={() => selectVersion(currentAsVersion)}
              class="w-full p-3 text-left hover:bg-accent border-b border-border/50 transition-colors {selectedVersion?.id ===
              'current'
                ? 'bg-accent'
                : ''} {mode === 'compare' && compareVersion?.id === 'current'
                ? 'bg-primary/10'
                : ''}"
            >
              <div class="text-sm font-medium truncate flex items-center gap-2">
                <span class="inline-block w-2 h-2 rounded-full bg-green-500"></span>
                <span class="truncate flex-1">{noteTitle}</span>
                {#if mode === 'compare' && selectedVersion?.id === 'current'}
                  <span
                    class="flex-shrink-0 text-xs bg-primary text-primary-foreground px-1.5 py-0.5 rounded"
                    >A</span
                  >
                {:else if mode === 'compare' && compareVersion?.id === 'current'}
                  <span
                    class="flex-shrink-0 text-xs bg-secondary text-secondary-foreground px-1.5 py-0.5 rounded"
                    >B</span
                  >
                {/if}
              </div>
              <div class="text-xs text-muted-foreground mt-1">
                {$_('component.version_history.current')} (v{currentVersion})
              </div>
            </button>

            <!-- Historical versions -->
            {#each versions as version (version.id)}
              <button
                onclick={() => selectVersion(version)}
                class="w-full p-3 text-left hover:bg-accent border-b border-border/50 transition-colors {selectedVersion?.id ===
                version.id
                  ? 'bg-accent'
                  : ''} {mode === 'compare' && compareVersion?.id === version.id
                  ? 'bg-primary/10'
                  : ''}"
              >
                <div class="text-sm font-medium truncate flex items-center gap-2">
                  <span class="truncate flex-1">{version.title}</span>
                  {#if mode === 'compare' && selectedVersion?.id === version.id}
                    <span
                      class="flex-shrink-0 text-xs bg-primary text-primary-foreground px-1.5 py-0.5 rounded"
                      >A</span
                    >
                  {:else if mode === 'compare' && compareVersion?.id === version.id}
                    <span
                      class="flex-shrink-0 text-xs bg-secondary text-secondary-foreground px-1.5 py-0.5 rounded"
                      >B</span
                    >
                  {/if}
                </div>
                <div class="text-xs text-muted-foreground mt-1">
                  v{version.version} · {formatRelativeTime(version.snapshot_at, $_)}
                </div>
              </button>
            {/each}

            {#if versions.length === 0}
              <div class="p-4 text-sm text-muted-foreground text-center">
                {$_('component.version_history.no_versions')}
                <br />
                <span class="text-xs">{$_('component.version_history.no_versions_hint')}</span>
              </div>
            {/if}

            {#if nextCursor}
              <button
                onclick={loadMore}
                disabled={loadingMore}
                class="ui-button ui-button-ghost m-2 w-[calc(100%-1rem)] justify-center"
              >
                {#if loadingMore}
                  <Loader2 size={16} class="animate-spin inline mr-2" />
                {/if}
                {$_('component.version_history.load_more')}
              </button>
            {/if}
          </div>
        {/if}
      </div>

      <!-- Main content: Preview or Diff -->
      <div
        class="flex-1 flex flex-col overflow-hidden {isMobile && mobileTab !== 'content'
          ? 'hidden'
          : ''}"
      >
        {#if selectedVersion}
          <!-- Navigation and actions bar -->
          <div class="flex items-center justify-between p-3 border-b border-border flex-shrink-0">
            <div class="flex items-center gap-2">
              <button
                onclick={() => navigateVersion('prev')}
                disabled={allItems.findIndex((v) => v.id === selectedVersion?.id) === 0}
                class="ui-icon-button ui-icon-button-sm disabled:opacity-30 disabled:cursor-not-allowed"
                title="Neuere Version"
              >
                <ChevronLeft size={16} />
              </button>
              <span class="text-sm">
                {getVersionLabel(selectedVersion)}
                {#if mode === 'compare' && compareVersion}
                  ↔ {getVersionLabel(compareVersion)}
                {/if}
              </span>
              <button
                onclick={() => navigateVersion('next')}
                disabled={allItems.findIndex((v) => v.id === selectedVersion?.id) ===
                  allItems.length - 1}
                class="ui-icon-button ui-icon-button-sm disabled:opacity-30 disabled:cursor-not-allowed"
                title="Ältere Version"
              >
                <ChevronRight size={16} />
              </button>
            </div>

            <div class="flex items-center gap-2">
              {#if !isCurrent(selectedVersion)}
                {#if !isMobile}
                  <span class="text-xs text-muted-foreground">
                    {formatDate(selectedVersion.snapshot_at)}
                  </span>
                {/if}
                <button
                  onclick={() => (showRestoreConfirm = true)}
                  class="ui-button ui-button-primary"
                >
                  <RotateCcw size={14} />
                  {$_('component.version_history.restore')}
                </button>
              {:else}
                <span class="text-xs text-green-600 font-medium">
                  {$_('component.version_history.current')}
                </span>
              {/if}
            </div>
          </div>

          <!-- Content area -->
          <div class="flex-1 overflow-auto p-4">
            {#if mode === 'preview'}
              <!-- Preview mode: show raw content -->
              <div class="space-y-4">
                <div class="text-lg font-semibold">{selectedVersion.title}</div>
                <pre
                  class="ui-panel-soft whitespace-pre-wrap overflow-x-auto p-4 font-mono text-sm">{selectedVersion.content}</pre>
              </div>
            {:else if mode === 'compare' && diffResult}
              <!-- Compare mode: show diff -->
              <div class="space-y-4">
                <div class="text-sm text-muted-foreground">
                  {$_('component.version_history.compare')}: {getVersionLabel(diffResult.older)} → {getVersionLabel(
                    diffResult.newer
                  )}
                </div>
                <div class="ui-panel-soft overflow-x-auto p-4 font-mono text-sm">
                  {#each diffResult.changes as change, i (i)}
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
              </div>
            {:else if mode === 'compare' && !compareVersion}
              <!-- Compare mode without second version selected -->
              <div class="flex items-center justify-center h-full text-muted-foreground">
                {$_('component.version_history.compare')}
              </div>
            {:else if mode === 'compare' && selectedVersion?.id === compareVersion?.id}
              <!-- Same version selected twice -->
              <div class="flex items-center justify-center h-full text-muted-foreground">
                {$_('component.version_history.compare')}
              </div>
            {/if}
          </div>
        {:else}
          <div class="flex-1 flex items-center justify-center text-muted-foreground">
            {$_('component.version_history.preview')}
          </div>
        {/if}
      </div>
    </div>
  </div>
</div>

<!-- Restore confirmation dialog (uses BaseDialog) -->
{#if showRestoreConfirm && selectedVersion && !isCurrent(selectedVersion)}
  <BaseDialog
    open={true}
    title={$_('component.version_history.restore_confirm_title')}
    onClose={() => (showRestoreConfirm = false)}
    size="sm"
  >
    {#snippet content()}
      <p class="text-sm text-muted-foreground">
        {$_('component.version_history.restore_confirm_message', {
          values: {
            version: selectedVersion!.version,
            date: formatDate(selectedVersion!.snapshot_at),
          },
        })}
      </p>
    {/snippet}
    {#snippet footer()}
      <DialogActions>
        <button
          type="button"
          onclick={() => (showRestoreConfirm = false)}
          class="ui-button ui-button-secondary"
        >
          {$_('common.cancel')}
        </button>
        <button
          type="button"
          onclick={handleRestore}
          disabled={restoring}
          class="ui-button ui-button-primary"
        >
          {#if restoring}
            <Loader2 size={16} class="animate-spin inline mr-2" />
          {/if}
          {$_('component.version_history.restore')}
        </button>
      </DialogActions>
    {/snippet}
  </BaseDialog>
{/if}
