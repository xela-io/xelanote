<script lang="ts">
  import { AlertTriangle, Check, Lock, RefreshCw, Unlock } from 'lucide-svelte';
  import { onMount } from 'svelte';
  import { _ } from 'svelte-i18n';

  import type { Note } from '$lib/api';
  import * as api from '$lib/api';
  import MobileSidebarInlineToggle from '$lib/components/MobileSidebarInlineToggle.svelte';
  import PageHeader from '$lib/components/ui/PageHeader.svelte';
  import * as encryption from '$lib/stores/encryption.svelte';

  let notes = $state<Note[]>([]);
  let loading = $state(false);
  let migrating = $state(false);
  let error = $state<string | null>(null);
  let migrationProgress = $state(0);
  let totalNotes = $state(0);
  let migratedNotes = $state(0);
  let failedNotes = $state<string[]>([]);
  let completed = $state(false);

  const plaintextNotes = $derived(
    notes.filter((note) => !note.content_encrypted || note.encryption_version === 0)
  );
  const encryptedNotes = $derived(
    notes.filter((note) => note.content_encrypted && note.encryption_version === 1)
  );

  async function loadNotes() {
    loading = true;
    error = null;
    try {
      const result = await api.listNotes({ limit: 10000 }); // Get all notes
      notes = result.notes;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load notes';
    } finally {
      loading = false;
    }
  }

  async function migrateNotes() {
    if (!encryption.isEncryptionUnlocked()) {
      error = 'Encryption is locked. Please login first.';
      return;
    }

    migrating = true;
    error = null;
    migratedNotes = 0;
    failedNotes = [];
    totalNotes = plaintextNotes.length;
    completed = false;

    for (let i = 0; i < plaintextNotes.length; i++) {
      const note = plaintextNotes[i];

      try {
        // Encrypt the note
        const { encryptedTitle, encryptedContent, keywords } = encryption.encryptNote(
          note.title,
          note.content
        );

        // Update the note with encrypted content
        await api.updateNote(
          note.id,
          {
            title: encryptedTitle ? '' : note.title,
            encrypted_title: encryptedTitle,
            title_encrypted: !!encryptedTitle,
            encrypted_content: encryptedContent.ciphertext,
            wrapped_dek: encryptedContent.metadata.wrapped_dek,
            encryption_metadata: JSON.stringify(encryptedContent.metadata),
            keywords: keywords,
            folder_path: note.folder_path,
          },
          note.version
        );

        migratedNotes++;
      } catch (e) {
        console.error(`Failed to migrate note ${note.id}:`, e);
        failedNotes.push(note.id);
      }

      migrationProgress = ((i + 1) / totalNotes) * 100;

      // Brief pause to prevent overwhelming the server
      await new Promise((resolve) => setTimeout(resolve, 50));
    }

    migrating = false;
    completed = true;

    // Reload notes to reflect migration
    await loadNotes();
  }

  // Load notes on mount
  onMount(() => {
    loadNotes();
  });

  const isUnlocked = $derived(encryption.isEncryptionUnlocked());
</script>

<svelte:head>
  <title>Notiz-Migration - xelanote</title>
</svelte:head>

