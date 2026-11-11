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

## Synology RS812 ohne DSM

RackPanel ist für eine Synology RS812 entstanden, auf der statt DSM ein aktuelles Debian läuft.
Wie das geht (Kernel, Bootvorgang, Lüfter, Wiederherstellung): **[rs812-debian](https://github.com/Syntax-XXX/rs812-debian)**.

## Apps

Apps sind normale Debian-Pakete mit Dienst und Weboberfläche (Transmission, MiniDLNA, Mosquitto, FTP).
Der Katalog steht in [`www/apps.json`](www/apps.json) – ein Eintrag pro App, Vorschläge willkommen.
Docker/Podman gibt es bewusst nicht: Dem RS812-Kernel fehlen Memory-Cgroups und overlayfs, für ARMv5 gibt es
kaum Images, und 512 MB reichen für Container nicht.

## Selbst bauen

```sh
./build.sh 1.1.0        # braucht Go 1.22+, legt die Release-Archive in dist/ ab
```

Die Oberfläche in `www/` ist reines HTML/CSS/JS ohne Build-Schritt, der Server in `server/` hat eine einzige Abhängigkeit (bcrypt).
Debian-Pakete und Paketquelle entstehen in [rsrb](https://github.com/Syntax-XXX/rsrb).

## Sicherheit

RackPanel läuft als root und ist für das eigene Netz gedacht. Nicht ungeschützt ins Internet stellen – dafür ein VPN
(z. B. Tailscale oder WireGuard) davor setzen.

## Lizenz

MIT
