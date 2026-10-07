// Info git reali raccolte il 2026-10-07 (branch, remote, ultimo commit,
// file modificati, behind/ahead rispetto all'upstream). Chiave = path relativo.
// Assente = la cartella non è un repository git.
window.GIT = {
  'github/curishi/play.evons.gg': { branch: 'main', remote: 'git@github.com:curishi/play.evons.gg.git', commit: { hash: '02c0e48', msg: 'first commit', author: 'RogerBanana', ago: '3 mesi fa' }, dirty: 0, behind: 0, ahead: 0 },
  'github/jacopofilonzi/KuberLAB': { branch: 'main', remote: 'git@github.com:jacopofilonzi/KuberLAB.git', commit: null, dirty: 4, behind: null, ahead: null },
  'github/jacopofilonzi/NtfyJS': { branch: 'main', remote: 'https://github.com/jacopofilonzi/NtfyJS', commit: { hash: '5428c14', msg: 'Blanked gitignore', author: 'Filonzi Jacopo', ago: '7 settimane fa' }, dirty: 7, behind: 0, ahead: 1 },
  'github/jacopofilonzi/ShellyPlot': { branch: 'main', remote: 'git@github.com:jacopofilonzi/ShellyPlot.git', commit: { hash: 'a0effd7', msg: 'Fix typos in filenames in manifest.json', author: 'Filonzi Jacopo', ago: '5 mesi fa' }, dirty: 0, behind: 0, ahead: 0 },
  'github/jacopofilonzi/TimeTable': { branch: 'main', remote: 'git@github.com:jacopofilonzi/TimeTable.git', commit: { hash: 'af8c550', msg: 'Feat: usage tracking, Prometheus metrics and Grafana dashboard', author: 'Filonzi Jacopo', ago: '8 giorni fa' }, dirty: 0, behind: 0, ahead: 0 },
  'github/jacopofilonzi/deploy': { branch: 'main', remote: 'git@github.com:jacopofilonzi/deploy.git', commit: { hash: '00f99f4', msg: 'fix: use goreleaser v2 in github actions', author: 'Filonzi Jacopo', ago: '2 mesi fa' }, dirty: 19, behind: 1, ahead: 0 },
  'github/jacopofilonzi/discord-bot-java': { branch: 'main', remote: 'git@github.com:dity-dev/discord-bot-java.git', commit: { hash: 'bc73ac4', msg: 'Add .env to .gitignore', author: 'Filonzi Jacopo', ago: '5 mesi fa' }, dirty: 6, behind: 0, ahead: 0 },
  'github/jacopofilonzi/dity': { branch: 'main', remote: 'git@github.com:jacopofilonzi/dity.git', commit: { hash: '4f0e49e', msg: 'feat: replace ESLint configuration with TypeScript version and update dependencies', author: 'Filonzi Jacopo', ago: '7 mesi fa' }, dirty: 4, behind: 0, ahead: 0 },
  'github/jacopofilonzi/gdr-metodologie_di_programmazione-2026': { branch: 'main', remote: 'git@github.com:jacopofilonzi/gdr-metodologie_di_programmazione-2026.git', commit: { hash: '336c8b6', msg: 'Added mise environment configuration', author: 'Filonzi Jacopo', ago: '7 mesi fa' }, dirty: 0, behind: 0, ahead: 0 },
  'github/jacopofilonzi/homelab-rice': { branch: 'main', remote: 'git@github.com:jacopofilonzi/homelab-rice.git', commit: { hash: '907182c', msg: "Aggiungi configurazioni di Traefik e file di esempio per l'integrazione con Cloudflare", author: 'Filonzi Jacopo', ago: '8 settimane fa' }, dirty: 0, behind: 0, ahead: 0 },
  'github/jacopofilonzi/ovh-vps-stocknotifier': { branch: 'main', remote: 'git@github.com:jacopofilonzi/ovh-vps-stocknotifier.git', commit: { hash: '5b3b68a', msg: 'Run the OVH API smoke test workflow only on demand', author: 'Filonzi Jacopo', ago: '64 minuti fa' }, dirty: 0, behind: 0, ahead: 0 },
  'local/awake': { branch: 'master', remote: null, commit: { hash: '06e157a', msg: 'Initial Awake implementation: server, web UI, agent protocol and Node agent', author: 'Filonzi Jacopo', ago: '11 giorni fa' }, dirty: 0, behind: null, ahead: null },
  'local/dity-bot-rs': { branch: 'main', remote: null, commit: null, dirty: 5, behind: null, ahead: null },
  'local/dity-bot-rs-old': { branch: 'master', remote: null, commit: null, dirty: 5, behind: null, ahead: null },
  'local/timetable-worker': { branch: 'main', remote: null, commit: null, dirty: 6, behind: null, ahead: null },
};

// README disponibili (estratto delle prime 60 righe in shared/readme/).
window.READMES = new Set([
  'local/awake', 'github/jacopofilonzi/NtfyJS', 'github/jacopofilonzi/TimeTable', 'github/jacopofilonzi/ovh-vps-stocknotifier',
  'github/curishi/play.evons.gg', 'github/jacopofilonzi/deploy', 'github/jacopofilonzi/ShellyPlot',
  'github/jacopofilonzi/gdr-metodologie_di_programmazione-2026',
]);
