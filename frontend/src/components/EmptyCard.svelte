<script lang="ts">
  import { store } from '../lib/state.svelte'
  import { t } from '../lib/i18n/index.svelte'
  import { icons } from '../lib/icons'
  import type { Node } from '../lib/api'
  import OpenWith from './OpenWith.svelte'

  let { node }: { node: Node } = $props()

  async function mark() {
    await store.setOverride(node.path, 'project')
    store.goToPath(node.path)
    store.toast(t('empty.marked', { name: node.name }))
  }
</script>

<section class="detail">
  <div class="in">
    <div class="head">
      <div>
        <div class="path selectable">{node.path}</div>
        <h1>{node.name}</h1>
        <span class="tag">{@html icons.folder()} {t('empty.tag')}</span>
      </div>
      <div class="acts">
        <OpenWith {node} />
        <button class="ibtn" title={t('card.explorer', { key: store.mod + ' E' })} aria-label={t('ctx.reveal')} onclick={() => store.reveal(node.path)}>{@html icons.folder(18)}</button>
      </div>
    </div>
    <div class="emptybox">
      <p>{t('empty.text')}</p>
      <div class="emptyacts">
        <button class="tbtn" onclick={() => (store.dialog = { kind: 'newFolder', parent: node.path })}>{t('col.newFolder')}</button>
        <button class="tbtn" onclick={() => (store.dialog = { kind: 'clone', parent: node.path })}>{t('col.cloneHere')}</button>
        <button class="tbtn" onclick={() => (store.dialog = { kind: 'init', path: node.path, name: node.name })}>{t('init.button')}</button>
        <button class="tbtn" onclick={mark}>{t('empty.markProject')}</button>
      </div>
    </div>
  </div>
</section>
