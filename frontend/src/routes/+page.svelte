<script lang="ts">
  import {
    ChevronDown,
    ChevronUp,
    Clock3,
    FilePlus,
    FileText,
    Search,
    Sparkles,
  } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import { ApiError } from '$lib/api';
  import { getPreferences, updateHomeDashboardLayoutPreference } from '$lib/api/preferences';
  import type { HomeDashboardLayoutPreference } from '$lib/api/types';
  import CreateNoteDialog from '$lib/components/CreateNoteDialog.svelte';
  import DashboardSection from '$lib/components/DashboardSection.svelte';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import * as folders from '$lib/stores/folders.svelte';
  import * as notes from '$lib/stores/notes.svelte';
  import * as ui from '$lib/stores/ui.svelte';
  import { formatRelativeTime } from '$lib/utils/time';

  let showCreateNoteDialog = $state(false);
  let allNotesSort = $state<'updated' | 'created' | 'alpha'>('updated');
  let allNotesQuery = $state('');
  type DashboardSectionId = 'hero' | 'recent' | 'activity' | 'created' | 'all';
  type RightColumnSectionId = Exclude<DashboardSectionId, 'hero'>;
  const DASHBOARD_COLLAPSE_KEY = 'xelanote-home-collapsed-sections-v1';
  const DASHBOARD_ORDER_KEY = 'xelanote-home-right-order-v1';
  const DEFAULT_RIGHT_SECTION_ORDER: RightColumnSectionId[] = [
    'recent',
    'activity',
    'created',
    'all',
  ];
  let collapsedSections = $state<Record<DashboardSectionId, boolean>>({
    hero: false,
    recent: false,
    activity: false,
    created: false,
    all: true,
  });
  let rightSectionOrder = $state<RightColumnSectionId[]>([...DEFAULT_RIGHT_SECTION_ORDER]);
  let draggingSectionId = $state<RightColumnSectionId | null>(null);
  let dragOverSectionId = $state<RightColumnSectionId | null>(null);
  let homeLayoutServerSyncSupported = $state<boolean | null>(null);
  let homeLayoutSyncTimer = $state<ReturnType<typeof setTimeout> | null>(null);
  const recentNotes = $derived(notes.getRecentNotes(5));
  const latestNote = $derived(notes.getRecentNotes(1)[0] ?? null);
  const newlyCreatedNotes = $derived.by(() =>
    [...notes.getNotes()]
      .sort((a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime())
      .slice(0, 5)
  );
  const allNotes = $derived.by(() => {
    const items = [...notes.getNotes()];
    switch (allNotesSort) {
      case 'alpha':
        return items.sort((a, b) =>
          a.title.localeCompare(b.title, undefined, { sensitivity: 'base' })
        );
      case 'created':
        return items.sort(
          (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        );
      case 'updated':
      default:
        return items.sort(
          (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
        );
    }
  });
  const totalNotes = $derived(notes.getNotes().length);
  const filteredAllNotes = $derived.by(() => {
    const query = allNotesQuery.trim().toLowerCase();
    if (!query) return allNotes;
    return allNotes.filter(
      (note) =>
        note.title.toLowerCase().includes(query) || note.folder_path.toLowerCase().includes(query)
    );
  });
  function startOfToday(): number {
    // eslint-disable-next-line svelte/prefer-svelte-reactivity
    const d = new Date();
    d.setHours(0, 0, 0, 0);
    return d.getTime();
  }

  function startOfWeek(): number {
    // eslint-disable-next-line svelte/prefer-svelte-reactivity
    const d = new Date();
    const day = d.getDay(); // 0 = Sunday
    const diffToMonday = day === 0 ? 6 : day - 1;
    d.setDate(d.getDate() - diffToMonday);
    d.setHours(0, 0, 0, 0);
    return d.getTime();
  }

  const updatedTodayCount = $derived.by(() => {
    const ts = startOfToday();
    return notes.getNotes().filter((note) => Date.parse(note.updated_at) >= ts).length;
  });
  const updatedThisWeekCount = $derived.by(() => {
    const ts = startOfWeek();
    return notes.getNotes().filter((note) => Date.parse(note.updated_at) >= ts).length;
  });
  const withoutFolderCount = $derived.by(
    () => notes.getNotes().filter((note) => note.folder_path === '/').length
  );
  const notesUpdatedToday = $derived.by(() => {
    const ts = startOfToday();
    return [...notes.getNotes()]
      .filter((note) => Date.parse(note.updated_at) >= ts)
      .sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at))
      .slice(0, 5);
  });
  const notesUpdatedThisWeek = $derived.by(() => {
    const ts = startOfWeek();
    return [...notes.getNotes()]
      .filter((note) => Date.parse(note.updated_at) >= ts)
      .sort((a, b) => Date.parse(b.updated_at) - Date.parse(a.updated_at))
      .slice(0, 5);
  });

  function formatFolderPath(folderPath: string) {
    return folderPath === '/' ? 'Root' : folderPath;
  }

  function isSectionCollapsed(id: DashboardSectionId) {
    return collapsedSections[id];
  }

  function buildDashboardLayoutPreference(): HomeDashboardLayoutPreference {
    return {
      version: 1,
      collapsed_sections: {
        hero: collapsedSections.hero,
        recent: collapsedSections.recent,
        activity: collapsedSections.activity,
        created: collapsedSections.created,
        all: collapsedSections.all,
      },
      right_section_order: [...rightSectionOrder],
    };
  }

  function applyDashboardLayoutPreference(
    layout: HomeDashboardLayoutPreference | null | undefined
  ) {
    if (!layout || layout.version !== 1) return;
    collapsedSections = {
      ...collapsedSections,
      ...layout.collapsed_sections,
    };

    const valid = layout.right_section_order.filter((id) =>
      DEFAULT_RIGHT_SECTION_ORDER.includes(id)
    );
    const unique = [...new Set(valid)];
    if (unique.length === DEFAULT_RIGHT_SECTION_ORDER.length) {
      rightSectionOrder = unique;
    }
  }

  function queueDashboardLayoutServerSync() {
    if (typeof window === 'undefined') return;
    if (homeLayoutServerSyncSupported === false) return;
    if (homeLayoutSyncTimer) clearTimeout(homeLayoutSyncTimer);

    homeLayoutSyncTimer = setTimeout(async () => {
      homeLayoutSyncTimer = null;
      try {
        await updateHomeDashboardLayoutPreference(buildDashboardLayoutPreference());
        homeLayoutServerSyncSupported = true;
      } catch (error) {
        if (error instanceof ApiError && [400, 404, 405, 422].includes(error.status)) {
          homeLayoutServerSyncSupported = false;
        }
      }
    }, 350);
  }

  function persistCollapsedSections() {
    if (typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(DASHBOARD_COLLAPSE_KEY, JSON.stringify(collapsedSections));
    } catch (error) {
      console.warn('Could not persist dashboard collapse state', error);
    }
    queueDashboardLayoutServerSync();
  }

  function persistRightSectionOrder() {
    if (typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(DASHBOARD_ORDER_KEY, JSON.stringify(rightSectionOrder));
    } catch (error) {
      console.warn('Could not persist dashboard section order', error);
    }
    queueDashboardLayoutServerSync();
  }

  function toggleSection(id: DashboardSectionId) {
    collapsedSections = { ...collapsedSections, [id]: !collapsedSections[id] };
    persistCollapsedSections();
  }

  function resetDashboardLayout() {
    collapsedSections = {
      hero: false,
      recent: false,
      activity: false,
      created: false,
      all: true,
    };
    rightSectionOrder = [...DEFAULT_RIGHT_SECTION_ORDER];
    persistCollapsedSections();
    persistRightSectionOrder();
  }

  function loadDashboardLayoutPreferences() {
    if (typeof localStorage === 'undefined') return;
    try {
      const storedCollapsed = localStorage.getItem(DASHBOARD_COLLAPSE_KEY);
      if (storedCollapsed) {
        const parsed = JSON.parse(storedCollapsed) as Partial<Record<DashboardSectionId, boolean>>;
        collapsedSections = {
          ...collapsedSections,
          hero: parsed.hero ?? collapsedSections.hero,
          recent: parsed.recent ?? collapsedSections.recent,
          activity: parsed.activity ?? collapsedSections.activity,
          created: parsed.created ?? collapsedSections.created,
          all: parsed.all ?? collapsedSections.all,
        };
      }
    } catch (error) {
      console.warn('Could not load dashboard collapse state', error);
    }

    try {
      const storedOrder = localStorage.getItem(DASHBOARD_ORDER_KEY);
      if (!storedOrder) return;
      const parsed = JSON.parse(storedOrder);
      if (!Array.isArray(parsed)) return;
      const valid = parsed.filter((id): id is RightColumnSectionId =>
        DEFAULT_RIGHT_SECTION_ORDER.includes(id as RightColumnSectionId)
      );
      const unique = [...new Set(valid)];
      if (unique.length === DEFAULT_RIGHT_SECTION_ORDER.length) {
        rightSectionOrder = unique;
      }
    } catch (error) {
      console.warn('Could not load dashboard section order', error);
    }
  }

  async function loadDashboardLayoutFromServer() {
    try {
      const prefs = await getPreferences();
      if (Object.prototype.hasOwnProperty.call(prefs, 'home_dashboard_layout')) {
        homeLayoutServerSyncSupported = true;
        applyDashboardLayoutPreference(prefs.home_dashboard_layout ?? undefined);
      } else {
        homeLayoutServerSyncSupported = false;
      }
    } catch {
      // Non-blocking: localStorage remains the primary fallback
    }
  }

  function getRightSectionOrder(id: RightColumnSectionId) {
    const index = rightSectionOrder.indexOf(id);
    return index === -1 ? DEFAULT_RIGHT_SECTION_ORDER.indexOf(id) : index;
  }

  function moveRightSection(sourceId: RightColumnSectionId, targetId: RightColumnSectionId) {
    if (sourceId === targetId) return;
    const current = [...rightSectionOrder];
    const sourceIndex = current.indexOf(sourceId);
    const targetIndex = current.indexOf(targetId);
    if (sourceIndex === -1 || targetIndex === -1) return;
    current.splice(sourceIndex, 1);
    current.splice(targetIndex, 0, sourceId);
    rightSectionOrder = current;
    persistRightSectionOrder();
  }

  function handleSectionDragStart(event: DragEvent, id: RightColumnSectionId) {
    draggingSectionId = id;
    dragOverSectionId = null;
    event.dataTransfer?.setData('text/plain', id);
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = 'move';
    }
  }

  function handleSectionDragOver(event: DragEvent, id: RightColumnSectionId) {
    event.preventDefault();
    if (draggingSectionId && draggingSectionId !== id) {
      dragOverSectionId = id;
    }
  }

  function handleSectionDrop(event: DragEvent, targetId: RightColumnSectionId) {
    event.preventDefault();
    const sourceId = draggingSectionId;
    if (sourceId) {
      moveRightSection(sourceId, targetId);
    }
    draggingSectionId = null;
    dragOverSectionId = null;
  }

  function handleSectionDragEnd() {
    draggingSectionId = null;
    dragOverSectionId = null;
  }

  onMount(() => {
    loadDashboardLayoutPreferences();
    void loadDashboardLayoutFromServer();
    return () => {
      if (homeLayoutSyncTimer) clearTimeout(homeLayoutSyncTimer);
    };
  });

  async function handleCreateNoteConfirm(title: string) {
    const folderPath = folders.getSelectedFolder() || '/';
    try {
      const note = await notes.createNote(title, '', folderPath);
      await folders.loadFolders();
      goto(`/note/${note.id}`);
    } catch (e) {
      console.error('Failed to create note:', e);
    }
  }
