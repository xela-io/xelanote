<script lang="ts">
  import {
    Calendar,
    ChevronDown,
    ChevronLeft,
    ChevronRight,
    ChevronUp,
    Loader2,
    Lock,
  } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import { goto } from '$app/navigation';
  import JournalActivityWidget from '$lib/components/JournalActivityWidget.svelte';
  import JournalHeatmap from '$lib/components/JournalHeatmap.svelte';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';
  import * as features from '$lib/stores/features.svelte';
  import * as journal from '$lib/stores/journal.svelte';
  import * as notes from '$lib/stores/notes.svelte';

  const featureEnabled = $derived(features.getJournalFeatureEnabled());
  const featureLoaded = $derived(features.getJournalFeatureLoaded());
  const entryList = $derived(journal.getEntries());
  const loading = $derived(journal.getEntriesLoading());
  const calendarDates = $derived(journal.getCalendarDates());
  const calendarYear = $derived(journal.getCalendarYear());
  const calendarMonth = $derived(journal.getCalendarMonth());
  const calendarLoading = $derived(journal.getCalendarLoading());
  const journalLoading = $derived(journal.getJournalLoading());
  const today = $derived(journal.getTodayDate());
  const isEncryptionLocked = $derived(!encryption.isEncryptionUnlocked());

  // Mobile calendar collapsed state
  const COLLAPSED_KEY = 'xelanote_journal_page_collapsed';
  let mobileCalendarCollapsed = $state(true);

  const monthNames = [
    'Jan',
    'Feb',
    'Mar',
    'Apr',
    'May',
    'Jun',
    'Jul',
    'Aug',
    'Sep',
    'Oct',
    'Nov',
    'Dec',
  ];
  const dayNames = ['Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa', 'Su'];

  const calendarDays = $derived.by(() => {
    const firstDay = new Date(calendarYear, calendarMonth - 1, 1);
    const lastDay = new Date(calendarYear, calendarMonth, 0);
    const daysInMonth = lastDay.getDate();

    let startDay = firstDay.getDay() - 1;
    if (startDay < 0) startDay = 6;

    const days: Array<{ date: string; day: number; isCurrentMonth: boolean }> = [];

    const prevMonth = calendarMonth === 1 ? 12 : calendarMonth - 1;
    const prevYear = calendarMonth === 1 ? calendarYear - 1 : calendarYear;
    const daysInPrevMonth = new Date(prevYear, prevMonth, 0).getDate();
    for (let i = startDay - 1; i >= 0; i--) {
      const day = daysInPrevMonth - i;
      const dateStr = `${prevYear}-${String(prevMonth).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
      days.push({ date: dateStr, day, isCurrentMonth: false });
    }

    for (let day = 1; day <= daysInMonth; day++) {
      const dateStr = `${calendarYear}-${String(calendarMonth).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
      days.push({ date: dateStr, day, isCurrentMonth: true });
    }

    const nextMonth = calendarMonth === 12 ? 1 : calendarMonth + 1;
    const nextYear = calendarMonth === 12 ? calendarYear + 1 : calendarYear;
    const remainingDays = 42 - days.length;
    for (let day = 1; day <= remainingDays; day++) {
      const dateStr = `${nextYear}-${String(nextMonth).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
      days.push({ date: dateStr, day, isCurrentMonth: false });
    }

    return days;
  });

  function formatEntryDate(dateStr: string): string {
    const d = new Date(dateStr + 'T00:00:00');
    return d.toLocaleDateString(undefined, {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }

  async function handleDateClick(date: string) {
    if (journalLoading) return;
    await journal.openJournalForDate(date);
  }

  async function handleOpenToday() {
    if (journalLoading) return;
    await journal.openJournalForDate(journal.getTodayDate());
  }

  function toggleMobileCalendar() {
    mobileCalendarCollapsed = !mobileCalendarCollapsed;
    try {
      localStorage.setItem(COLLAPSED_KEY, JSON.stringify(mobileCalendarCollapsed));
    } catch {
      // localStorage might not be available
    }
  }

  // Redirect when feature is confirmed disabled (wait for load to complete)
  $effect(() => {
    if (featureLoaded && !featureEnabled) {
      goto('/');
    }
  });

  // Track whether data has been loaded to avoid re-fetching
  let dataLoaded = false;

  onMount(() => {
    // Load collapsed state from localStorage
    try {
      const stored = localStorage.getItem(COLLAPSED_KEY);
      if (stored !== null) {
        const parsed = parseStoredCollapsed(stored);
        if (parsed !== null) {
          mobileCalendarCollapsed = parsed;
        }
      }
    } catch {
      // localStorage might not be available
    }
  });

  function parseStoredCollapsed(raw: string): boolean | null {
    try {
      const parsed = JSON.parse(raw);
      return typeof parsed === 'boolean' ? parsed : null;
    } catch {
      return null;
    }
  }

  // Load data reactively once feature is confirmed enabled and encryption is unlocked
  $effect(() => {
    if (featureLoaded && featureEnabled && !isEncryptionLocked && !dataLoaded) {
      dataLoaded = true;
      Promise.all([
        journal.loadEntries(),
        journal.loadCalendar(calendarYear, calendarMonth),
        // force=true bypasses cache so heatmap reflects deletions made on other pages
        journal.loadYearCalendar(new Date().getFullYear(), true),
      ]);
    }
  });

  // Reset when encryption locks again, so data reloads after unlock
  $effect(() => {
    if (isEncryptionLocked) {
      dataLoaded = false;
    }
  });
</script>

<svelte:head>
  <title>{$_('page.journal.title')} - xelanote</title>
</svelte:head>

<div class="h-full flex flex-col">
  <!-- Header -->
  <PageHeader
    title={$_('page.journal.title')}
    class="shrink-0 px-4 py-2.5 sm:px-6 sm:py-4"
    titleClass="min-w-0 truncate text-xl font-bold"
  >
    {#snippet leading()}
      <div
        class="grid grid-cols-[2.5rem_minmax(0,1fr)] items-center gap-1.5 sm:flex sm:items-center sm:gap-3"
      >
        <MobileSidebarInlineToggle />
      </div>
    {/snippet}
    {#snippet actions()}
      <button
        onclick={handleOpenToday}
        disabled={journalLoading}
        class="ui-button ui-button-primary self-start sm:self-auto px-2.5 py-1 text-xs sm:px-3 sm:py-1.5 sm:text-sm"
      >
        {#if journalLoading}
          <Loader2 size={16} class="animate-spin" />
        {/if}
        <span class="sm:hidden">Heute öffnen</span>
        <span class="hidden sm:inline">{$_('page.journal.openToday')}</span>
      </button>
    {/snippet}
  </PageHeader>

  {#if !featureLoaded || loading}
    <div class="flex items-center justify-center flex-1">
      <Loader2 class="w-8 h-8 animate-spin" />
    </div>
  {:else if isEncryptionLocked}
    <div class="flex flex-col items-center justify-center flex-1 gap-4 text-muted-foreground">
      <Lock class="w-12 h-12 opacity-50" />
      <p class="text-center max-w-sm">{$_('page.journal.encryptionRequired')}</p>
      <button onclick={() => notes.signalEncryptionLocked()} class="ui-button ui-button-primary">
        <Lock size={16} />
        {$_('page.journal.unlockButton')}
      </button>
    </div>
  {:else}
    <div class="flex-1 overflow-y-auto p-6">
      <div class="max-w-5xl">
        <!-- Mobile Layout: Activity widget, entries, then collapsible calendar -->
        <div class="md:hidden">
          <!-- Activity Widget -->
          <div class="mb-4">
            <JournalActivityWidget />
          </div>

          <!-- Entry List -->
          {@render entryListBlock()}

          <!-- Collapsible Calendar -->
          <div class="ui-panel mt-6">
            <button
              class="w-full flex items-center justify-between p-3 hover:bg-accent/30 rounded-2xl transition-colors"
              onclick={toggleMobileCalendar}
            >
              <span class="flex items-center gap-2 font-medium text-sm">
                <Calendar size={16} />
                {mobileCalendarCollapsed
                  ? $_('page.journal.showCalendar')
                  : $_('page.journal.hideCalendar')}
              </span>
              {#if mobileCalendarCollapsed}
                <ChevronDown size={16} />
              {:else}
                <ChevronUp size={16} />
              {/if}
            </button>
            {#if !mobileCalendarCollapsed}
              <div class="px-3 pb-3">
                {@render calendarBlock()}
              </div>
            {/if}
          </div>
        </div>

        <!-- Year Heatmap (Desktop only) -->
        <div class="hidden md:block mb-6">
          <JournalHeatmap />
        </div>

        <!-- Desktop Layout: 2-column grid -->
        <div class="hidden md:grid md:grid-cols-2 gap-6">
          <!-- Calendar (left) -->
          <div>
            <h2 class="ui-kicker mb-3">
              {$_('page.journal.title')}
            </h2>
            <div class="ui-panel p-4">
              {@render calendarBlock()}
            </div>
          </div>

          <!-- Entry List (right) -->
          <div>
            {@render entryListBlock()}
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Calendar Snippet -->
{#snippet calendarBlock()}
  <!-- Month Navigation -->
  <div class="flex items-center justify-between mb-3">
    <button
      class="ui-icon-button ui-icon-button-sm disabled:opacity-50"
      onclick={() => journal.previousMonth()}
      disabled={calendarLoading}
      title={$_('page.journal.previousMonth')}
    >
      <ChevronLeft size={18} />
    </button>
    <span class="font-medium text-sm">{monthNames[calendarMonth - 1]} {calendarYear}</span>
    <button
      class="ui-icon-button ui-icon-button-sm disabled:opacity-50"
      onclick={() => journal.nextMonth()}
      disabled={calendarLoading}
      title={$_('page.journal.nextMonth')}
    >
      <ChevronRight size={18} />
    </button>
  </div>

  <!-- Day headers -->
  <div class="grid grid-cols-7 gap-1 mb-1">
    {#each dayNames as dayName (dayName)}
      <div class="text-center text-xs text-muted-foreground font-medium">{dayName}</div>
    {/each}
  </div>

  <!-- Calendar grid -->
  <div class="grid grid-cols-7 gap-1">
    {#each calendarDays as { date, day, isCurrentMonth } (date)}
      {@const isToday = date === today}
      {@const hasEntry = calendarDates.includes(date)}
      <button
        class="calendar-day h-8 text-sm rounded transition-colors
					{!isCurrentMonth ? 'opacity-40' : ''}
					{isToday ? 'font-bold ring-2 ring-primary ring-inset' : ''}
					{hasEntry ? 'bg-primary/20 hover:bg-primary/30' : 'hover:bg-accent'}"
        onclick={() => handleDateClick(date)}
        disabled={journalLoading}>{day}</button
      >
    {/each}
  </div>
{/snippet}

<!-- Entry List Snippet -->
{#snippet entryListBlock()}
  <div>
    <h2 class="ui-kicker mb-3">
      {$_('page.journal.allEntries')} ({entryList.length})
    </h2>

    <div class="ui-panel p-4">
      {#if entryList.length === 0}
        <div class="ui-empty-state ui-empty-state-compact">
          <Calendar class="w-10 h-10 opacity-50" />
          <p class="text-sm">{$_('page.journal.noEntries')}</p>
        </div>
      {:else}
        <div class="space-y-2.5">
          {#each entryList as entry (entry.id)}
            <button
              onclick={() => goto(`/note/${entry.id}`)}
              class="ui-list-item w-full text-left p-3"
            >
              <div class="text-xs text-muted-foreground">
                {formatEntryDate(entry.journal_date)}
              </div>
              <div class="flex items-center gap-2 mt-1">
                <span class="font-medium text-sm flex-1 truncate">{entry.title}</span>
                {#if entry.content_encrypted}
                  <Lock size={12} class="text-muted-foreground shrink-0" />
                {/if}
              </div>
            </button>
          {/each}
        </div>
      {/if}
    </div>
  </div>
{/snippet}

<style>
  @media (pointer: coarse) {
    .calendar-day {
      min-height: 36px;
    }
  }
</style>