<div class="ui-page-shell overflow-y-auto">
  <PageHeader
    title="Notiz-Migration"
    subtitle="Migriere bestehende Klartext-Notizen zu Ende-zu-Ende verschlüsselten Notizen"
    class="sticky top-0 z-10 px-4 py-3 sm:px-6 sm:py-4"
    containerClass="mx-auto max-w-4xl"
    subtitleClass="hidden sm:block"
  >
    {#snippet leading()}
      <MobileSidebarInlineToggle />
      <RefreshCw class="w-5 h-5 text-primary" />
    {/snippet}
  </PageHeader>

  <div class="mx-auto w-full max-w-4xl px-4 py-5 sm:px-6 sm:py-6">
    <!-- Encryption Status Warning -->
    {#if !isUnlocked}
      <div
        class="ui-panel-soft mb-6 p-4 bg-yellow-50 border-yellow-200 dark:bg-yellow-900/20 dark:border-yellow-800"
      >
        <div class="flex items-center gap-2 mb-2">
          <AlertTriangle class="w-5 h-5 text-yellow-600 dark:text-yellow-400" />
          <span class="font-semibold text-yellow-800 dark:text-yellow-300">
            Verschlüsselung gesperrt
          </span>
        </div>
        <p class="text-sm text-yellow-800 dark:text-yellow-300">
          Melde dich an, um die Migration durchzuführen. Dein Passwort wird benötigt, um die
          Verschlüsselung zu aktivieren.
        </p>
      </div>
    {/if}

    <!-- Loading State -->
    {#if loading}
      <div class="ui-panel-soft flex items-center justify-center py-12">
        <RefreshCw class="w-8 h-8 animate-spin text-primary" />
        <span class="ml-3 text-muted-foreground">Lade Notizen...</span>
      </div>
    {:else}
      <!-- Statistics -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div class="ui-panel p-5 sm:p-6">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-muted-foreground mb-1">Gesamt</p>
              <p class="text-3xl font-bold text-foreground">{notes.length}</p>
            </div>
            <RefreshCw class="w-8 h-8 text-muted-foreground" />
          </div>
        </div>

        <div class="ui-panel p-5 sm:p-6">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-muted-foreground mb-1">Verschlüsselt</p>
              <p class="text-3xl font-bold text-success">
                {encryptedNotes.length}
              </p>
            </div>
            <Lock class="w-8 h-8 text-success" />
          </div>
        </div>

        <div class="ui-panel p-5 sm:p-6">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-muted-foreground mb-1">Klartext</p>
              <p class="text-3xl font-bold text-warning">
                {plaintextNotes.length}
              </p>
            </div>
            <Unlock class="w-8 h-8 text-warning" />
          </div>
        </div>
      </div>

      <!-- Migration Section -->
      {#if plaintextNotes.length > 0}
        <div class="ui-panel p-5 sm:p-6 mb-6">
          <h2 class="text-xl font-semibold mb-4 flex items-center gap-2">
            <RefreshCw class="w-6 h-6 text-primary" />
            Migration starten
          </h2>

          <div class="space-y-4">
            <div class="ui-panel-soft p-4 bg-primary/10 border-primary/30">
              <p class="text-sm text-foreground mb-3">
                <strong>{plaintextNotes.length} Notiz(en)</strong> werden von Klartext zu verschlüsseltem
                Format migriert.
              </p>
              <ul class="space-y-1 text-sm text-muted-foreground">
                <li>• Alle Notizen werden Ende-zu-Ende verschlüsselt</li>
                <li>• Die Migration kann einige Minuten dauern</li>
                <li>• Du kannst während der Migration weiterarbeiten</li>
                <li>
                  • <strong class="text-foreground">WICHTIG:</strong> Stelle sicher, dass du einen Recovery-Key
                  hast!
                </li>
              </ul>
            </div>

            {#if migrating}
              <!-- Progress Bar -->
              <div class="space-y-2">
                <div class="flex items-center justify-between text-sm text-muted-foreground">
                  <span>Migration läuft...</span>
                  <span>{migratedNotes} / {totalNotes}</span>
                </div>
                <div class="w-full bg-muted rounded-full h-3 overflow-hidden">
                  <div
                    class="bg-primary h-3 rounded-full transition-all duration-300"
                    style="width: {migrationProgress}%"
                  ></div>
                </div>
              </div>
            {:else if completed}
              <!-- Completion Message -->
              <div class="ui-panel-soft p-4 bg-success/10 border-success/30">
                <div class="flex items-center gap-2 mb-2">
                  <Check class="w-5 h-5 text-success" />
                  <span class="font-semibold text-success"> Migration abgeschlossen! </span>
                </div>
                <p class="text-sm text-success">
                  {migratedNotes} Notiz(en) erfolgreich verschlüsselt.
                </p>
                {#if failedNotes.length > 0}
                  <p class="text-sm text-red-800 dark:text-red-300 mt-2">
                    {failedNotes.length} Notiz(en) konnten nicht migriert werden.
                  </p>
                {/if}
              </div>
            {:else}
              <!-- Start Migration Button -->
              <button
                class="ui-button ui-button-primary w-full justify-center font-semibold px-6 py-3 disabled:cursor-not-allowed"
                disabled={!isUnlocked || migrating}
                onclick={migrateNotes}
              >
                <RefreshCw class="w-5 h-5" />
                {isUnlocked ? 'Migration starten' : 'Melde dich an, um zu migrieren'}
              </button>
            {/if}
          </div>
        </div>
      {:else if notes.length > 0}
        <!-- All Notes Encrypted -->
        <div class="ui-panel-soft p-5 sm:p-6 bg-success/10 border-success/30">
          <div class="flex items-center gap-3 mb-3">
            <Check class="w-8 h-8 text-success" />
            <h2 class="text-xl font-semibold text-success">Alle Notizen verschlüsselt!</h2>
          </div>
          <p class="text-sm text-success">
            Alle deine {notes.length} Notiz(en) sind bereits verschlüsselt. Es gibt nichts zu migrieren.
          </p>
        </div>
      {:else}
        <!-- No Notes -->
        <div class="ui-panel-soft ui-empty-state py-12">
          <Lock class="w-16 h-16 text-muted-foreground mx-auto mb-4" />
          <p class="text-muted-foreground">Keine Notizen gefunden.</p>
        </div>
      {/if}

      <!-- Important Notes -->
      <div
        class="ui-panel-soft bg-yellow-50 dark:bg-yellow-900/20 p-5 sm:p-6 border-yellow-200 dark:border-yellow-800 mt-6"
      >
        <div class="flex items-start gap-3">
          <AlertTriangle
            class="w-6 h-6 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5"
          />
          <div>
            <h3 class="text-lg font-semibold mb-2 text-yellow-800 dark:text-yellow-300">
              ⚠️ Wichtige Hinweise
            </h3>
            <ul class="space-y-2 text-sm text-yellow-800 dark:text-yellow-300">
              <li>
                • <strong>Recovery Key:</strong> Stelle sicher, dass du einen Recovery-Key erstellt und
                sicher aufbewahrt hast, bevor du die Migration startest!
              </li>
              <li>
                • <strong>Backup:</strong> Erstelle ein Backup deiner Notizen vor der Migration (optional,
                aber empfohlen).
              </li>
              <li>
                • <strong>Irreversibel:</strong> Nach der Migration können die Notizen nur noch mit deinem
                Passwort oder Recovery-Key entschlüsselt werden.
              </li>
              <li>
                • <strong>Geschwindigkeit:</strong> Die Migration kann je nach Anzahl der Notizen mehrere
                Minuten dauern.
              </li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Error Display -->
      {#if error}
        <div
          class="ui-panel-soft mt-6 p-4 bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800"
        >
          <div class="flex items-center gap-2">
            <AlertTriangle class="w-5 h-5 text-red-600 dark:text-red-400" />
            <span class="font-semibold text-red-800 dark:text-red-300">Fehler:</span>
          </div>
          <p class="text-sm text-red-800 dark:text-red-300 mt-2">{error}</p>
        </div>
      {/if}
    {/if}
  </div>
</div>
