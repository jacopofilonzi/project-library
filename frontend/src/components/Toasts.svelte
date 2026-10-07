<script lang="ts">
  import { store } from '../lib/state.svelte'
</script>

<div class="toasts" role="status" aria-live="polite">
  {#each store.toasts as tt (tt.id)}
    <div class="toast" class:err={tt.err}>{tt.text}</div>
  {/each}
</div>
