#!/usr/bin/env bash
# RackPanel – Installer. Aufruf:
#   bash <(curl -fsSL https://raw.githubusercontent.com/Syntax-XXX/rackpanel/main/install.sh)
# Ohne Fragen (Skripte):  RACKPANEL_YES=1 RACKPANEL_PASSWORD=… [RACKPANEL_USER=admin] [RACKPANEL_PORT=80] [RACKPANEL_EXTRAS="samba wsdd"]
# Entfernen:              … install.sh) --uninstall
set -euo pipefail

main() {
  local REPO="Syntax-XXX/rackpanel" BIN=/usr/local/bin/rackpanel WWW=/usr/share/rackpanel UNIT=/etc/systemd/system/rackpanel.service
  local TITLE="RackPanel Installer"
  die() { echo "Fehler: $*" >&2; exit 1; }

  [ "$(id -u)" = 0 ] || die "Bitte als root ausführen (sudo -i)."
  command -v apt-get >/dev/null || die "Nur für Debian/Ubuntu (apt-get fehlt)."
  command -v systemctl >/dev/null || die "systemd wird benötigt."

  if [ "${1:-}" = "--uninstall" ]; then
    systemctl disable --now rackpanel.service 2>/dev/null || true
    rm -rf "$BIN" "$WWW" "$UNIT"
    systemctl daemon-reload
    echo "RackPanel entfernt. Einstellungen liegen noch in /var/lib/rackpanel (bei Bedarf löschen)."
    return
  fi

  local arch
  case "$(uname -m)" in
    armv5*|armv6*) arch=armv5 ;;
    armv7*) arch=armv7 ;;
    aarch64|arm64) arch=arm64 ;;
    x86_64|amd64) arch=amd64 ;;
    *) die "Architektur $(uname -m) wird nicht unterstützt." ;;
  esac

  local user="${RACKPANEL_USER:-admin}" pass="${RACKPANEL_PASSWORD:-}" port="${RACKPANEL_PORT:-80}" extras="${RACKPANEL_EXTRAS:-}"
  local ui=1
  if [ -n "${RACKPANEL_YES:-}" ] || ! [ -r /dev/tty ]; then ui=0; fi

  export DEBIAN_FRONTEND=noninteractive
  local need=()
  command -v curl >/dev/null || need+=(curl ca-certificates)
  [ $ui = 0 ] || command -v whiptail >/dev/null || need+=(whiptail)
  if [ ${#need[@]} -gt 0 ]; then
    echo "Installiere Hilfsprogramme: ${need[*]}"
    apt-get update -q && apt-get install -y -q "${need[@]}"
  fi

  # whiptail schreibt die Antwort auf stderr; /dev/tty, weil stdin bei "curl | bash" das Skript ist
  ask() { whiptail --title "$TITLE" "$@" 3>&1 1>&2 2>&3 </dev/tty; }

  if [ $ui = 1 ]; then
    ask --yesno "Willkommen!\n\nRackPanel ist eine Weboberfläche für dein NAS: Speicher, Dateien, Freigaben, Benutzer, Apps, Updates.\n\nGerät: $(hostname) ($(uname -m))\n\nJetzt installieren?" 16 70 || { echo "Abgebrochen."; return; }
    while :; do
      user=$(ask --inputbox "Benutzername für die Anmeldung im Panel\n(Kleinbuchstaben, Ziffern)" 10 70 "$user") || { echo "Abgebrochen."; return; }
      [[ "$user" =~ ^[a-z_][a-z0-9_.-]{0,31}$ ]] && break
      ask --msgbox "Der Name darf nur Kleinbuchstaben, Ziffern, . _ - enthalten." 8 70
    done
    while :; do
      pass=$(ask --passwordbox "Passwort für $user (mindestens 5 Zeichen)" 10 70) || { echo "Abgebrochen."; return; }
      local again; again=$(ask --passwordbox "Passwort wiederholen" 10 70) || { echo "Abgebrochen."; return; }
      if [ ${#pass} -lt 5 ]; then ask --msgbox "Das Passwort ist zu kurz." 8 70
      elif [ "$pass" != "$again" ]; then ask --msgbox "Die Passwörter stimmen nicht überein." 8 70
      else break; fi
    done
    while :; do
      port=$(ask --inputbox "Port der Weboberfläche\n(80 = einfach http://$(hostname) im Browser)" 10 70 "$port") || { echo "Abgebrochen."; return; }
      [[ "$port" =~ ^[0-9]+$ ]] && [ "$port" -ge 1 ] && [ "$port" -le 65535 ] || { ask --msgbox "Bitte eine Zahl von 1 bis 65535." 8 70; continue; }
      if ss -ltnH "sport = :$port" 2>/dev/null | grep -q . && ! systemctl is-active -q rackpanel.service; then
        ask --yesno "Port $port ist schon belegt. Trotzdem verwenden?" 8 70 || continue
      fi
      break
    done
    extras=$(ask --checklist "Was soll gleich mit installiert werden?\n(Leertaste wählt aus – alles geht auch später im Panel)" 16 74 5 \
      "samba wsdd" "Windows-/Mac-Freigaben (SMB)" ON \
      "nfs-kernel-server" "Linux-Freigaben (NFS)" OFF \
      "smartmontools" "Festplatten-Gesundheit (SMART)" ON \
      "hdparm" "Festplatten schlafen legen" ON \
      "avahi-daemon" "Im Netzwerk als name.local sichtbar" OFF) || { echo "Abgebrochen."; return; }
    extras=${extras//\"/}
  else
    [ ${#pass} -ge 5 ] || die "RACKPANEL_PASSWORD fehlt oder ist kürzer als 5 Zeichen."
  fi

  tmp=$(mktemp -d) # nicht local: der EXIT-Trap räumt nach main auf
  trap 'rm -rf "$tmp"' EXIT
  if [ -n "${RACKPANEL_TARBALL:-}" ]; then
    cp "$RACKPANEL_TARBALL" "$tmp/rp.tar.gz"
  else
    echo "Lade RackPanel ($arch) …"
    curl -fL --retry 3 -o "$tmp/rp.tar.gz" "https://github.com/$REPO/releases/latest/download/rackpanel-linux-$arch.tar.gz" || die "Download fehlgeschlagen."
  fi
  tar -xzf "$tmp/rp.tar.gz" -C "$tmp"
  [ -x "$tmp/rackpanel" ] && [ -f "$tmp/www/index.html" ] || die "Archiv unvollständig."
  "$tmp/rackpanel" -version >/dev/null || die "Das Programm läuft auf diesem Gerät nicht."

  systemctl stop rackpanel.service 2>/dev/null || true
  install -m 755 "$tmp/rackpanel" "$BIN"
  rm -rf "$WWW" && mkdir -p "$WWW" && cp -r "$tmp/www" "$WWW/www"
  printf '%s\n' "$pass" | "$BIN" -set-user "$user"

  cat > "$UNIT" <<UNIT
[Unit]
Description=RackPanel NAS-Verwaltung
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=$BIN -listen :$port
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT
  systemctl daemon-reload
  systemctl enable --now rackpanel.service

  if [ -n "$extras" ]; then
    echo "Installiere: $extras"
    # shellcheck disable=SC2086
    apt-get update -q && apt-get install -y -q $extras || echo "Hinweis: Zusatzpakete konnten nicht alle installiert werden – im Panel unter »Updates & Pakete« nachholen."
  fi

  sleep 1
  systemctl is-active -q rackpanel.service || die "Der Dienst startet nicht: journalctl -u rackpanel -n 30"
  local ip; ip=$(ip -4 route get 1.1.1.1 2>/dev/null | sed -n 's/.* src \([0-9.]*\).*/\1/p')
  local url="http://${ip:-$(hostname)}$([ "$port" = 80 ] || echo ":$port")"
  if [ $ui = 1 ]; then ask --msgbox "Fertig!\n\nRackPanel läuft jetzt unter\n\n    $url\n\nAnmelden mit: $user" 13 70; fi
  echo "RackPanel läuft: $url  (Benutzer: $user)"
}

main "$@"
