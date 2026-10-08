<script lang="ts">
  import { tick } from 'svelte'
  import { store } from '../lib/state.svelte'

  let el: HTMLDivElement | undefined = $state()
  let pos = $state({ x: 0, y: 0 })

  // keeps the menu inside the window and moves the focus to the first item
  $effect(() => {
    const c = store.ctx
    if (!c) return
    pos = { x: c.x, y: c.y }
    tick().then(() => {
      if (!el) return
      pos = { x: Math.min(c.x, innerWidth - el.offsetWidth - 8), y: Math.min(c.y, innerHeight - el.offsetHeight - 8) }
      el.querySelector('button')?.focus()
    })
  })

  function run(i: number) {
    const it = store.ctx?.items[i]
    store.ctx = null
    if (it && it !== '-') it.run()
  }

  function onkey(e: KeyboardEvent) {
    const btns = [...(el?.querySelectorAll('button') ?? [])]
    const i = btns.indexOf(document.activeElement as HTMLButtonElement)
    if (e.key === 'ArrowDown') { e.preventDefault(); btns[(i + 1) % btns.length]?.focus() }
    if (e.key === 'ArrowUp') { e.preventDefault(); btns[(i - 1 + btns.length) % btns.length]?.focus() }
  }
</script>

<svelte:window onmousedown={(e) => { if (store.ctx && el && !el.contains(e.target as Element)) store.ctx = null }} onblur={() => (store.ctx = null)} />

{#if store.ctx}
  <div class="ctx" role="menu" tabindex="-1" bind:this={el} style="left:{pos.x}px;top:{pos.y}px" onkeydown={onkey}>
    {#each store.ctx.items as it, i}
      {#if it === '-'}
        <hr />
      {:else}
        <button role="menuitem" class:danger={it.danger} onclick={() => run(i)}><span>{it.label}</span>{#if it.key}<small>{it.key}</small>{/if}</button>
      {/if}
    {/each}
  </div>
{/if}
