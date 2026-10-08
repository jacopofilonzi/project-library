<script lang="ts">
  // Finestra della ricerca flottante: solo la palette, su sfondo trasparente.
  // L'altezza della finestra segue quella del pannello.
  import { onMount } from 'svelte'
  import { Window } from '@wailsio/runtime'
  import { store } from '../lib/state.svelte'
  import Palette from './Palette.svelte'

  let wrap: HTMLDivElement | undefined = $state()

  onMount(() => {
    document.documentElement.classList.add('spotlight')
    store.init()
  })

  $effect(() => {
    if (!wrap) return
    let last = 0
    const ro = new ResizeObserver(() => {
      const h = Math.ceil(wrap!.getBoundingClientRect().height) + 24 // margine per l'ombra
      if (h !== last) {
        last = h
        Window.SetSize(720, h)
      }
    })
    ro.observe(wrap)
    return () => ro.disconnect()
  })
</script>

<div class="spot" bind:this={wrap}>
  {#if store.ready}<Palette spotlight />{/if}
</div>