</script>

<svelte:head>
  <title>xelanote</title>
</svelte:head>

<div class="relative h-full overflow-hidden bg-background">
  <div
    class="pointer-events-none absolute inset-0 opacity-70 [background:radial-gradient(50rem_24rem_at_70%_30%,color-mix(in_oklch,var(--color-primary),transparent_82%),transparent),radial-gradient(32rem_18rem_at_20%_15%,color-mix(in_oklch,var(--color-sidebar-accent),transparent_65%),transparent)]"
  ></div>
  <div
    class="pointer-events-none absolute inset-0 opacity-15 [background-image:linear-gradient(to_right,var(--color-border)_1px,transparent_1px),linear-gradient(to_bottom,var(--color-border)_1px,transparent_1px)] [background-size:28px_28px]"
  ></div>

  <div class="relative h-full overflow-y-auto">
    <PageHeader
      title={$_('page.home.dashboard_title')}
      subtitle={$_('page.home.dashboard_subtitle')}
      class="sticky top-0 z-10 px-4 py-3 sm:px-6 sm:py-4 lg:px-10"
      containerClass="mx-auto w-full max-w-7xl"
      subtitleClass="hidden sm:block"
    >
      {#snippet leading()}
        <MobileSidebarInlineToggle />
      {/snippet}
      {#snippet actions()}
        <button
          type="button"
          onclick={resetDashboardLayout}
          class="ui-button ui-button-secondary text-sm"
        >
          {$_('page.home.layout_reset')}
        </button>
      {/snippet}
    </PageHeader>

    <div
      class="mx-auto flex min-h-full w-full max-w-7xl items-start px-4 py-5 sm:px-6 sm:py-6 lg:px-10"
    >
      <div class="w-full">
        <div class="grid w-full grid-cols-1 gap-5 lg:grid-cols-[1.08fr_0.92fr] lg:gap-6">
          <section class="ui-panel relative overflow-hidden p-5 sm:p-6">
            <div
              class="pointer-events-none absolute -right-12 -top-10 h-40 w-40 rounded-full bg-primary/10 blur-2xl"
            ></div>

            <div class="relative">
              <div class="mb-4 flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  <div
                    class="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/50 px-3 py-1 text-xs font-medium text-muted-foreground"
                  >
                    <Sparkles size={13} class="text-primary" />
                    xelanote
                  </div>
                </div>
                <button
                  type="button"
                  onclick={() => toggleSection('hero')}
                  class="inline-flex items-center gap-1 rounded-lg border border-border/60 bg-background/40 px-2 py-1 text-xs text-muted-foreground transition hover:bg-accent/25"
                  aria-expanded={!isSectionCollapsed('hero')}
                  aria-label={$_(
                    isSectionCollapsed('hero')
                      ? 'page.home.section_expand'
                      : 'page.home.section_collapse'
                  )}
                >
                  {#if isSectionCollapsed('hero')}
                    <ChevronDown size={14} />
                  {:else}
                    <ChevronUp size={14} />
                  {/if}
                </button>
              </div>

              {#if !isSectionCollapsed('hero')}
                <div class="mt-1 grid gap-3 sm:max-w-md">
                  <button
                    onclick={() => (showCreateNoteDialog = true)}
                    class="group w-full rounded-xl border border-primary/25 bg-primary px-4 py-3 text-primary-foreground shadow-[0_8px_24px_-16px_var(--color-primary)] transition hover:brightness-105"
                  >
                    <span class="flex items-center justify-center gap-2 font-medium">
                      <FilePlus size={18} />
                      {$_('page.home.create_new_note')}
                    </span>
                  </button>

                  <button
                    onclick={() => ui.toggleQuickSwitcher()}
                    class="w-full rounded-xl border border-border/70 bg-background/40 px-4 py-3 text-left transition hover:bg-accent/40"
                  >
                    <span class="flex items-center gap-2.5">
                      <Search size={18} class="text-muted-foreground" />
                      <span class="flex-1 text-sm sm:text-[0.95rem]">
                        {$_('page.home.open_quick_search')}
                      </span>
                      <span
                        class="hidden sm:inline-flex rounded-md border border-border/70 bg-background/70 px-2 py-0.5 text-[11px] font-medium tracking-wide text-muted-foreground"
                      >
                        Ctrl+P
                      </span>
                    </span>
                  </button>
                </div>

                <div class="mt-5 flex flex-wrap gap-2">
                  <div
                    class="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-background/40 px-3 py-2 text-xs text-muted-foreground"
                  >
                    <FileText size={14} class="text-primary/85" />
                    <span>
                      {$_('page.home.notes_available', { values: { count: totalNotes } })}
                    </span>
                  </div>
                  <div
                    class="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-background/40 px-3 py-2 text-xs text-muted-foreground"
                  >
                    <Clock3 size={14} class="text-primary/85" />
                    <span>{$_('page.home.recently_edited')}</span>
                  </div>
                  <div
                    class="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-background/40 px-3 py-2 text-xs text-muted-foreground"
                  >
                    <Clock3 size={14} class="text-primary/85" />
                    <span
                      >{$_('page.home.edited_today', {
                        values: { count: updatedTodayCount },
                      })}</span
                    >
                  </div>
                  <div
                    class="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-background/40 px-3 py-2 text-xs text-muted-foreground"
                  >
                    <Clock3 size={14} class="text-primary/85" />
                    <span
                      >{$_('page.home.this_week', {
                        values: { count: updatedThisWeekCount },
                      })}</span
                    >
                  </div>
                  <div
                    class="inline-flex items-center gap-2 rounded-lg border border-border/60 bg-background/40 px-3 py-2 text-xs text-muted-foreground"
                  >
                    <FileText size={14} class="text-primary/85" />
                    <span
                      >{$_('page.home.in_root_folder', {
                        values: { count: withoutFolderCount },
                      })}</span
                    >
                  </div>
                </div>

                {#if latestNote}
                  <div class="mt-5 rounded-xl border border-border/60 bg-background/35 p-3.5">
                    <div class="mb-2 flex items-center justify-between gap-3">
                      <div class="text-xs font-medium text-foreground">
                        {$_('page.home.continue_working')}
                      </div>
                      <div class="text-[11px] text-muted-foreground">
                        {formatRelativeTime(latestNote.updated_at, $_)}
                      </div>
                    </div>
                    <button
                      onclick={() => goto(`/note/${latestNote.id}`)}
                      class="group w-full rounded-lg border border-border/60 bg-background/35 px-3 py-2.5 text-left transition hover:bg-accent/30"
                    >
                      <div class="flex items-center gap-3">
                        <span
                          class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/60"
                        >
                          <FileText size={15} class="text-primary/90" />
                        </span>
                        <span class="min-w-0 flex-1">
                          <span class="block truncate text-sm font-medium">{latestNote.title}</span>
                          <span class="block truncate text-xs text-muted-foreground">
                            {formatFolderPath(latestNote.folder_path)}
                          </span>
                        </span>
                      </div>
                    </button>
                  </div>
                {/if}
              {/if}
            </div>
          </section>

          <div class="flex flex-col gap-5">
            <DashboardSection
              title={$_('page.home.recently_edited')}
              subtitle={recentNotes.length > 0
                ? $_('page.home.items_count', { values: { count: recentNotes.length } })
                : '—'}
              collapsed={isSectionCollapsed('recent')}
              isDragOver={dragOverSectionId === 'recent'}
              order={getRightSectionOrder('recent')}
              ariaLabel={$_('page.home.recently_edited_section')}
              onToggle={() => toggleSection('recent')}
              onDragStart={(event) => handleSectionDragStart(event, 'recent')}
              onDragOver={(event) => handleSectionDragOver(event, 'recent')}
              onDrop={(event) => handleSectionDrop(event, 'recent')}
              onDragEnd={handleSectionDragEnd}
            >
              {#if recentNotes.length > 0}
                <div class="space-y-1.5">
                  {#each recentNotes as note (note.id)}
                    <button
                      onclick={() => goto(`/note/${note.id}`)}
                      class="group w-full rounded-xl border border-transparent bg-background/25 px-3 py-2.5 text-left transition hover:border-border/60 hover:bg-accent/30"
                    >
                      <span class="flex items-center gap-3">
                        <span
                          class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/50"
                        >
                          <FileText size={15} class="text-primary/90" />
                        </span>
                        <span class="min-w-0 flex-1">
                          <span class="block truncate text-sm font-medium">{note.title}</span>
                          <span class="block text-xs text-muted-foreground">
                            {formatFolderPath(note.folder_path)} • {formatRelativeTime(
                              note.updated_at,
                              $_
                            )}
                          </span>
                        </span>
                      </span>
                    </button>
                  {/each}
                </div>
              {:else if totalNotes > 0}
                <div
                  class="rounded-xl border border-dashed border-border/60 bg-background/25 p-4 text-sm text-muted-foreground"
                >
                  {$_('page.home.notes_available', { values: { count: totalNotes } })}
                </div>
              {:else}
                <div
                  class="rounded-xl border border-dashed border-border/60 bg-background/25 p-5 text-sm text-muted-foreground"
                >
                  {$_('page.home.no_notes_hint')}
                </div>
              {/if}
            </DashboardSection>

            <DashboardSection
              title={$_('page.home.activity_title')}
              subtitle={$_('page.home.activity_subtitle')}
              collapsed={isSectionCollapsed('activity')}
              isDragOver={dragOverSectionId === 'activity'}
              order={getRightSectionOrder('activity')}
              ariaLabel={$_('page.home.activity_section')}
              onToggle={() => toggleSection('activity')}
              onDragStart={(event) => handleSectionDragStart(event, 'activity')}
              onDragOver={(event) => handleSectionDragOver(event, 'activity')}
              onDrop={(event) => handleSectionDrop(event, 'activity')}
              onDragEnd={handleSectionDragEnd}
            >
              <div class="grid gap-3 md:grid-cols-2">
                <div class="rounded-xl border border-border/60 bg-background/20 p-3">
                  <div class="mb-2 text-xs font-medium text-foreground">
                    {$_('page.home.today_count', { values: { count: updatedTodayCount } })}
                  </div>
                  {#if notesUpdatedToday.length > 0}
                    <div class="space-y-1.5">
                      {#each notesUpdatedToday as note (note.id)}
                        <button
                          onclick={() => goto(`/note/${note.id}`)}
                          class="w-full rounded-lg border border-transparent bg-background/20 px-2.5 py-2 text-left transition hover:border-border/60 hover:bg-accent/25"
                        >
                          <div class="truncate text-sm font-medium">{note.title}</div>
                          <div class="truncate text-xs text-muted-foreground">
                            {formatRelativeTime(note.updated_at, $_)}
                          </div>
                        </button>
                      {/each}
                    </div>
                  {:else}
                    <div class="text-xs text-muted-foreground">
                      {$_('page.home.no_changes_today')}
                    </div>
                  {/if}
                </div>

                <div class="rounded-xl border border-border/60 bg-background/20 p-3">
                  <div class="mb-2 text-xs font-medium text-foreground">
                    {$_('page.home.this_week_count', { values: { count: updatedThisWeekCount } })}
                  </div>
                  {#if notesUpdatedThisWeek.length > 0}
                    <div class="space-y-1.5">
                      {#each notesUpdatedThisWeek as note (note.id)}
                        <button
                          onclick={() => goto(`/note/${note.id}`)}
                          class="w-full rounded-lg border border-transparent bg-background/20 px-2.5 py-2 text-left transition hover:border-border/60 hover:bg-accent/25"
                        >
                          <div class="truncate text-sm font-medium">{note.title}</div>
                          <div class="truncate text-xs text-muted-foreground">
                            {formatRelativeTime(note.updated_at, $_)}
                          </div>
                        </button>
                      {/each}
                    </div>
                  {:else}
                    <div class="text-xs text-muted-foreground">
                      {$_('page.home.no_changes_week')}
                    </div>
                  {/if}
                </div>
              </div>
            </DashboardSection>

            <DashboardSection
              title={$_('page.home.recently_created')}
              subtitle={newlyCreatedNotes.length > 0
                ? $_('page.home.items_count', { values: { count: newlyCreatedNotes.length } })
                : '—'}
              collapsed={isSectionCollapsed('created')}
              isDragOver={dragOverSectionId === 'created'}
              order={getRightSectionOrder('created')}
              ariaLabel={$_('page.home.recently_created_section')}
              onToggle={() => toggleSection('created')}
              onDragStart={(event) => handleSectionDragStart(event, 'created')}
              onDragOver={(event) => handleSectionDragOver(event, 'created')}
              onDrop={(event) => handleSectionDrop(event, 'created')}
              onDragEnd={handleSectionDragEnd}
            >
              {#if newlyCreatedNotes.length > 0}
                <div class="space-y-1.5">
                  {#each newlyCreatedNotes as note (note.id)}
                    <button
                      onclick={() => goto(`/note/${note.id}`)}
                      class="group w-full rounded-xl border border-transparent bg-background/20 px-3 py-2.5 text-left transition hover:border-border/60 hover:bg-accent/25"
                    >
                      <span class="flex items-center gap-3">
                        <span
                          class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/50"
                        >
                          <FileText size={15} class="text-primary/90" />
                        </span>
                        <span class="min-w-0 flex-1">
                          <span class="block truncate text-sm font-medium">{note.title}</span>
                          <span class="block truncate text-xs text-muted-foreground">
                            {formatFolderPath(note.folder_path)} • {$_('page.home.created_prefix')}
                            {formatRelativeTime(note.created_at, $_)}
                          </span>
                        </span>
                      </span>
                    </button>
                  {/each}
                </div>
              {:else}
                <div class="text-sm text-muted-foreground">{$_('page.home.no_notes_hint')}</div>
              {/if}
            </DashboardSection>

            <DashboardSection
              title={$_('page.home.all_notes')}
              subtitle={totalNotes > 0
                ? $_('page.home.total_count', { values: { count: totalNotes } })
                : '—'}
              collapsed={isSectionCollapsed('all')}
              isDragOver={dragOverSectionId === 'all'}
              order={getRightSectionOrder('all')}
              ariaLabel={$_('page.home.all_notes_section')}
              onToggle={() => toggleSection('all')}
              onDragStart={(event) => handleSectionDragStart(event, 'all')}
              onDragOver={(event) => handleSectionDragOver(event, 'all')}
              onDrop={(event) => handleSectionDrop(event, 'all')}
              onDragEnd={handleSectionDragEnd}
            >
              {#if totalNotes > 0}
                <div class="space-y-3">
                  <div>
                    <label for="home-notes-search" class="sr-only"
                      >{$_('page.home.search_notes')}</label
                    >
                    <div
                      class="flex items-center gap-2 rounded-xl border border-border/70 bg-background/35 px-3 py-2"
                    >
                      <Search size={15} class="text-muted-foreground" />
                      <input
                        id="home-notes-search"
                        bind:value={allNotesQuery}
                        type="text"
                        placeholder={$_('page.home.filter_placeholder')}
                        class="w-full bg-transparent text-sm outline-none placeholder:text-muted-foreground"
                      />
                    </div>
                  </div>

                  <div class="flex flex-wrap gap-2">
                    <button
                      onclick={() => (allNotesSort = 'updated')}
                      class={`rounded-full border px-3 py-1 text-xs transition ${
                        allNotesSort === 'updated'
                          ? 'border-primary/30 bg-primary/15 text-foreground'
                          : 'border-border/60 bg-background/35 text-muted-foreground hover:bg-accent/30'
                      }`}
                    >
                      {$_('page.home.sort_recent')}
                    </button>
                    <button
                      onclick={() => (allNotesSort = 'created')}
                      class={`rounded-full border px-3 py-1 text-xs transition ${
                        allNotesSort === 'created'
                          ? 'border-primary/30 bg-primary/15 text-foreground'
                          : 'border-border/60 bg-background/35 text-muted-foreground hover:bg-accent/30'
                      }`}
                    >
                      {$_('page.home.sort_newest')}
                    </button>
                    <button
                      onclick={() => (allNotesSort = 'alpha')}
                      class={`rounded-full border px-3 py-1 text-xs transition ${
                        allNotesSort === 'alpha'
                          ? 'border-primary/30 bg-primary/15 text-foreground'
                          : 'border-border/60 bg-background/35 text-muted-foreground hover:bg-accent/30'
                      }`}
                    >
                      {$_('page.home.sort_az')}
                    </button>
                  </div>

                  <div class="max-h-[22rem] space-y-1.5 overflow-y-auto pr-1">
                    {#if filteredAllNotes.length > 0}
                      {#each filteredAllNotes as note (note.id)}
                        <button
                          onclick={() => goto(`/note/${note.id}`)}
                          class="group w-full rounded-xl border border-transparent bg-background/20 px-3 py-2.5 text-left transition hover:border-border/60 hover:bg-accent/25"
                        >
                          <span class="flex items-center gap-3">
                            <span
                              class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/50"
                            >
                              <FileText size={15} class="text-primary/90" />
                            </span>
                            <span class="min-w-0 flex-1">
                              <span class="block truncate text-sm font-medium">{note.title}</span>
                              <span class="block truncate text-xs text-muted-foreground">
                                {formatFolderPath(note.folder_path)} • {formatRelativeTime(
                                  note.updated_at,
                                  $_
                                )}
                              </span>
                            </span>
                          </span>
                        </button>
                      {/each}
                    {:else}
                      <button
                        type="button"
                        disabled
                        class="w-full rounded-xl border border-dashed border-border/60 bg-background/20 px-3 py-4 text-left text-sm text-muted-foreground"
                      >
                        {$_('page.home.no_filter_results')}
                      </button>
                    {/if}
                  </div>
                </div>
              {:else}
                <div
                  class="rounded-xl border border-dashed border-border/60 bg-background/25 p-5 text-sm text-muted-foreground"
                >
                  {$_('page.home.no_notes_hint')}
                </div>
              {/if}
            </DashboardSection>
          </div>
        </div>
      </div>
    </div>
  </div>
</div>

<CreateNoteDialog
  open={showCreateNoteDialog}
  folderPath={folders.getSelectedFolder() || '/'}
  onClose={() => (showCreateNoteDialog = false)}
  onCreate={handleCreateNoteConfirm}
/>
