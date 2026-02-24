<script lang="ts">
  import { _ } from 'svelte-i18n';

  import BaseDialog from '$lib/components/ui/BaseDialog.svelte';
  import DialogActions from '$lib/components/ui/DialogActions.svelte';
  import DialogField from '$lib/components/ui/DialogField.svelte';
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
      <DialogField forId="feedback-description" label={$_('feedback.description_label')}>
        <textarea
          id="feedback-description"
          bind:value={description}
          placeholder={$_('feedback.description_placeholder')}
          rows="4"
          class="ui-textarea resize-y"
        ></textarea>
      </DialogField>

      <DialogField forId="feedback-steps" label={$_('feedback.steps_label')}>
        <textarea
          id="feedback-steps"
          bind:value={steps}
          placeholder={$_('feedback.steps_placeholder')}
          rows="3"
          class="ui-textarea resize-y"
        ></textarea>
      </DialogField>

      <div class="ui-panel-soft text-xs text-muted-foreground p-3">
        {$_('feedback.privacy_notice')}
      </div>

      {#if errorMessage}
        <div class="ui-alert ui-alert-danger text-sm">
          {errorMessage}
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet footer()}
    <DialogActions>
      <button type="button" onclick={onClose} class="ui-button ui-button-secondary text-sm">
        {$_('feedback.cancel')}
      </button>
      <button
        type="button"
        onclick={handleSubmit}
        disabled={isSending}
        class="ui-button ui-button-primary text-sm"
      >
        {isSending ? $_('feedback.sending') : $_('feedback.submit')}
      </button>
    </DialogActions>
  {/snippet}
</BaseDialog>
