# RackPanel

Eine schlanke Weboberfläche für dein NAS – wie DSM, nur für ganz normales Debian. Ein einzelnes Programm ohne Datenbank,
läuft auch auf alter Hardware mit 512 MB RAM (ARMv5), genauso auf Raspberry Pi und PCs.

**Was drin ist:** Übersicht mit Live-Werten · Speicher (SMART, RAID, Ruhezustand) · Dateimanager · SMB- und NFS-Freigaben ·
Benutzer und Gruppen · Apps · Netzwerk · Dienste · Updates und Pakete · Protokolle · Zeit, Neustart, Herunterfahren ·
[API für eigene Dashboards](docs/API.md)

## Installieren

Als root im Terminal einfügen:

```sh
bash <(curl -fsSL https://raw.githubusercontent.com/Syntax-XXX/rackpanel/main/install.sh)
```

Der Installer fragt nach Benutzername, Passwort und Port, richtet alles ein und zeigt am Ende die Adresse an.
Braucht Debian oder Ubuntu mit systemd (armv5, armv7, arm64, amd64).

Ohne Fragen, z. B. in Skripten:

```sh
RACKPANEL_YES=1 RACKPANEL_USER=admin RACKPANEL_PASSWORD='geheim' bash <(curl -fsSL https://raw.githubusercontent.com/Syntax-XXX/rackpanel/main/install.sh)
```

Aktualisieren: den Installer noch einmal ausführen. Entfernen: `… install.sh) --uninstall`.

## Synology RS812 ohne DSM

RackPanel ist für eine Synology RS812 entstanden, auf der statt DSM ein aktuelles Debian läuft.
Wie das geht (Kernel, Bootvorgang, Lüfter, Wiederherstellung): **[rs812-debian](https://github.com/Syntax-XXX/rs812-debian)**.

## Apps

Apps sind normale Debian-Pakete mit Dienst und Weboberfläche (Transmission, MiniDLNA, Mosquitto, FTP).
Der Katalog steht in [`www/apps.json`](www/apps.json) – ein Eintrag pro App, Vorschläge willkommen.
Container (Docker/Podman) gibt es bewusst nicht: Für ARMv5 existieren kaum Images und 512 MB sind dafür zu wenig.

## Selbst bauen

```sh
./build.sh 1.0.0        # braucht Go 1.22+, legt die Archive in dist/ ab
```

Die Oberfläche in `www/` ist reines HTML/CSS/JS ohne Build-Schritt, der Server in `server/` hat eine einzige Abhängigkeit (bcrypt).

## Sicherheit

RackPanel läuft als root und ist für das eigene Netz gedacht. Nicht ungeschützt ins Internet stellen – dafür ein VPN
(z. B. Tailscale oder WireGuard) davor setzen.
