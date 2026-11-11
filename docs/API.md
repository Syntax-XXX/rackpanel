# RackPanel API

**English** · [Deutsch](API.de.md)

Everything the web UI does is plain HTTP, so you can put your NAS into your own dashboard.

## Get a key

In the panel: **System → API keys → New key**. "Read only" is enough for dashboards and can't change anything.
The key (`rp_…`) is shown only once.

## Calling it

Send the key as header `X-API-Key` (or `Authorization: Bearer rp_…`, or `?token=rp_…`):

```sh
curl -H "X-API-Key: rp_…" http://nas/api/overview
```

Every response has the same shape: `{"success": 200, "message": "ok", "data": …}`. On errors `message` says why.
CORS is open, so it also works directly from a browser:

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

A read-only key inside a web page is visible to anyone who can open that page – only use it in dashboards on your own network.

## Main endpoints

| Method | Path | Returns |
|---|---|---|
| GET | `/api/overview` | name, model, OS, uptime, load, temperatures, fan, drives, volumes, RAID, network, memory |
| GET | `/api/stats` | snapshot: CPU %, memory, network counters, load (for live views, e.g. every 3 s) |
| GET | `/api/storage` | drives (SMART), RAID arrays, volumes |
| GET | `/api/storage/smart/{dev}` | SMART report of one drive, e.g. `sda` |
| GET | `/api/services` | services and their state |
| GET | `/api/apps` | app catalog with "installed" and "running" |
| GET | `/api/files?path=/volume1` | folder listing |
| GET | `/api/network` | addresses, routes, DNS |
| GET | `/api/logs?unit=ssh.service` | journal |

With a full-access key, additionally (selection):

| Method | Path | Effect |
|---|---|---|
| POST | `/api/services/{name}/{start\|stop\|restart\|enable\|disable}` | control a service |
| POST | `/api/apps/{id}/{install\|remove}` | install/remove an app, returns a job ID |
| GET | `/api/jobs/{id}` | progress and output of a job |
| POST | `/api/storage/{dev}/sleep` | spin a drive down |
| POST | `/api/updates/check`, `/api/updates/install` | system updates |
| POST | `/api/power/{reboot\|poweroff}` | restart, shut down |

Managing keys and changing the panel password require signing in – never possible with a key.
