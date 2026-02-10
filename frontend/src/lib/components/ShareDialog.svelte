<script lang="ts">
  import { Loader2,Trash2, UserPlus } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import type { UserSearchResult } from '$lib/api';
  import * as api from '$lib/api';
  import * as toast from '$lib/stores/toast.svelte';

  import BaseDialog from './ui/BaseDialog.svelte';

  type ResourceType = 'note' | 'folder' | 'collection';

  interface ShareEntry {
    id: number;
    shared_with_user_id: number;
    shared_with_username: string;
    role: string;
  }

  interface Props {
    resourceType: ResourceType;
    resourceId: string | number;
    isEncrypted?: boolean;
    onClose: () => void;
  }

  const { resourceType, resourceId, isEncrypted = false, onClose }: Props = $props();

  // I18n key mapping per resource type
  const i18nKeys: Record<
    ResourceType,
    { title: string; shareSuccess: string; unshareSuccess: string; encryptedWarning: string }
  > = {
    note: {
      title: 'sharing.title',
      shareSuccess: 'sharing.share_success',
      unshareSuccess: 'sharing.unshare_success',
      encryptedWarning: 'sharing.encrypted_warning',
    },
    folder: {
      title: 'sharing.folder_title',
      shareSuccess: 'sharing.folder_share_success',
      unshareSuccess: 'sharing.folder_unshare_success',
      encryptedWarning: 'sharing.folder_encrypted_warning',
    },
    collection: {
      title: 'sharing.collection_title',
      shareSuccess: 'sharing.collection_share_success',
      unshareSuccess: 'sharing.collection_unshare_success',
      encryptedWarning: 'sharing.collection_encrypted_warning',
    },
  };

  const keys = i18nKeys[resourceType];

  // API dispatch per resource type
  async function apiGetShares(): Promise<ShareEntry[]> {
    switch (resourceType) {
      case 'note':
        return (await api.getNoteShares(resourceId as string)) as ShareEntry[];
      case 'folder':
        return (await api.getFolderShares(resourceId as number)) as ShareEntry[];
      case 'collection':
        return (await api.getCollectionShares(resourceId as number)) as ShareEntry[];
    }
  }

  async function apiShare(username: string, role: string): Promise<void> {
    switch (resourceType) {
      case 'note':
        await api.shareNote(resourceId as string, username, role);
        break;
      case 'folder':
        await api.shareFolder(resourceId as number, username, role);
        break;
      case 'collection':
        await api.shareCollection(resourceId as number, username, role);
        break;
    }
  }

  async function apiRemoveShare(userId: number): Promise<void> {
    switch (resourceType) {
      case 'note':
        await api.removeShare(resourceId as string, userId);
        break;
      case 'folder':
        await api.removeFolderShare(resourceId as number, userId);
        break;
      case 'collection':
        await api.removeCollectionShare(resourceId as number, userId);
        break;
    }
  }

  async function apiUpdateRole(userId: number, newRole: string): Promise<void> {
    switch (resourceType) {
      case 'note':
        await api.updateShareRole(resourceId as string, userId, newRole);
        break;
      case 'folder':
        await api.updateFolderShareRole(resourceId as number, userId, newRole);
        break;
      case 'collection':
        await api.updateCollectionShareRole(resourceId as number, userId, newRole);
        break;
    }
  }

  let searchQuery = $state('');
  let searchResults = $state<UserSearchResult[]>([]);
  let searching = $state(false);
  let shares = $state<ShareEntry[]>([]);
  let loadingShares = $state(true);
  let selectedRole = $state<'viewer' | 'editor'>('viewer');
  let sharing = $state(false);
  let searchTimeout: ReturnType<typeof setTimeout> | null = null;

  $effect(() => {
    loadShares();
  });

  async function loadShares() {
    try {
      loadingShares = true;
      shares = await apiGetShares();
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
      await apiShare(user.username, selectedRole);
      toast.success($_(keys.shareSuccess));
      searchQuery = '';
      searchResults = [];
      await loadShares();
    } catch (err: unknown) {
      const error = err as { message?: string };
      if (error.message?.includes('yourself')) {
        toast.error($_('sharing.cannot_share_self'));
      } else if (error.message?.includes('encrypted')) {
        toast.error($_(keys.encryptedWarning));
      } else if (error.message?.includes('already shared')) {
        toast.error($_('sharing.collection_already_shared'));
      } else {
        toast.error(error.message || $_('common.error'));
      }
    } finally {
      sharing = false;
    }
  }

  async function handleRemoveShare(userId: number) {
    try {
      await apiRemoveShare(userId);
      toast.success($_(keys.unshareSuccess));
      await loadShares();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : $_('common.error'));
    }
  }

  async function handleRoleChange(userId: number, newRole: string) {
    try {
      await apiUpdateRole(userId, newRole);
      toast.success($_('sharing.role_updated'));
      await loadShares();
    } catch (err: unknown) {
      toast.error(err instanceof Error ? err.message : $_('common.error'));
    }
  }
</script>

<BaseDialog open={true} title={$_(keys.title)} {onClose} size="md" scrollable>
  {#snippet content()}
    {#if isEncrypted}
      <div
        class="bg-amber-500/10 border border-amber-500/30 rounded-md p-3 mb-4 text-sm text-amber-600 dark:text-amber-400"
      >
        {$_(keys.encryptedWarning)}
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
