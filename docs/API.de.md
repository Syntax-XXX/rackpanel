# RackPanel API

[English](API.md) · **Deutsch**

Alles, was die Oberfläche kann, geht auch per HTTP. Damit lässt sich das NAS in eigene Dashboards einbauen.

## Schlüssel holen

Im Panel: **System → API keys → New key**. „Read only“ reicht für Dashboards und kann nichts verändern.
Der Schlüssel (`rp_…`) wird nur einmal angezeigt.

## Aufruf

Schlüssel als Header `X-API-Key` (oder `Authorization: Bearer rp_…`, oder `?token=rp_…`) mitschicken:

```sh
curl -H "X-API-Key: rp_…" http://nas/api/overview
```

Jede Antwort hat dieselbe Form: `{"success": 200, "message": "ok", "data": …}`. Bei Fehlern steht der Grund in `message`.
CORS ist offen, der Aufruf geht also auch direkt aus dem Browser:

```html
<div id="nas"></div>
<script>
fetch('http://nas/api/overview', { headers: { 'X-API-Key': 'rp_…' } })
  .then((r) => r.json())
  .then(({ data: d }) => {
    const v = d.volumes.map((x) => `${x.path}: ${Math.round((x.used / x.size) * 100)} %`).join(', ');
    document.getElementById('nas').textContent = `${d.hostname} · ${d.cpu_temp} °C · ${v}`;
  });
</script>
```

Ein Nur-Lese-Schlüssel im Browser ist für jeden sichtbar, der die Seite öffnen kann – also nur in Dashboards im eigenen Netz verwenden.

## Die wichtigsten Adressen

| Methode | Pfad | Liefert |
|---|---|---|
| GET | `/api/overview` | Name, Modell, System, Laufzeit, Last, Temperaturen, Lüfter, Laufwerke, Volumes, RAID, Netzwerk, Speicher |
| GET | `/api/stats` | Momentaufnahme: CPU %, Arbeitsspeicher, Netzwerk-Zähler, Platten-I/O-Zähler (`io`: gelesene/geschriebene Bytes pro Platte und RAID-Array), Last (für Live-Anzeigen, z. B. alle 3 s) |
| GET | `/api/storage` | Laufwerke (SMART), RAID-Gruppen, Volumes |
| GET | `/api/storage/smart/{dev}` | SMART-Bericht eines Laufwerks, z. B. `sda` |
| GET | `/api/services` | Dienste mit Zustand |
| GET | `/api/apps` | App-Katalog mit „installiert“ und „läuft“ |
| GET | `/api/files?path=/volume1` | Ordnerinhalt |
| GET | `/api/network` | Adressen, Route, DNS |
| GET | `/api/logs?unit=ssh.service` | Journal |

Mit einem Schlüssel mit vollem Zugriff zusätzlich (Auswahl):

| Methode | Pfad | Wirkung |
|---|---|---|
| POST | `/api/services/{name}/{start\|stop\|restart\|enable\|disable}` | Dienst steuern |
| POST | `/api/apps/{id}/{install\|remove}` | App installieren/entfernen, liefert eine Job-ID |
| GET | `/api/jobs/{id}` | Fortschritt und Ausgabe eines Jobs |
| POST | `/api/storage/{dev}/sleep` | Laufwerk schlafen legen |
| POST | `/api/updates/check`, `/api/updates/install` | Systemupdates |
| POST | `/api/power/{reboot\|poweroff}` | Neustart, Herunterfahren |

Schlüssel verwalten und das Panel-Passwort ändern geht nur mit Anmeldung, nie mit einem Schlüssel.
