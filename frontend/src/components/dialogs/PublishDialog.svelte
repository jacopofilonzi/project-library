<script lang="ts">
  import { untrack } from 'svelte'
  import { store } from '../../lib/state.svelte'
  import { t, tn } from '../../lib/i18n/index.svelte'
  import { lib, errMessage, serviceName, type Node, type Owner, type GitInfo, type PublishRequest } from '../../lib/api'

  // crea il repository su GitHub o GitLab per un progetto git senza remote e fa il primo push
  let { node }: { node: Node } = $props()
  const n = untrack(() => node)

  const accounts = store.forgeAccounts
  let accIdx = $state(0)
  let acc = $derived(accounts[accIdx])
  let owners = $state<Owner[]>([])
  let ownerIdx = $state(0)

  // nome valido per GitHub e GitLab: lettere, numeri, punti, trattini e underscore
  const cleanName = (s: string) => s.trim().replace(/[^A-Za-z0-9._-]+/g, '-').replace(/-{2,}/g, '-').replace(/^[-.]+|-+$/g, '')
  let name = $state(cleanName(n.name))
  let description = $state(n.desc ?? '')
  let priv = $state(true)
  let git = $state<GitInfo | null>(null)
  let busy = $state(false)
  let nameInput: HTMLInputElement | undefined = $state()
  $effect(() => nameInput?.focus())

  lib.GitInfo(n.path).then((g) => (git = g)).catch(() => {})

  $effect(() => {
    const a = acc
    if (!a) return
    owners = [{ name: a.user, personal: true, id: 0 }]
    ownerIdx = 0
    lib.ForgeOwners(a.kind, a.host).then((o) => { if (a === acc && o?.length) owners = o }).catch(() => {})
  })

  let nameOk = $derived(/^[A-Za-z0-9._-]+$/.test(name) && name !== '.' && name !== '..')
  let ok = $derived(!!acc && nameOk && !busy)

  async function run() {
    if (!ok) return
    busy = true
    try {
      const res = await lib.Publish({
        path: n.path, kind: acc.kind as unknown as PublishRequest['kind'], host: acc.host, owner: $state.snapshot(owners[ownerIdx]),
        name, description, private: priv,
      })
      store.dialog = null
      store.gitChanged++
      if (res.pushError) store.toast(t('pub.pushFailed', { err: res.pushError }), true)
      else store.toast(t(res.pushed ? 'pub.done' : 'pub.doneNoPush', { url: res.web }))
    } catch (e) {
      busy = false
      store.toast(errMessage(e), true)
    }
  }
</script>

<h3>{t('pub.title', { name: n.name })}</h3>
<div class="bd">
  <p>{t('pub.text')}</p>
  {#if accounts.length > 1}
    <label>{t('pub.account')}
      <select bind:value={accIdx} disabled={busy}>
        {#each accounts as a, i (a.kind + a.host)}<option value={i}>{serviceName(a.kind)} · {a.user} · {a.host}</option>{/each}
      </select>
    </label>
  {/if}
  <div class="row2">
    <label>{t('pub.owner')}
      <select bind:value={ownerIdx} disabled={busy}>
        {#each owners as o, i (o.name)}<option value={i}>{o.name}</option>{/each}
      </select>
    </label>
    <span class="slash">/</span>
    <label class="grow">{t('pub.name')}
      <input type="text" bind:this={nameInput} bind:value={name} autocomplete="off" spellcheck="false" disabled={busy} onkeydown={(e) => e.key === 'Enter' && run()} />
    </label>
  </div>
  {#if !nameOk}<div class="hint err">{t('pub.nameHint')}</div>{/if}
  <label>{t('pub.desc')}<input type="text" bind:value={description} autocomplete="off" disabled={busy} /></label>
  <div class="vis" role="radiogroup">
    <label class="chk"><input type="radio" name="vis" checked={priv} onchange={() => (priv = true)} disabled={busy} /> {t('pub.private')}</label>
    <label class="chk"><input type="radio" name="vis" checked={!priv} onchange={() => (priv = false)} disabled={busy} /> {t('pub.public')}</label>
  </div>
  {#if git && !git.commit}
    <div class="warn">{t('pub.noCommits')}</div>
  {:else if git?.dirty}
    <div class="hint">{tn('pub.dirty', git.dirty)}</div>
  {/if}
</div>
<div class="ft">
  <button onclick={() => (store.dialog = null)} disabled={busy}>{t('dlg.cancel')}</button>
  <button class="p" disabled={!ok} onclick={run}>{busy ? t('pub.working') : t('pub.go')}</button>
</div>
