export interface ImportMarkdownFile {
  path: string;
  filename: string;
  content: string;
}

export interface ImportMarkdownResult {
  imported: number;
  folders_created: number;
  skipped: number;
  failed: number;
  errors?: string[];
}

export interface ImportExportDeps {
  openWindow: (url: string, target?: string) => void;
  getExportUrl: () => string;
  triggerFileDialog: () => void;
  setImporting: (value: boolean) => void;
  importMarkdown: (files: ImportMarkdownFile[], merge: boolean) => Promise<ImportMarkdownResult>;
  alert: (options: {
    title: string;
    message: string;
    variant?: 'warning' | 'default' | 'danger';
  }) => Promise<void>;
  messages: {
    noteTitle: string;
    errorTitle: string;
    noMdSelected: string;
    importCompleted: string;
    notesImported: (count: number) => string;
    foldersCreated: (count: number) => string;
    skippedNotes: (count: number) => string;
    failedNotes: (count: number) => string;
    errorsLabel: string;
    importFailed: (error: string) => string;
  };
}

export function handleExport(deps: Pick<ImportExportDeps, 'openWindow' | 'getExportUrl'>) {
  deps.openWindow(deps.getExportUrl(), '_blank');
}

export function handleImportClick(deps: Pick<ImportExportDeps, 'triggerFileDialog'>) {
  deps.triggerFileDialog();
}

export async function handleImportFiles(
  e: Event,
  deps: Omit<ImportExportDeps, 'getExportUrl' | 'openWindow' | 'triggerFileDialog'>
) {
  const input = e.target as HTMLInputElement;
  const files = Array.from(input.files || []);
  const mdFiles = files.filter((file) => file.name.endsWith('.md'));

  if (mdFiles.length === 0) {
    await deps.alert({
      title: deps.messages.noteTitle,
      message: deps.messages.noMdSelected,
      variant: 'warning',
    });
    return;
  }

  deps.setImporting(true);

  try {
    const importFiles = await Promise.all(
      mdFiles.map(async (file) => ({
        path: (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name,
        filename: file.name,
        content: await file.text(),
      }))
    );

    const result = await deps.importMarkdown(importFiles, true);

    let message = `${deps.messages.importCompleted}\n\n`;
    message += `${deps.messages.notesImported(result.imported)}\n`;
    message += `${deps.messages.foldersCreated(result.folders_created)}\n`;

    if (result.skipped > 0) {
      message += `${deps.messages.skippedNotes(result.skipped)}\n`;
    }

    if (result.failed > 0) {
      message += `${deps.messages.failedNotes(result.failed)}\n`;
      if (result.errors) {
        message += `\n${deps.messages.errorsLabel}:\n${result.errors.slice(0, 5).join('\n')}`;
      }
    }

    await deps.alert({
      title: deps.messages.importCompleted,
      message,
      variant: result.failed > 0 ? 'warning' : 'default',
    });
  } catch (err: unknown) {
    console.error('Import failed:', err);
    await deps.alert({
      title: deps.messages.errorTitle,
      message: deps.messages.importFailed(err instanceof Error ? err.message : String(err)),
      variant: 'danger',
    });
  } finally {
    deps.setImporting(false);
    input.value = '';
  }
}
