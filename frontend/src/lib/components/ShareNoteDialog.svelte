<script lang="ts">
  import { _ } from 'svelte-i18n';
  import { UserPlus, Trash2, Loader2 } from 'lucide-svelte';
  import BaseDialog from './ui/BaseDialog.svelte';
  import * as api from '$lib/api';
  import type { NoteShare, UserSearchResult } from '$lib/api';
  import * as toast from '$lib/stores/toast.svelte';

  interface Props {
    noteId: string;
    isEncrypted?: boolean;
    onClose: () => void;
  }

  const { noteId, isEncrypted = false, onClose }: Props = $props();

  let searchQuery = $state('');
  let searchResults = $state<UserSearchResult[]>([]);
  let searching = $state(false);
  let shares = $state<NoteShare[]>([]);
  let loadingShares = $state(true);
  let selectedRole = $state<'viewer' | 'editor'>('viewer');
  let sharing = $state(false);
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;

  // Load existing shares on mount
  $effect(() => {
    loadShares();
  });

  async function loadShares() {
    try {
      loadingShares = true;
      shares = await api.getNoteShares(noteId);
    } catch (err) {
      console.error('Failed to load shares:', err);
    } finally {
      loadingShares = false;
    }
  }

  function handleSearchInput(e: Event) {
    const query = (e.target as HTMLInputElement).value;
    searchQuery = query;

    if (searchTimeout) clearTimeout(searchTimeout);

    if (query.length < 3) {
      searchResults = [];
      return;
    }

    searchTimeout = setTimeout(async () => {
      searching = true;
      try {
        searchResults = await api.searchUsers(query);
      } catch (err) {
        console.error('User search failed:', err);
        searchResults = [];
      } finally {
        searching = false;
      }
    }, 300);
  }

  async function handleShare(user: UserSearchResult) {
    sharing = true;
    try {
      await api.shareNote(noteId, user.username, selectedRole);
      toast.success($_('sharing.share_success'));
      searchQuery = '';
      searchResults = [];
      await loadShares();
    } catch (err: unknown) {
      const error = err as { message?: string };
      if (error.message?.includes('yourself')) {
        toast.error($_('sharing.cannot_share_self'));
      } else if (error.message?.includes('encrypted')) {
        toast.error($_('sharing.encrypted_warning'));
      } else {
        toast.error(error.message || $_('common.error'));
      }
    } finally {
      sharing = false;
    }
  }

  async function handleRemoveShare(userId: number) {
    try {
      await api.removeShare(noteId, userId);
      toast.success($_('sharing.unshare_success'));
      await loadShares();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : $_('common.error'));
    }
  }

  async function handleRoleChange(userId: number, newRole: string) {
    try {
      await api.updateShareRole(noteId, userId, newRole);
      toast.success($_('sharing.role_updated'));
      await loadShares();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : $_('common.error'));
    }
  }
</script>

<BaseDialog open={true} title={$_('sharing.title')} {onClose} size="md" scrollable>
  {#snippet content()}
    {#if isEncrypted}
      <div
        class="bg-amber-500/10 border border-amber-500/30 rounded-md p-3 mb-4 text-sm text-amber-600 dark:text-amber-400"
      >
        {$_('sharing.encrypted_warning')}
      </div>
    {:else}
      <!-- Search and add users -->
      <div class="space-y-3 mb-4">
        <div class="flex gap-2">
          <input
            type="text"
            value={searchQuery}
            oninput={handleSearchInput}
            placeholder={$_('sharing.search_placeholder')}
            class="flex-1 px-3 py-2 bg-background border border-border rounded-md text-sm focus:ring-1 focus:ring-ring focus:outline-none"
          />
          <select
            bind:value={selectedRole}
            class="px-3 py-2 bg-background border border-border rounded-md text-sm focus:ring-1 focus:ring-ring focus:outline-none"
          >
            <option value="viewer">{$_('sharing.role_viewer')}</option>
            <option value="editor">{$_('sharing.role_editor')}</option>
          </select>
        </div>

        <!-- Search results -->
        {#if searching}
          <div class="flex items-center gap-2 text-sm text-muted-foreground px-1">
            <Loader2 size={14} class="animate-spin" />
            <span>{$_('common.loading')}</span>
          </div>
        {:else if searchResults.length > 0}
          <div
            class="border border-border rounded-md divide-y divide-border max-h-40 overflow-y-auto"
          >
            {#each searchResults as user (user.id)}
              <button
                onclick={() => handleShare(user)}
                disabled={sharing}
                class="w-full flex items-center justify-between px-3 py-2 text-sm hover:bg-accent transition-colors text-left"
              >
                <span>{user.username}</span>
                <UserPlus size={14} class="text-muted-foreground" />
              </button>
            {/each}
          </div>
        {/if}
      </div>

      <!-- Existing shares -->
      <div>
        <h3 class="text-sm font-medium mb-2 text-muted-foreground">
          {$_('sharing.shared_with_me')}
        </h3>
        {#if loadingShares}
          <div class="flex items-center gap-2 text-sm text-muted-foreground">
            <Loader2 size={14} class="animate-spin" />
            <span>{$_('common.loading')}</span>
          </div>
        {:else if shares.length === 0}
          <p class="text-sm text-muted-foreground">{$_('sharing.no_shares')}</p>
        {:else}
          <div class="space-y-2">
            {#each shares as share (share.id)}
              <div class="flex items-center justify-between p-2 bg-muted/30 rounded-md">
                <span class="text-sm font-medium">{share.shared_with_username}</span>
                <div class="flex items-center gap-2">
                  <select
                    value={share.role}
                    onchange={(e) =>
                      handleRoleChange(
                        share.shared_with_user_id,
                        (e.target as HTMLSelectElement).value
                      )}
                    class="px-2 py-1 bg-background border border-border rounded text-xs focus:ring-1 focus:ring-ring focus:outline-none"
                  >
                    <option value="viewer">{$_('sharing.role_viewer')}</option>
                    <option value="editor">{$_('sharing.role_editor')}</option>
                  </select>
                  <button
                    onclick={() => handleRemoveShare(share.shared_with_user_id)}
                    class="p-1 text-destructive hover:bg-destructive/10 rounded transition-colors"
                    title={$_('sharing.remove')}
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/snippet}
</BaseDialog>
