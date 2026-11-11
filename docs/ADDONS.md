# Writing RackPanel add-ons

**English** · [Deutsch](ADDONS.de.md)

An add-on is a folder `/usr/share/rackpanel/addons/<id>/` – usually shipped in a Debian package. RackPanel finds it on
the next page load and adds it to the navigation. No restart, no build step.

```
/usr/share/rackpanel/addons/hello/
├── addon.json
└── page.js
```

## addon.json

```json
{
  "id": "hello",
  "title": "Hello",
  "icon": "box",
  "version": "1.0.0",
  "description": "Shown under the page title",
  "page": "page.js",
  "exec": ["/usr/bin/hello-cli"],
  "read": ["status"],
  "write": ["restart"],
  "jobs": ["update"]
}
```

| Field | Meaning |
|---|---|
| `id` | same as the folder name: `a-z`, `0-9`, `-` |
| `icon` | one of RackPanel's icons (`box`, `grid`, `disk`, `net`, `layers`, `users`, `share`, …) |
| `exec` | optional program (plus fixed arguments) the add-on may run as root |
| `read` | first arguments allowed for reading – also with read-only API keys |
| `write` | first arguments for changes (signed-in users and full-access keys) |
| `jobs` | first arguments for long tasks: they run in the background with a live output window |

The add-on can **only** run `exec` with a first argument from these lists – nothing else.

## page.js

```js
RackPanel.addon('hello', async (view, rp) => {
  const status = await rp.read('status');          // GET  /api/addons/hello/run?a=status
  view.innerHTML = `<div class="card glass"><h2>Status</h2><pre class="code">${rp.esc(status)}</pre>
    <button class="btn primary" data-up>${rp.ic('download')}Update</button></div>`;
  view.querySelector('[data-up]').onclick = () =>
    rp.job('Update', 'update').then(() => rp.reload()); // background job with live output
  return () => {};                                     // optional cleanup when the page is left
});
```

`rp` gives you RackPanel's own helpers, so add-ons look native:

| Helper | |
|---|---|
| `rp.read(...args)` / `rp.run(...args)` | run the add-on program, returns its output |
| `rp.job(title, ...args)` | run a `jobs` command with a live output window; resolves to `true` on success |
| `rp.api(method, path, body)` | any RackPanel API call |
| `rp.toast(msg, 'ok'\|'bad')`, `rp.fail(err)` | notifications |
| `rp.modal({title, body, buttons})`, `rp.confirmDlg(text, label)`, `rp.promptDlg(title, label)` | dialogs |
| `rp.esc(text)`, `rp.ic(name)`, `rp.fmtBytes(n)`, `rp.fmtDate(t)` | formatting |
| `rp.setSub(text)`, `rp.reload()`, `rp.host()` | page subtitle, re-render, host info |

Use the CSS classes of the panel: `grid` with `card glass c4/c6/c12`, `rows`/`row`, `btn` (`primary`, `danger`, `sm`),
`badge`, `field`/`input`, `code`. The [RackBox add-on](../rackbox/addon/page.js) is a complete example.

## Security

Add-ons are installed by root (via apt) and run inside the panel with the signed-in user's rights. Install only add-ons
you trust – just like any other package.
