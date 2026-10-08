<script lang="ts">
  import { store } from '../../lib/state.svelte'
  import NameDialog from './NameDialog.svelte'
  import DeleteDialog from './DeleteDialog.svelte'
  import CloneDialog from './CloneDialog.svelte'
  import InitDialog from './InitDialog.svelte'
  import PublishDialog from './PublishDialog.svelte'
  import MoveDialog from './MoveDialog.svelte'

  let d = $derived(store.dialog)
  // a running clone does not close on outside clicks: it must be cancelled with the button
  function backdrop(e: MouseEvent) {
    if (e.target === e.currentTarget && d?.kind !== 'clone' && d?.kind !== 'publish') store.dialog = null
  }
</script>

{#if d}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="backdrop" onmousedown={backdrop}>
    <div class="dlg" role="dialog" aria-modal="true">
      {#key d}
        {#if d.kind === 'newFolder'}
          <NameDialog mode="new" parent={d.parent} />
        {:else if d.kind === 'rename'}
          <NameDialog mode="rename" path={d.path} name={d.name} />
        {:else if d.kind === 'delete'}
          <DeleteDialog node={d.node} />
        {:else if d.kind === 'clone'}
          <CloneDialog parent={d.parent} />
        {:else if d.kind === 'init'}
          <InitDialog path={d.path} name={d.name} />
        {:else if d.kind === 'move'}
          <MoveDialog node={d.node} dest={d.dest} />
        {:else if d.kind === 'publish'}
          <PublishDialog node={d.node} />
        {/if}
      {/key}
    </div>
  </div>
{/if}
