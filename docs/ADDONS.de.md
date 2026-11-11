# RackPanel-Add-ons schreiben

[English](ADDONS.md) · **Deutsch**

Ein Add-on ist ein Ordner `/usr/share/rackpanel/addons/<id>/` – meist in einem Debian-Paket. RackPanel findet es beim
nächsten Laden der Seite und nimmt es in die Navigation auf. Kein Neustart, kein Build-Schritt.

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
  "description": "Steht unter dem Seitentitel",
  "page": "page.js",
  "exec": ["/usr/bin/hello-cli"],
  "read": ["status"],
  "write": ["restart"],
  "jobs": ["update"]
}
```

| Feld | Bedeutung |
|---|---|
| `id` | gleich dem Ordnernamen: `a-z`, `0-9`, `-` |
| `icon` | eines der RackPanel-Icons (`box`, `grid`, `disk`, `net`, `layers`, `users`, `share`, …) |
| `exec` | optionales Programm (plus feste Argumente), das das Add-on als root aufrufen darf |
| `read` | erlaubte erste Argumente zum Lesen – auch mit Nur-Lese-API-Schlüsseln |
| `write` | erste Argumente für Änderungen (angemeldete Benutzer und Schlüssel mit vollem Zugriff) |
| `jobs` | erste Argumente für lange Aufgaben: laufen im Hintergrund mit Live-Ausgabe |

Das Add-on kann **nur** `exec` mit einem ersten Argument aus diesen Listen aufrufen – sonst nichts.

## page.js

```js
RackPanel.addon('hello', async (view, rp) => {
  const status = await rp.read('status');          // GET  /api/addons/hello/run?a=status
  view.innerHTML = `<div class="card glass"><h2>Status</h2><pre class="code">${rp.esc(status)}</pre>
    <button class="btn primary" data-up>${rp.ic('download')}Update</button></div>`;
  view.querySelector('[data-up]').onclick = () =>
    rp.job('Update', 'update').then(() => rp.reload()); // Hintergrund-Job mit Live-Ausgabe
  return () => {};                                     // optional: Aufräumen beim Verlassen der Seite
});
```

`rp` enthält die Helfer von RackPanel, damit Add-ons nativ aussehen:

| Helfer | |
|---|---|
| `rp.read(...args)` / `rp.run(...args)` | Add-on-Programm aufrufen, liefert die Ausgabe |
| `rp.job(titel, ...args)` | `jobs`-Befehl mit Live-Ausgabe; liefert `true` bei Erfolg |
| `rp.api(methode, pfad, body)` | beliebiger RackPanel-API-Aufruf |
| `rp.toast(text, 'ok'\|'bad')`, `rp.fail(fehler)` | Meldungen |
| `rp.modal({title, body, buttons})`, `rp.confirmDlg(text, label)`, `rp.promptDlg(titel, label)` | Dialoge |
| `rp.esc(text)`, `rp.ic(name)`, `rp.fmtBytes(n)`, `rp.fmtDate(t)` | Formatierung |
| `rp.setSub(text)`, `rp.reload()`, `rp.host()` | Untertitel, neu zeichnen, Geräteinfos |

Nutze die CSS-Klassen des Panels: `grid` mit `card glass c4/c6/c12`, `rows`/`row`, `btn` (`primary`, `danger`, `sm`),
`badge`, `field`/`input`, `code`. Das [RackBox-Add-on](../rackbox/addon/page.js) ist ein vollständiges Beispiel.

## Sicherheit

Add-ons werden von root (per apt) installiert und laufen im Panel mit den Rechten des angemeldeten Benutzers. Nur
Add-ons installieren, denen du vertraust – wie bei jedem anderen Paket.
