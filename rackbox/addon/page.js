/* RackBox add-on for RackPanel – containers, app store, images */
'use strict';
RackPanel.addon('rackbox', async (v, rb) => {
  const { esc, ic } = rb;
  const json = (p) => p.then((t) => JSON.parse(t || '[]'));
  const [cts, apps, imgs] = await Promise.all([json(rb.read('ls', '--json')), json(rb.read('apps', '--json')), json(rb.read('images', '--json'))]);
  const running = cts.filter((c) => c.state === 'running').length;
  rb.setSub(`${running} of ${cts.length} containers running · Docker images without a daemon`);
  const used = new Set(cts.map((c) => c.app || c.name));
  const url = (c) => `http://${location.hostname}${c.web}`;
  const dot = (s) => (s === 'running' ? '' : s === 'failed' ? 'bad' : 'off');
  const badge = (s) => `<span class="badge ${s === 'running' ? '' : s === 'failed' ? 'bad' : 'off'}">${esc(s)}</span>`;

  v.innerHTML = `<div class="grid">
    <div class="card glass c12"><h2>${ic('box')}Containers<span class="grow"></span><button class="btn sm" data-new>${ic('plus')}Custom image</button></h2>
      <div class="rows">${cts.map((c, i) => `<div class="row"><span class="dot ${dot(c.state)}"></span><div class="main-col">
        <div class="t">${esc(c.name)} <span class="faint" style="font-weight:400">· ${esc(c.image)}</span></div>
        <div class="s">${c.memory_used ? 'RAM ' + esc(c.memory_used) + ' · ' : ''}${c.memory && c.memory_used ? 'limit ' + esc(c.memory) + ' · ' : ''}${(c.binds || []).map((b) => esc(b.split(':')[0])).join(', ') || 'no folders'}</div></div>
        <div class="acts">${badge(c.state)}
          ${c.web && c.state === 'running' ? `<a class="btn sm primary" href="${esc(url(c))}" target="_blank" rel="noopener">${ic('link')}Open</a>` : ''}
          ${c.state === 'running' ? `<button class="btn sm" data-a="stop" data-i="${i}">${ic('stop')}Stop</button>` : `<button class="btn sm" data-a="start" data-i="${i}">${ic('play')}Start</button>`}
          <button class="btn sm" data-a="restart" data-i="${i}">${ic('rotate')}Restart</button>
          <button class="btn sm" data-log="${i}">${ic('logs')}Logs</button>
          <button class="btn sm danger" data-rm="${i}" aria-label="Remove ${esc(c.name)}">${ic('trash')}</button></div></div>`).join('') || '<div class="empty">No containers yet – install one from the app store below.</div>'}</div></div>
    <div class="card glass c12"><h2>${ic('grid')}App store</h2><p class="muted" style="margin:0">Official Docker images that exist for this machine's CPU. Each app runs as its own systemd service – nothing else stays in memory.</p></div>
    ${apps.map((a, i) => `<div class="card glass c6"><div class="row" style="border:0;padding:0;background:none">
      <span class="ico">${ic(a.icon || 'box')}</span><div class="main-col"><div class="t">${esc(a.name)}</div><div class="s">${esc(a.image)}${a.memory ? ' · up to ' + esc(a.memory) : ''}</div></div>
      ${used.has(a.id) ? '<span class="badge">Installed</span>' : `<button class="btn sm primary" data-inst="${i}">${ic('download')}Install</button>`}</div>
      <p class="muted">${esc(a.desc)}</p></div>`).join('')}
    <div class="card glass c12"><h2>${ic('archive')}Images</h2><div class="rows">${imgs.map((m, i) => `<div class="row"><span class="ico">${ic('archive')}</span><div class="main-col">
      <div class="t">${esc(m.ref)}</div><div class="s">${esc(m.platform)} · ${rb.fmtBytes(m.size)} download · ${rb.fmtDate(m.pulled)}</div></div>
      <button class="btn sm danger" data-rmi="${i}">${ic('trash')}Remove</button></div>`).join('') || '<div class="empty">No images yet.</div>'}</div></div></div>`;

  const done = () => rb.reload();
  v.querySelectorAll('[data-a]').forEach((b) => (b.onclick = () => {
    b.disabled = true;
    rb.run(b.dataset.a, cts[+b.dataset.i].name).then(() => setTimeout(done, 800)).catch((e) => { rb.fail(e); b.disabled = false; });
  }));
  v.querySelectorAll('[data-log]').forEach((b) => (b.onclick = async () => {
    const t = await rb.read('logs', cts[+b.dataset.log].name, '-n', '300').catch(rb.fail);
    if (t == null) return;
    const m = rb.modal({ title: 'Logs · ' + cts[+b.dataset.log].name, wide: true, body: `<pre class="code">${esc(t || 'No output yet')}</pre>` });
    const pre = m.el.querySelector('pre'); pre.scrollTop = pre.scrollHeight;
  }));
  v.querySelectorAll('[data-rm]').forEach((b) => (b.onclick = async () => {
    const c = cts[+b.dataset.rm];
    if (await rb.confirmDlg(`Remove container ${c.name}? Its data folders (${(c.binds || []).map((x) => x.split(':')[0]).join(', ') || 'none'}) are kept.`, 'Remove'))
      rb.run('rm', c.name).then(() => { rb.toast('Removed', 'ok'); done(); }).catch(rb.fail);
  }));
  v.querySelectorAll('[data-rmi]').forEach((b) => (b.onclick = async () => {
    const m = imgs[+b.dataset.rmi];
    if (await rb.confirmDlg(`Remove image ${m.ref}?`, 'Remove')) rb.run('rmi', m.ref).then(() => { rb.toast('Removed', 'ok'); done(); }).catch(rb.fail);
  }));
  v.querySelectorAll('[data-inst]').forEach((b) => (b.onclick = () => {
    const a = apps[+b.dataset.inst];
    const go = (kv = []) => rb.job('Install ' + a.name, 'install', a.id, ...kv).then(done);
    if (!a.ask?.length) {
      rb.confirmDlg(`Install ${a.name}? The image is downloaded for this CPU${a.memory ? ', memory limit ' + a.memory : ''}.`, 'Install', 'primary').then((y) => y && go());
      return;
    }
    rb.modal({ title: 'Install ' + a.name, body: a.ask.map((q, k) => `<div class="field"><label for="q${k}">${esc(q.label)}</label><input class="input" id="q${k}" type="${q.secret ? 'password' : 'text'}" autocomplete="new-password"></div>`).join(''),
      buttons: [{ label: 'Cancel' }, { label: 'Install', cls: 'primary', onClick: (m) => {
        const kv = a.ask.map((q, k) => [q.key, m.querySelector('#q' + k).value]);
        if (kv.some(([, val]) => !val)) throw new Error('Please fill in all fields');
        go(kv.map(([k, val]) => k + '=' + val));
      } }] });
  }));
  v.querySelector('[data-new]').onclick = () => rb.modal({ title: 'Run a custom image', wide: true, body: `
    <div class="field"><label for="cn">Name</label><input class="input" id="cn" placeholder="e.g. myapp"></div>
    <div class="field"><label for="ci">Image</label><input class="input" id="ci" placeholder="e.g. httpd or ghcr.io/user/app:tag"></div>
    <div class="field"><label for="ce">Environment (one KEY=VALUE per line)</label><textarea class="input code" id="ce" style="min-height:70px"></textarea></div>
    <div class="field"><label for="cb">Folders (one /host/path:/container/path[:ro] per line)</label><textarea class="input code" id="cb" style="min-height:70px"></textarea></div>
    <div class="field"><label for="cw">Web address (optional, e.g. :8090/)</label><input class="input" id="cw"></div>
    <p class="faint">Containers share the NAS network: the app listens on its own port. Only images built for this CPU work.</p>`,
    buttons: [{ label: 'Cancel' }, { label: 'Create & start', cls: 'primary', onClick: (m) => {
      const val = (id) => m.querySelector(id).value.trim();
      const name = val('#cn'), image = val('#ci');
      if (!/^[a-z0-9][a-z0-9-]{0,30}$/.test(name)) throw new Error('Name: lowercase letters, digits and -');
      if (!image) throw new Error('Image missing');
      const args = ['create'];
      val('#ce').split('\n').map((x) => x.trim()).filter(Boolean).forEach((e) => args.push('-e', e));
      val('#cb').split('\n').map((x) => x.trim()).filter(Boolean).forEach((b) => args.push('-v', b));
      if (val('#cw')) args.push('--web', val('#cw'));
      args.push(name, image);
      rb.job('Create ' + name, ...args).then((okd) => okd && rb.run('start', name)).then(done).catch(rb.fail);
    } }] });
});
