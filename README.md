# RackPanel

**English** · [Deutsch](README.de.md)

A lightweight web UI for your NAS – like DSM, but for plain Debian. One program, no database,
runs even on old hardware with 512 MB RAM (ARMv5), just as well on a Raspberry Pi or a PC.

**Features:** live overview · storage (SMART, RAID, disk standby) · file manager · SMB and NFS shares ·
users and groups · apps · network · services · updates and packages · logs · time, restart, shutdown ·
[API for your own dashboards](docs/API.md)

## Install

Paste as root into a terminal:

```sh
bash <(curl -fsSL https://raw.githubusercontent.com/Syntax-XXX/rackpanel/main/install.sh)
```

The installer asks for username, password and port, adds the [RSRB package repository](https://syntax-xxx.github.io/rsrb/),
installs the `rackpanel` package and shows the address at the end.
Needs Debian or Ubuntu with systemd (armel, armhf, arm64, amd64).

Already have the RSRB repository? Then it's just:

```sh
sudo apt install rackpanel        # asks for user and password
```

Without questions, e.g. in scripts:

```sh
RACKPANEL_YES=1 RACKPANEL_USER=admin RACKPANEL_PASSWORD='secret' bash <(curl -fsSL https://raw.githubusercontent.com/Syntax-XXX/rackpanel/main/install.sh)
```

**Update:** `apt upgrade` or *Updates & Packages* in the panel. **Change port/user:** `dpkg-reconfigure rackpanel`.
**Forgot the password:** `rackpanel -set-user admin` (reads the new password from stdin). **Remove:** `apt remove rackpanel`.

## Containers (RackBox)

`apt install rackbox` adds a **Containers** page: Docker images without a daemon. RackBox pulls an image for your CPU
straight from the registry (ARMv5 included), unpacks it into a folder and runs every container as its own systemd
service via `systemd-nspawn` – no Docker daemon, no overlayfs, nothing resident besides the containers themselves.
The app store lists images that really exist for your CPU (on ARMv5: nginx, Nextcloud, PostgreSQL, Redis, …),
and any other image can be run from the same page or the command line:

```sh
rackbox install nginx                       # app from the catalog
rackbox create -v /volume1/www:/usr/share/nginx/html -web :8080/ web httpd
rackbox start web; rackbox ls; rackbox logs web
```

Data folders live on your data volume (`/volume1/apps/<app>`), images in `/volume1/@rackbox`.
Memory limits need a kernel with memory cgroup (on the RS812: the `rsrb-full` kernel).

## Add-ons

RackPanel can be extended with add-ons – a folder with `addon.json` and `page.js`, usually shipped in a package.
RackBox is one. How to write your own: [docs/ADDONS.md](docs/ADDONS.md).

## Synology RS812 without DSM

RackPanel was made for a Synology RS812 running current Debian instead of DSM.
How that works (kernel, boot, fan, recovery): **[rs812-debian](https://github.com/Syntax-XXX/rs812-debian)**.

## Apps

Apps are regular Debian packages with a service and a web UI (Transmission, MiniDLNA, Mosquitto, FTP).
The catalog lives in [`www/apps.json`](www/apps.json) – one entry per app, suggestions welcome.
For Docker images see *Containers* below.

## Build

```sh
./build.sh 1.2.0        # needs Go 1.22+, puts the release archives into dist/
```

The web UI in `www/` is plain HTML/CSS/JS without a build step; the server in `server/` has a single dependency (bcrypt).
Debian packages and the repository are built in [rsrb](https://github.com/Syntax-XXX/rsrb).

## Security

RackPanel runs as root and is meant for your own network. Don't expose it to the internet – put a VPN
(e.g. Tailscale or WireGuard) in front of it.

## License

MIT
