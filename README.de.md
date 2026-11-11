# RackPanel

[English](README.md) · **Deutsch**

Eine schlanke Weboberfläche für dein NAS – wie DSM, nur für ganz normales Debian. Ein Programm ohne Datenbank,
läuft auch auf alter Hardware mit 512 MB RAM (ARMv5), genauso auf Raspberry Pi und PCs.

**Funktionen:** Live-Übersicht · Speicher (SMART, RAID, Ruhezustand) · Dateimanager · SMB- und NFS-Freigaben ·
Benutzer und Gruppen · Apps · Netzwerk · Dienste · Updates und Pakete · Protokolle · Zeit, Neustart, Herunterfahren ·
[API für eigene Dashboards](docs/API.de.md)

Die Oberfläche selbst ist auf Englisch.

## Installieren

Als root im Terminal einfügen:

```sh
bash <(curl -fsSL https://raw.githubusercontent.com/Syntax-XXX/rackpanel/main/install.sh)
```

Der Installer fragt nach Benutzername, Passwort und Port, richtet die [RSRB-Paketquelle](https://syntax-xxx.github.io/rsrb/) ein,
installiert das Paket `rackpanel` und zeigt am Ende die Adresse an.
Braucht Debian oder Ubuntu mit systemd (armel, armhf, arm64, amd64).

Ist die RSRB-Paketquelle schon eingerichtet, reicht:

```sh
sudo apt install rackpanel        # fragt nach Benutzer und Passwort
```

Ohne Fragen, z. B. in Skripten:

```sh
RACKPANEL_YES=1 RACKPANEL_USER=admin RACKPANEL_PASSWORD='geheim' bash <(curl -fsSL https://raw.githubusercontent.com/Syntax-XXX/rackpanel/main/install.sh)
```

**Aktualisieren:** `apt upgrade` oder im Panel unter *Updates & Packages*. **Port/Benutzer ändern:** `dpkg-reconfigure rackpanel`.
**Passwort vergessen:** `rackpanel -set-user admin` (liest das neue Passwort von stdin). **Entfernen:** `apt remove rackpanel`.

## Container (RackBox)

`apt install rackbox` bringt die Seite **Containers**: Docker-Images ohne Daemon. RackBox lädt ein Image für deine CPU
direkt aus der Registry (auch ARMv5), entpackt es in einen Ordner und startet jeden Container als eigenen systemd-Dienst
über `systemd-nspawn` – kein Docker-Daemon, kein overlayfs, nichts läuft dauerhaft außer den Containern selbst.
Der App-Store zeigt Images, die es für deine CPU wirklich gibt (auf ARMv5: nginx, Nextcloud, PostgreSQL, Redis, …),
und jedes andere Image lässt sich auf derselben Seite oder per Befehl starten:

```sh
rackbox install nginx                       # App aus dem Katalog
rackbox create -v /volume1/www:/usr/share/nginx/html -web :8080/ web httpd
rackbox start web; rackbox ls; rackbox logs web
```

Die Daten liegen auf dem Daten-Volume (`/volume1/apps/<app>`), die Images in `/volume1/@rackbox`.
Speicherlimits brauchen einen Kernel mit memory-cgroup (auf der RS812: den `rsrb-full`-Kernel).

## Add-ons

RackPanel lässt sich mit Add-ons erweitern – ein Ordner mit `addon.json` und `page.js`, meist in einem Paket.
RackBox ist eines davon. Eigene schreiben: [docs/ADDONS.de.md](docs/ADDONS.de.md).

## Synology RS812 ohne DSM

RackPanel ist für eine Synology RS812 entstanden, auf der statt DSM ein aktuelles Debian läuft.
Wie das geht (Kernel, Bootvorgang, Lüfter, Wiederherstellung): **[rs812-debian](https://github.com/Syntax-XXX/rs812-debian)**.

## Apps

Apps sind normale Debian-Pakete mit Dienst und Weboberfläche (Transmission, MiniDLNA, Mosquitto, FTP).
Der Katalog steht in [`www/apps.json`](www/apps.json) – ein Eintrag pro App, Vorschläge willkommen.
Für Docker-Images siehe *Container* weiter unten.

## Selbst bauen

```sh
./build.sh 1.2.0        # braucht Go 1.22+, legt die Release-Archive in dist/ ab
```

Die Oberfläche in `www/` ist reines HTML/CSS/JS ohne Build-Schritt, der Server in `server/` hat eine einzige Abhängigkeit (bcrypt).
Debian-Pakete und Paketquelle entstehen in [rsrb](https://github.com/Syntax-XXX/rsrb).

## Sicherheit

RackPanel läuft als root und ist für das eigene Netz gedacht. Nicht ungeschützt ins Internet stellen – dafür ein VPN
(z. B. Tailscale oder WireGuard) davor setzen.

## Lizenz

MIT
