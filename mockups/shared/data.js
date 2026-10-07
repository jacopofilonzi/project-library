// Dati mock condivisi da tutti i mockup: albero reale di ~/Development
// (incluso il caso annidato local/UNI/Ingegneria del Software/…) e piccoli helper
// per navigazione, ricerca locale/globale e azioni finte.
(function () {
  const p = (name, lang, desc) => ({ type: 'project', name, lang, desc: desc || '' });
  const d = (name, children) => ({ type: 'dir', name, children });

  const TREE = d('Development', [
    d('github', [
      d('curishi', [
        p('play.evons.gg', 'Node', 'Web game client for the Evons CCG.'),
      ]),
      d('jacopofilonzi', [
        p('deploy', 'Go'),
        p('discord-bot-java', 'Java'),
        p('dity', 'Node'),
        p('gdr-metodologie_di_programmazione-2026', 'Docs', "Repository per il codice di un gioco di ruolo per l'esame di Metodologie di Programmazione, A.S. 2025-2026"),
        p('homelab-rice', 'Altro'),
        p('KuberLAB', 'Altro'),
        p('NtfyJS', 'Node', 'JavaScript/TypeScript client for Ntfy.sh'),
        p('ovh-vps-stocknotifier', 'Node', 'Watches OVHcloud VPS stock and notifies you when a plan becomes available.'),
        p('ShellyPlot', 'Docs'),
        p('TimeTable', 'Docs', "Subscribe to your university's lesson timetable from any iCal-compatible calendar (Google, Apple, Outlook, …)."),
      ]),
    ]),
    d('local', [
      p('awake', 'Node', 'Self-hosted Wake-on-LAN over the internet. Keep power-hungry machines off, and wake them from anywhere with one tap.'),
      p('discord-bot-java', 'Java'),
      p('dity-bot-rs', 'Rust'),
      p('dity-bot-rs-old', 'Rust'),
      p('dity-java', 'Java'),
      d('Nuova cartella', []),
      p('project-library', 'Go', 'Launcher per i progetti in ~/Development.'),
      d('space-engineers-1-server', []), // vuota su disco: il caso "progetto non ancora inizializzato"
      p('timetable-worker', 'Rust'),
      d('UNI', [
        d('Ingegneria del Software', [
          p('BuildPatternDemo', 'Java'),
        ]),
      ]),
    ]),
  ]);

  const LANG = {
    Node: '#6cc24a', Rust: '#e3a27a', Go: '#29beb0', Java: '#e76f00', Docs: '#9aa0a6', Altro: '#a78bfa',
  };

  const LAUNCHERS = [
    { id: 'vscode', label: 'VS Code', short: 'Code', key: '↵' },
    { id: 'intellij', label: 'IntelliJ', short: 'IDEA', key: 'Ctrl 2' },
    { id: 'explorer', label: 'Esplora file', short: 'Dir', key: 'Ctrl E' },
  ];

  function get(path) {
    let node = TREE;
    for (const seg of path) {
      node = (node.children || []).find((c) => c.type === 'dir' && c.name === seg);
      if (!node) return TREE;
    }
    return node;
  }

  function sorted(node) {
    return [...(node.children || [])].sort((a, b) =>
      a.type === b.type ? a.name.localeCompare(b.name) : a.type === 'dir' ? -1 : 1);
  }

  function count(node) {
    if (node.type === 'project') return 1;
    return (node.children || []).reduce((n, c) => n + count(c), 0);
  }

  // Tutti i progetti sotto `path`, ciascuno col suo percorso relativo alla root.
  function walk(path) {
    const out = [];
    (function rec(node, at) {
      for (const c of node.children || []) {
        if (c.type === 'project') out.push({ project: c, path: at });
        else rec(c, [...at, c.name]);
      }
    })(get(path), path);
    return out;
  }

  function match(entry, q) {
    const hay = (entry.project.name + ' ' + entry.project.desc + ' ' + entry.path.join('/')).toLowerCase();
    return q.toLowerCase().split(/\s+/).filter(Boolean).every((t) => hay.includes(t));
  }

  // Ricerca contestuale: risultati nella cartella corrente + quelli fuori.
  function search(q, path) {
    const here = walk(path).filter((e) => match(e, q));
    const key = (e) => [...e.path, e.project.name].join('/');
    const hereKeys = new Set(here.map(key));
    const elsewhere = walk([]).filter((e) => match(e, q) && !hereKeys.has(key(e)));
    return { here, elsewhere };
  }

  function rel(entryPath, from) {
    return entryPath.slice(from.length).join('/');
  }

  function toast(msg) {
    let el = document.getElementById('pl-toast');
    if (!el) {
      el = document.createElement('div');
      el.id = 'pl-toast';
      el.setAttribute('role', 'status');
      el.style.cssText = 'position:fixed;left:50%;bottom:24px;transform:translateX(-50%);background:#111;color:#fff;font:13px/1.3 system-ui,sans-serif;padding:10px 16px;border-radius:8px;box-shadow:0 8px 24px rgba(0,0,0,.3);opacity:0;transition:opacity .15s;pointer-events:none;z-index:9999';
      document.body.appendChild(el);
    }
    el.textContent = msg;
    el.style.opacity = '1';
    clearTimeout(el._t);
    el._t = setTimeout(() => (el.style.opacity = '0'), 1600);
  }

  function launch(launcherId, project) {
    const l = LAUNCHERS.find((x) => x.id === launcherId);
    toast(`${l.label} → ${project.name}`);
  }

  const esc = (s) => String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

  function highlight(text, q) {
    const safe = esc(text);
    const terms = q.trim().split(/\s+/).filter(Boolean).map((t) => t.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'));
    if (!terms.length) return safe;
    return safe.replace(new RegExp('(' + terms.join('|') + ')', 'gi'), '<mark>$1</mark>');
  }

  window.PL = { TREE, LANG, LAUNCHERS, get, sorted, count, walk, search, rel, launch, toast, esc, highlight };
})();
