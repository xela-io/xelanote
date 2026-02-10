<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  interface Props {
    iframeUrl: string;
    onToken: (token: string) => void;
    onExpired?: () => void;
    onError?: () => void;
  }

  const { iframeUrl, onToken, onExpired, onError }: Props = $props();

  let iframeRef = $state<HTMLIFrameElement | null>(null);
  let timeoutId: ReturnType<typeof setTimeout> | null = null;

  // Derive the expected origin from the iframe URL for security validation
  function getExpectedOrigin(url: string): string {
    try {
      const parsed = new URL(url);
      return parsed.origin;
    } catch {
      return '';
    }
  }

  function handleMessage(event: MessageEvent) {
    // Origin validation: only accept messages from the server hosting the CAPTCHA page
    const expectedOrigin = getExpectedOrigin(iframeUrl);
    if (expectedOrigin && event.origin !== expectedOrigin) {
      return;
    }

    if (!event.data || typeof event.data !== 'object') {
      return;
    }

    switch (event.data.type) {
      case 'captcha-token':
        if (timeoutId) {
          clearTimeout(timeoutId);
          timeoutId = null;
        }
        onToken(event.data.token);
        break;
      case 'captcha-expired':
        onExpired?.();
        break;
      case 'captcha-error':
        onError?.();
        break;
    }
  }

  /**
   * Reset the CAPTCHA widget by sending a message to the iframe.
   */
  export function reset() {
    if (iframeRef?.contentWindow) {
      const expectedOrigin = getExpectedOrigin(iframeUrl);
      iframeRef.contentWindow.postMessage({ type: 'captcha-reset' }, expectedOrigin || '*');
    }
  }

  onMount(() => {
    window.addEventListener('message', handleMessage);

    // Timeout: if no token received within 10s, assume iframe failed to load
    timeoutId = setTimeout(() => {
      timeoutId = null;
      onError?.();
    }, 10000);
  });

  onDestroy(() => {
    window.removeEventListener('message', handleMessage);
    if (timeoutId) {
      clearTimeout(timeoutId);
      timeoutId = null;
    }
  });
</script>

<iframe
  bind:this={iframeRef}
  src={iframeUrl}
  sandbox="allow-scripts allow-same-origin allow-popups"
  title="CAPTCHA verification"
  class="captcha-iframe"
></iframe>

<style>
  .captcha-iframe {
    width: 300px;
    height: 65px;
    border: none;
    overflow: hidden;
  }
</style>
