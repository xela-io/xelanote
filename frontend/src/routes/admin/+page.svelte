<script lang="ts">
  import { onMount } from 'svelte';
  import { _, locale } from 'svelte-i18n';

  import type { ActivityLogsOptions, AdminUser, SystemSettings } from '$lib/api';
  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import * as admin from '$lib/stores/admin.svelte';
  import * as auth from '$lib/stores/auth.svelte';
  import * as toast from '$lib/stores/toast.svelte';

  // i18n prefix
  const a = 'page.admin';

  // Tab state
  type Tab = 'dashboard' | 'users' | 'activity' | 'settings';
  let activeTab = $state<Tab>('dashboard');

  // Activity pagination
  let activityPage = $state(1);
  const activityLimit = $state(20);
  let activityFilter = $state<string>('');

  // Settings form state
  let settingsForm = $state<Partial<SystemSettings>>({});
  let settingsSaving = $state(false);

  // Delete confirmation
  let deleteConfirmUser = $state<AdminUser | null>(null);

  onMount(() => {
    loadTabData('dashboard');
  });

  async function loadTabData(tab: Tab) {
    try {
      switch (tab) {
        case 'dashboard':
          await admin.loadDetailedStats();
          break;
        case 'users':
          await admin.loadUsers();
          break;
        case 'activity':
          await loadActivity();
          break;
        case 'settings':
          await admin.loadSettings();
          initSettingsForm();
          break;
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : $_(`${a}.load_failed`));
    }
  }

  function initSettingsForm() {
    const settings = admin.getSettings();
    if (settings) {
      settingsForm = { ...settings };
    }
  }

  async function loadActivity() {
    const options: ActivityLogsOptions = {
      limit: activityLimit,
      page: activityPage,
    };
    if (activityFilter) {
      options.action = activityFilter;
    }
    await admin.loadActivityLogs(options);
  }

  function switchTab(tab: Tab) {
    activeTab = tab;
    loadTabData(tab);
  }

  async function handleToggleAdmin(user: AdminUser) {
    const currentUser = auth.getCurrentUser();
    if (currentUser && user.id === currentUser.id) {
      toast.error($_(`${a}.users.cannot_change_own_admin`));
      return;
    }

    try {
      await admin.toggleUserAdmin(user.id, !user.is_admin);
      const key = user.is_admin ? `${a}.users.admin_removed` : `${a}.users.admin_granted`;
      toast.success($_(key, { values: { username: user.username } }));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : $_(`${a}.users.admin_toggle_failed`));
    }
  }

  function confirmDeleteUser(user: AdminUser) {
    const currentUser = auth.getCurrentUser();
    if (currentUser && user.id === currentUser.id) {
      toast.error($_(`${a}.users.cannot_delete_self`));
      return;
    }
    deleteConfirmUser = user;
  }

  async function handleDeleteUser() {
    if (!deleteConfirmUser) return;

    try {
      await admin.deleteUser(deleteConfirmUser.id);
      toast.success(
        $_(`${a}.users.user_deleted`, { values: { username: deleteConfirmUser.username } })
      );
      deleteConfirmUser = null;
    } catch (err) {
      toast.error(err instanceof Error ? err.message : $_(`${a}.users.delete_failed`));
    }
  }

  async function handleSaveSettings() {
    settingsSaving = true;
    try {
      await admin.updateSettings(settingsForm);
      toast.success($_(`${a}.settings.saved`));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : $_(`${a}.settings.save_failed`));
    } finally {
      settingsSaving = false;
    }
  }

  function formatDate(dateStr: string): string {
    const date = new Date(dateStr);
    return date.toLocaleDateString($locale || 'en', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    });
  }

  function formatStorage(mb: number): string {
    if (mb < 1) return `${(mb * 1024).toFixed(0)} KB`;
    if (mb < 1024) return `${mb.toFixed(1)} MB`;
    return `${(mb / 1024).toFixed(2)} GB`;
  }

  function getActionLabel(action: string): string {
    const key = `${a}.activity.action_${action}`;
    const label = $_(key);
    return label !== key ? label : action;
  }

  // Derived values
  const stats = $derived(admin.getDetailedStats()?.stats || admin.getStats());
  const detailedStats = $derived(admin.getDetailedStats());
  const users = $derived(admin.getUsers());
  const activityLogs = $derived(admin.getActivityLogs());
  const activityTotal = $derived(admin.getActivityTotal());
  const _settings = $derived(admin.getSettings());
  const isLoading = $derived(admin.isLoading());
  const totalPages = $derived(Math.ceil(activityTotal / activityLimit));
