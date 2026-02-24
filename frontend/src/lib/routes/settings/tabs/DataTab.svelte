<script lang="ts">
  import { Download, Loader2, Upload } from 'lucide-svelte';
  import { _ } from 'svelte-i18n';

  import { getExportUrl, importMarkdown } from '$lib/api';
  import {
    handleExport as handleExportHelper,
    handleImportClick as handleImportClickHelper,
    handleImportFiles as handleImportFilesHelper,
  } from '$lib/routes/settings/import-export';
  import * as dialog from '$lib/stores/dialog.svelte';

  let importInput: HTMLInputElement;
  let importing = $state(false);

  function handleExport() {
    handleExportHelper({
      openWindow: (url, target) => window.open(url, target),
      getExportUrl: () => getExportUrl(),
    });
  }

  function handleImportClick() {
    handleImportClickHelper({
      triggerFileDialog: () => importInput.click(),
    });
  }

  async function handleImportFiles(e: Event) {
    await handleImportFilesHelper(e, {
      setImporting: (value) => {
        importing = value;
      },
      importMarkdown: (files, merge) => importMarkdown(files, merge),
      alert: (options) => dialog.alert(options),
      messages: {
        noteTitle: $_('common.note'),
        errorTitle: $_('common.error'),
        noMdSelected: $_('page.settings.data.no_md_files_selected'),
        importCompleted: $_('page.settings.data.import_completed'),
        notesImported: (count) => $_('page.settings.data.notes_imported', { values: { count } }),
        foldersCreated: (count) => $_('page.settings.data.folders_created', { values: { count } }),
        skippedNotes: (count) => $_('page.settings.data.skipped_notes', { values: { count } }),
        failedNotes: (count) => $_('page.settings.data.failed_notes', { values: { count } }),
        errorsLabel: $_('page.settings.data.errors'),
        importFailed: (error) => $_('page.settings.data.import_failed', { values: { error } }),
      },
    });
  }
</script>

<div class="space-y-8">
  <!-- Export -->
  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">
      {$_('page.settings.data.export_title')}
    </h3>
    <p class="text-sm text-muted-foreground mb-4">
      {$_('page.settings.data.export_description')}
    </p>
    <button onclick={handleExport} class="ui-button ui-button-primary">
      <Download size={16} />
      {$_('page.settings.data.export_button')}
    </button>
  </div>

  <!-- Import -->
  <div>
    <h3 class="text-lg font-medium text-foreground mb-2">
      {$_('page.settings.data.import_title')}
    </h3>
    <p class="text-sm text-muted-foreground mb-4">
      {$_('page.settings.data.import_description')}
    </p>
    <button onclick={handleImportClick} disabled={importing} class="ui-button ui-button-secondary">
      {#if importing}
        <Loader2 size={16} class="animate-spin" />
        {$_('page.settings.data.importing')}
      {:else}
        <Upload size={16} />
        {$_('page.settings.data.import_button')}
      {/if}
    </button>
  </div>

  <!-- Info -->
  <div class="p-4 rounded-lg bg-primary/10 border border-primary/30 text-sm text-foreground">
    <strong>{$_('common.note')}</strong>
    {$_('page.settings.data.info_note')}
  </div>
</div>

<!-- Hidden file input for markdown import -->
<input
  type="file"
  accept=".md"
  multiple
  webkitdirectory
  bind:this={importInput}
  onchange={handleImportFiles}
  style="display:none"
/>
