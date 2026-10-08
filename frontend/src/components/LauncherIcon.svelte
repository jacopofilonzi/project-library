<script lang="ts">
  import vscode from '../assets/vscode.svg'
  import intellij from '../assets/intellij.svg'
  import androidstudio from '../assets/androidstudio.svg'
  import webstorm from '../assets/webstorm.svg'
  import pycharm from '../assets/pycharm.svg'
  import goland from '../assets/goland.svg'
  import rider from '../assets/rider.svg'
  import clion from '../assets/clion.svg'
  import phpstorm from '../assets/phpstorm.svg'
  import rubymine from '../assets/rubymine.svg'
  import cursor from '../assets/cursor.svg'
  import zed from '../assets/zed.svg'
  import sublime from '../assets/sublime.svg'
  import { initials } from '../lib/icons'
  import type { Launcher } from '../lib/api'

  let { launcher }: { launcher: Launcher } = $props()
  const logos: Record<string, string> = { vscode, intellij, androidstudio, webstorm, pycharm, goland, rider, clion, phpstorm, rubymine, cursor, zed, sublime }
  // known editors without a logo: a recognizable color instead of the gray of custom launchers
  const colors: Record<string, string> = { rustrover: '#E8590C' }
  let logo = $derived(logos[launcher.builtin ?? ''])
</script>

{#if logo}
  <img class="lico" src={logo} alt="" title={launcher.name} />
{:else}
  <span class="lico" style="background:{colors[launcher.builtin ?? ''] ?? '#5f5f68'}" title={launcher.name}>{initials(launcher.name)}</span>
{/if}