</script>

<div class="h-full overflow-auto bg-background">
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
    <!-- Header -->
    <div class="mb-8">
      <h1 class="text-2xl font-bold text-foreground">{$_('page.admin.title')}</h1>
      <p class="text-muted-foreground">{$_('page.admin.subtitle')}</p>
    </div>

    <!-- Tabs -->
    <div class="border-b border-border mb-6">
      <nav class="-mb-px flex space-x-8">
        <button
          onclick={() => switchTab('dashboard')}
          class="py-2 px-1 border-b-2 font-medium text-sm transition-colors
						{activeTab === 'dashboard'
            ? 'border-primary text-primary'
            : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'}"
        >
          {$_('page.admin.tabs.dashboard')}
        </button>
        <button
          onclick={() => switchTab('users')}
          class="py-2 px-1 border-b-2 font-medium text-sm transition-colors
						{activeTab === 'users'
            ? 'border-primary text-primary'
            : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'}"
        >
          {$_('page.admin.tabs.users')}
        </button>
        <button
          onclick={() => switchTab('activity')}
          class="py-2 px-1 border-b-2 font-medium text-sm transition-colors
						{activeTab === 'activity'
            ? 'border-primary text-primary'
            : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'}"
        >
          {$_('page.admin.tabs.activity')}
        </button>
        <button
          onclick={() => switchTab('settings')}
          class="py-2 px-1 border-b-2 font-medium text-sm transition-colors
						{activeTab === 'settings'
            ? 'border-primary text-primary'
            : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'}"
        >
          {$_('page.admin.tabs.settings')}
        </button>
      </nav>
    </div>

    <!-- Loading State -->
    {#if isLoading}
      <div class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    {:else}
      <!-- Dashboard Tab -->
      {#if activeTab === 'dashboard'}
        <div class="space-y-6">
          <!-- Stats Cards -->
          {#if stats}
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
              <div class="bg-card rounded-lg p-6 shadow">
                <div class="text-sm font-medium text-muted-foreground">
                  {$_('page.admin.dashboard.users')}
                </div>
                <div class="text-2xl font-bold text-foreground">{stats.total_users}</div>
              </div>
              <div class="bg-card rounded-lg p-6 shadow">
                <div class="text-sm font-medium text-muted-foreground">
                  {$_('page.admin.dashboard.notes')}
                </div>
                <div class="text-2xl font-bold text-foreground">{stats.total_notes}</div>
              </div>
              <div class="bg-card rounded-lg p-6 shadow">
                <div class="text-sm font-medium text-muted-foreground">
                  {$_('page.admin.dashboard.folders')}
                </div>
                <div class="text-2xl font-bold text-foreground">{stats.total_folders}</div>
              </div>
              <div class="bg-card rounded-lg p-6 shadow">
                <div class="text-sm font-medium text-muted-foreground">
                  {$_('page.admin.dashboard.tags')}
                </div>
                <div class="text-2xl font-bold text-foreground">{stats.total_tags}</div>
              </div>
              <div class="bg-card rounded-lg p-6 shadow">
                <div class="text-sm font-medium text-muted-foreground">
                  {$_('page.admin.dashboard.storage')}
                </div>
                <div class="text-2xl font-bold text-foreground">
                  {formatStorage(stats.storage_used_mb)}
                </div>
              </div>
            </div>
          {/if}

          <!-- Charts -->
          {#if detailedStats}
            <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <!-- User Growth Chart -->
              <div class="bg-card rounded-lg p-6 shadow">
                <h3 class="text-lg font-medium text-foreground mb-4">
                  {$_('page.admin.dashboard.user_growth')}
                </h3>
                {#if detailedStats.user_growth && detailedStats.user_growth.length > 0}
                  {@const maxCount = Math.max(...detailedStats.user_growth.map((d) => d.count), 1)}
                  <div class="h-48 flex items-end space-x-1">
                    {#each detailedStats.user_growth as day (day.date)}
                      <div
                        class="flex-1 bg-primary rounded-t min-h-[4px]"
                        style="height: {(day.count / maxCount) * 100}%"
                        title={$_('page.admin.dashboard.user_growth_tooltip', {
                          values: { date: day.date, count: day.count },
                        })}
                      ></div>
                    {/each}
                  </div>
                {:else}
                  <div class="text-muted-foreground text-center py-8">
                    {$_('page.admin.dashboard.no_data')}
                  </div>
                {/if}
              </div>

              <!-- Note Growth Chart -->
              <div class="bg-card rounded-lg p-6 shadow">
                <h3 class="text-lg font-medium text-foreground mb-4">
                  {$_('page.admin.dashboard.note_activity')}
                </h3>
                {#if detailedStats.note_growth && detailedStats.note_growth.length > 0}
                  {@const maxCount = Math.max(...detailedStats.note_growth.map((d) => d.count), 1)}
                  <div class="h-48 flex items-end space-x-1">
                    {#each detailedStats.note_growth as day (day.date)}
                      <div
                        class="flex-1 bg-success rounded-t min-h-[4px]"
                        style="height: {(day.count / maxCount) * 100}%"
                        title={$_('page.admin.dashboard.note_activity_tooltip', {
                          values: { date: day.date, count: day.count },
                        })}
                      ></div>
                    {/each}
                  </div>
                {:else}
                  <div class="text-muted-foreground text-center py-8">
                    {$_('page.admin.dashboard.no_data')}
                  </div>
                {/if}
              </div>
            </div>
          {/if}
        </div>

        <!-- Users Tab -->
      {:else if activeTab === 'users'}
        <div class="bg-card rounded-lg shadow overflow-hidden">
          <table class="min-w-full divide-y divide-border">
            <thead class="bg-muted">
              <tr>
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_id')}</th
                >
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_username')}</th
                >
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_email')}</th
                >
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_notes')}</th
                >
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_storage')}</th
                >
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_2fa')}</th
                >
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_admin')}</th
                >
                <th
                  class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_created')}</th
                >
                <th
                  class="px-6 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider"
                  >{$_('page.admin.users.col_actions')}</th
                >
              </tr>
            </thead>
            <tbody class="bg-card divide-y divide-border">
              {#each users as user (user.id)}
                <tr>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-foreground">{user.id}</td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-foreground"
                    >{user.username}</td
                  >
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground"
                    >{user.email}</td
                  >
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground"
                    >{user.note_count}</td
                  >
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground"
                    >{formatStorage(user.storage_mb)}</td
                  >
                  <td class="px-6 py-4 whitespace-nowrap text-sm">
                    {#if user.totp_enabled}
                      <span
                        class="px-2 py-1 text-xs font-medium rounded-full bg-success/15 text-success"
                        title={user.totp_verified_at
                          ? $_('page.admin.users.totp_enabled_since', {
                              values: { date: formatDate(user.totp_verified_at) },
                            })
                          : $_('page.admin.users.totp_enabled')}
                      >
                        &#x2713;
                      </span>
                    {:else}
                      <span class="text-muted-foreground/50">&ndash;</span>
                    {/if}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm">
                    {#if user.is_admin}
                      <span
                        class="px-2 py-1 text-xs font-medium rounded-full bg-primary/15 text-primary"
                        >{$_('page.admin.users.role_admin')}</span
                      >
                    {:else}
                      <span
                        class="px-2 py-1 text-xs font-medium rounded-full bg-muted text-muted-foreground"
                        >{$_('page.admin.users.role_user')}</span
                      >
                    {/if}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground"
                    >{formatDate(user.created_at)}</td
                  >
                  <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                    <button
                      onclick={() => handleToggleAdmin(user)}
                      class="text-primary hover:text-primary/80"
                    >
                      {user.is_admin
                        ? $_('page.admin.users.remove_admin')
                        : $_('page.admin.users.make_admin')}
                    </button>
                    <button
                      onclick={() => confirmDeleteUser(user)}
                      class="text-destructive hover:text-destructive/80"
                    >
                      {$_('page.admin.users.delete')}
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <!-- Activity Tab -->
      {:else if activeTab === 'activity'}
        <div class="space-y-4">
          <!-- Filter -->
          <div class="flex items-center space-x-4">
            <select
              bind:value={activityFilter}
              onchange={() => {
                activityPage = 1;
                loadActivity();
              }}
              class="px-3 py-2 border border-border rounded-lg bg-card text-foreground"
            >
              <option value="">{$_('page.admin.activity.filter_all')}</option>
              <option value="login">{$_('page.admin.activity.action_login')}</option>
              <option value="logout">{$_('page.admin.activity.action_logout')}</option>
              <option value="register">{$_('page.admin.activity.action_register')}</option>
              <option value="note_create">{$_('page.admin.activity.action_note_create')}</option>
              <option value="note_update">{$_('page.admin.activity.action_note_update')}</option>
              <option value="note_delete">{$_('page.admin.activity.action_note_delete')}</option>
              <option value="settings_change"
                >{$_('page.admin.activity.action_settings_change')}</option
              >
            </select>
            <span class="text-sm text-muted-foreground">
              {$_('page.admin.activity.total_entries', { values: { count: activityTotal } })}
            </span>
          </div>

          <!-- Activity Table -->
          <div class="bg-card rounded-lg shadow overflow-hidden">
            <table class="min-w-full divide-y divide-border">
              <thead class="bg-muted">
                <tr>
                  <th
                    class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                    >{$_('page.admin.activity.col_time')}</th
                  >
                  <th
                    class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                    >{$_('page.admin.activity.col_user')}</th
                  >
                  <th
                    class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                    >{$_('page.admin.activity.col_action')}</th
                  >
                  <th
                    class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                    >{$_('page.admin.activity.col_details')}</th
                  >
                  <th
                    class="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider"
                    >{$_('page.admin.activity.col_ip')}</th
                  >
                </tr>
              </thead>
              <tbody class="bg-card divide-y divide-border">
                {#each activityLogs as log (log.id)}
                  <tr>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground"
                      >{formatDate(log.created_at)}</td
                    >
                    <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-foreground"
                      >{log.username || $_('page.admin.activity.unknown_user')}</td
                    >
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground"
                      >{getActionLabel(log.action)}</td
                    >
                    <td class="px-6 py-4 text-sm text-muted-foreground max-w-xs truncate">
                      {#if log.details}
                        {JSON.stringify(log.details)}
                      {:else}
                        -
                      {/if}
                    </td>
                    <td class="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground"
                      >{log.ip_address || '-'}</td
                    >
                  </tr>
                {:else}
                  <tr>
                    <td colspan="5" class="px-6 py-12 text-center text-muted-foreground"
                      >{$_('page.admin.activity.no_logs')}</td
                    >
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          {#if totalPages > 1}
            <div class="flex items-center justify-between">
              <button
                onclick={() => {
                  activityPage = Math.max(1, activityPage - 1);
                  loadActivity();
                }}
                disabled={activityPage === 1}
                class="px-4 py-2 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted disabled:opacity-50"
              >
                {$_('page.admin.activity.previous')}
              </button>
              <span class="text-sm text-muted-foreground">
                {$_('page.admin.activity.page_info', {
                  values: { page: activityPage, total: totalPages },
                })}
              </span>
              <button
                onclick={() => {
                  activityPage = Math.min(totalPages, activityPage + 1);
                  loadActivity();
                }}
                disabled={activityPage >= totalPages}
                class="px-4 py-2 text-sm font-medium text-foreground bg-card border border-border rounded-lg hover:bg-muted disabled:opacity-50"
              >
                {$_('page.admin.activity.next')}
              </button>
            </div>
          {/if}
        </div>

        <!-- Settings Tab -->
      {:else if activeTab === 'settings'}
        <div class="bg-card rounded-lg shadow p-6 max-w-2xl">
          <h3 class="text-lg font-medium text-foreground mb-6">
            {$_('page.admin.settings.title')}
          </h3>

          <div class="space-y-6">
            <!-- Registration -->
            <div class="flex items-center justify-between">
              <div>
                <span id="admin-registration-label" class="text-sm font-medium text-foreground"
                  >{$_('page.admin.settings.registration_enabled')}</span
                >
                <p class="text-sm text-muted-foreground">
                  {$_('page.admin.settings.registration_description')}
                </p>
              </div>
              <button
                onclick={() =>
                  (settingsForm.registration_enabled =
                    settingsForm.registration_enabled === 'true' ? 'false' : 'true')}
                role="switch"
                aria-checked={settingsForm.registration_enabled === 'true'}
                aria-labelledby="admin-registration-label"
                class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none
									{settingsForm.registration_enabled === 'true' ? 'bg-primary' : 'bg-muted'}"
              >
                <span
                  class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out
										{settingsForm.registration_enabled === 'true' ? 'translate-x-5' : 'translate-x-0'}"
                ></span>
              </button>
            </div>

            <!-- Max Notes -->
            <div>
              <label for="admin-max-notes" class="block text-sm font-medium text-foreground mb-1"
                >{$_('page.admin.settings.max_notes')}</label
              >
              <p class="text-sm text-muted-foreground mb-2">
                {$_('page.admin.settings.unlimited_hint')}
              </p>
              <input
                id="admin-max-notes"
                type="number"
                bind:value={settingsForm.max_notes_per_user}
                min="0"
                class="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground"
              />
            </div>

            <!-- Max Storage -->
            <div>
              <label for="admin-max-storage" class="block text-sm font-medium text-foreground mb-1"
                >{$_('page.admin.settings.max_storage')}</label
              >
              <p class="text-sm text-muted-foreground mb-2">
                {$_('page.admin.settings.unlimited_hint')}
              </p>
              <input
                id="admin-max-storage"
                type="number"
                bind:value={settingsForm.max_storage_mb_per_user}
                min="0"
                class="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground"
              />
            </div>

            <!-- Activity Retention -->
            <div>
              <label
                for="admin-activity-retention"
                class="block text-sm font-medium text-foreground mb-1"
                >{$_('page.admin.settings.activity_retention')}</label
              >
              <p class="text-sm text-muted-foreground mb-2">
                {$_('page.admin.settings.keep_forever_hint')}
              </p>
              <input
                id="admin-activity-retention"
                type="number"
                bind:value={settingsForm.activity_retention_days}
                min="0"
                class="w-full px-3 py-2 border border-border rounded-lg bg-background text-foreground"
              />
            </div>

            <!-- Maintenance Mode -->
            <div class="flex items-center justify-between">
              <div>
                <span id="admin-maintenance-label" class="text-sm font-medium text-foreground"
                  >{$_('page.admin.settings.maintenance_mode')}</span
                >
                <p class="text-sm text-muted-foreground">
                  {$_('page.admin.settings.maintenance_description')}
                </p>
              </div>
              <button
                onclick={() =>
                  (settingsForm.maintenance_mode =
                    settingsForm.maintenance_mode === 'true' ? 'false' : 'true')}
                role="switch"
                aria-checked={settingsForm.maintenance_mode === 'true'}
                aria-labelledby="admin-maintenance-label"
                class="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none
									{settingsForm.maintenance_mode === 'true' ? 'bg-destructive' : 'bg-muted'}"
              >
                <span
                  class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out
										{settingsForm.maintenance_mode === 'true' ? 'translate-x-5' : 'translate-x-0'}"
                ></span>
              </button>
            </div>

            <!-- Save Button -->
            <div class="pt-4">
              <button
                onclick={handleSaveSettings}
                disabled={settingsSaving}
                class="w-full px-4 py-2 text-sm font-medium text-primary-foreground bg-primary rounded-lg hover:bg-primary/90 disabled:opacity-50"
              >
                {settingsSaving ? $_('page.admin.settings.saving') : $_('page.admin.settings.save')}
              </button>
            </div>
          </div>
        </div>
      {/if}
    {/if}
  </div>
</div>

<!-- Delete Confirmation Modal -->
<BaseDialog
  open={deleteConfirmUser !== null}
  title={$_('page.admin.delete_dialog.title')}
  onClose={() => (deleteConfirmUser = null)}
  size="sm"
  variant="danger"
>
  {#snippet content()}
    <p class="text-muted-foreground">
      {$_('page.admin.delete_dialog.message_pre')}
      <strong>{deleteConfirmUser?.username ?? ''}</strong>
      {$_('page.admin.delete_dialog.message_post')}
    </p>
  {/snippet}
  {#snippet footer()}
    <button
      onclick={() => (deleteConfirmUser = null)}
      class="px-4 py-2 text-sm font-medium text-foreground bg-background border border-border rounded-lg hover:bg-muted"
    >
      {$_('page.admin.delete_dialog.cancel')}
    </button>
    <button
      onclick={handleDeleteUser}
      class="px-4 py-2 text-sm font-medium text-destructive-foreground bg-destructive rounded-lg hover:bg-destructive/90"
    >
      {$_('page.admin.delete_dialog.confirm')}
    </button>
  {/snippet}
</BaseDialog>
