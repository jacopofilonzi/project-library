<script lang="ts">
  import vscode from '../assets/vscode.svg'
  import intellij from '../assets/intellij.svg'
  import { initials } from '../lib/icons'
  import type { Launcher } from '../lib/api'

  let { launcher }: { launcher: Launcher } = $props()
  const logos: Record<string, string> = { vscode, intellij }
  let logo = $derived(logos[launcher.builtin ?? ''])
</script>

{#if logo}
  <img class="lico" src={logo} alt="" />
{:else}
  <span class="lico" style="background:#5f5f68">{initials(launcher.name)}</span>
{/if}
