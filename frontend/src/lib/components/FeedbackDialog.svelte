<script lang="ts">
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import * as errorReporter from '$lib/stores/error-reporter.svelte';
  import * as toast from '$lib/stores/toast.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
  }

  const { open, onClose }: Props = $props();

  let description = $state('');
  let steps = $state('');
  let isSending = $state(false);
  let errorMessage = $state<string | null>(null);

  async function handleSubmit() {
    errorMessage = null;

    if (description.trim().length < 10) {
      errorMessage = $_('feedback.error_too_short');
      return;
    }

    isSending = true;
    try {
      const result = await errorReporter.reportManualFeedback(
        description.trim(),
        steps.trim() || undefined
      );
      if (result.accepted) {
        toast.success($_('feedback.sent_success'));
        description = '';
        steps = '';
        onClose();
      } else {
        errorMessage = $_('feedback.error_send_failed');
      }
    } catch {
      errorMessage = $_('feedback.error_send_failed');
    } finally {
      isSending = false;
    }
  }
</script>

<BaseDialog {open} title={$_('feedback.dialog_title')} {onClose} size="md">
  {#snippet content()}
    <div class="space-y-4">
      <div class="space-y-2">
        <label for="feedback-description" class="text-sm font-medium"
          >{$_('feedback.description_label')}</label
        >
        <textarea
          id="feedback-description"
          bind:value={description}
          placeholder={$_('feedback.description_placeholder')}
          rows="4"
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring resize-y"
        ></textarea>
      </div>

      <div class="space-y-2">
        <label for="feedback-steps" class="text-sm font-medium">{$_('feedback.steps_label')}</label>
        <textarea
          id="feedback-steps"
          bind:value={steps}
          placeholder={$_('feedback.steps_placeholder')}
          rows="3"
          class="w-full px-3 py-2 bg-background border border-border rounded-md focus:outline-none focus:ring-2 focus:ring-ring resize-y"
        ></textarea>
      </div>

      <div class="text-xs text-muted-foreground bg-muted/50 p-3 rounded-md">
        {$_('feedback.privacy_notice')}
      </div>

      {#if errorMessage}
        <div class="text-sm text-red-600 bg-red-50 dark:bg-red-900/20 p-2 rounded">
          {errorMessage}
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <button type="button" onclick={onClose} class="px-4 py-2 text-sm hover:bg-accent rounded-md">
      {$_('feedback.cancel')}
    </button>
    <button
      type="button"
      onclick={handleSubmit}
      disabled={isSending}
      class="px-4 py-2 text-sm bg-primary text-primary-foreground hover:bg-primary/90 rounded-md disabled:opacity-50"
    >
      {isSending ? $_('feedback.sending') : $_('feedback.submit')}
    </button>
  {/snippet}
</BaseDialog>
