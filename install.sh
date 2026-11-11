#!/usr/bin/env bash
# RackPanel installer – adds the RSRB APT repository and installs the rackpanel package.
#   bash <(curl -fsSL https://raw.githubusercontent.com/Syntax-XXX/rackpanel/main/install.sh)
# Unattended:  RACKPANEL_YES=1 RACKPANEL_PASSWORD=… [RACKPANEL_USER=admin] [RACKPANEL_PORT=80] [RACKPANEL_EXTRAS="samba wsdd"]
# Remove:      … install.sh) --uninstall
set -euo pipefail

main() {
  local APT_URL="https://syntax-xxx.github.io/rsrb" TITLE="RackPanel installer"
  die() { echo "Error: $*" >&2; exit 1; }

  [ "$(id -u)" = 0 ] || die "please run as root (sudo -i)."
  command -v apt-get >/dev/null || die "Debian/Ubuntu only (apt-get not found)."
  command -v systemctl >/dev/null || die "systemd is required."

  if [ "${1:-}" = "--uninstall" ]; then
    apt-get remove -y rackpanel || true
    echo "RackPanel removed. Settings are kept in /var/lib/rackpanel (apt purge rackpanel deletes them)."
    return
  fi

  local arch; arch=$(dpkg --print-architecture)
  case "$arch" in armel|armhf|arm64|amd64) ;; *) die "architecture $arch is not supported." ;; esac

  local user="${RACKPANEL_USER:-admin}" pass="${RACKPANEL_PASSWORD:-}" port="${RACKPANEL_PORT:-80}" extras="${RACKPANEL_EXTRAS:-}"
  local ui=1
  # -r is not enough: without a controlling terminal (cron, ssh without -t) /dev/tty can't be opened
  if [ -n "${RACKPANEL_YES:-}" ] || ! { : </dev/tty; } 2>/dev/null; then ui=0; fi

  export DEBIAN_FRONTEND=noninteractive
  local need=()
  command -v curl >/dev/null || need+=(curl ca-certificates)
  [ $ui = 0 ] || command -v whiptail >/dev/null || need+=(whiptail)
  if [ ${#need[@]} -gt 0 ]; then
    echo "Installing helpers: ${need[*]}"
    apt-get update -q && apt-get install -y -q "${need[@]}"
  fi

  # whiptail prints the answer on stderr; read from /dev/tty because stdin is the script with "curl | bash"
  ask() { whiptail --title "$TITLE" "$@" 3>&1 1>&2 2>&3 </dev/tty; }

  if [ $ui = 1 ]; then
    ask --yesno "Welcome!\n\nRackPanel is a web UI for your NAS: storage, files, shares, users, apps, updates.\n\nDevice: $(hostname) ($arch)\n\nInstall now?" 16 70 || { echo "Cancelled."; return; }
    while :; do
      user=$(ask --inputbox "Username for signing in to the panel\n(lowercase letters, digits)" 10 70 "$user") || { echo "Cancelled."; return; }
      [[ "$user" =~ ^[a-z_][a-z0-9_.-]{0,31}$ ]] && break
      ask --msgbox "Only lowercase letters, digits and . _ - are allowed." 8 70
    done
    while :; do
      pass=$(ask --passwordbox "Password for $user (at least 5 characters)" 10 70) || { echo "Cancelled."; return; }
      local again; again=$(ask --passwordbox "Repeat the password" 10 70) || { echo "Cancelled."; return; }
      if [ ${#pass} -lt 5 ]; then ask --msgbox "The password is too short." 8 70
      elif [ "$pass" != "$again" ]; then ask --msgbox "The passwords don't match." 8 70
      else break; fi
    done
    while :; do
      port=$(ask --inputbox "Web UI port\n(80 = just open http://$(hostname) in a browser)" 10 70 "$port") || { echo "Cancelled."; return; }
      [[ "$port" =~ ^[0-9]+$ ]] && [ "$port" -ge 1 ] && [ "$port" -le 65535 ] || { ask --msgbox "Please enter a number from 1 to 65535." 8 70; continue; }
      if ss -ltnH "sport = :$port" 2>/dev/null | grep -q . && ! systemctl is-active -q rackpanel.service; then
        ask --yesno "Port $port is already in use. Use it anyway?" 8 70 || continue
      fi
      break
    done
    extras=$(ask --checklist "Install these right away?\n(space toggles – everything can be added later in the panel)" 16 74 5 \
      "samba wsdd" "Windows/Mac shares (SMB)" ON \
      "nfs-kernel-server" "Linux shares (NFS)" OFF \
      "smartmontools" "Disk health (SMART)" ON \
      "hdparm" "Disk standby" ON \
      "avahi-daemon" "Visible on the network as name.local" OFF) || { echo "Cancelled."; return; }
    extras=${extras//\"/}
  else
    [ ${#pass} -ge 5 ] || die "RACKPANEL_PASSWORD is missing or shorter than 5 characters."
  fi

  echo "Adding the RSRB package repository ($APT_URL) …"
  curl -fsSL "$APT_URL/rsrb.gpg" -o /usr/share/keyrings/rsrb.gpg || die "could not download the repository key."
  echo "deb [signed-by=/usr/share/keyrings/rsrb.gpg] $APT_URL stable main" > /etc/apt/sources.list.d/rsrb.list

  # answers for the package's debconf questions; the password goes through stdin, never the command line,
  # and the package removes it from the debconf database right after use
  printf 'rackpanel rackpanel/username string %s\nrackpanel rackpanel/password password %s\nrackpanel rackpanel/port string %s\n' \
    "$user" "$pass" "$port" | debconf-set-selections

  apt-get update -q
  # shellcheck disable=SC2086
  apt-get install -y -q rackpanel $extras || apt-get install -y -q rackpanel || die "installation failed."

  sleep 1
  systemctl is-active -q rackpanel.service || die "the service does not start: journalctl -u rackpanel -n 30"
  local ip; ip=$(ip -4 route get 1.1.1.1 2>/dev/null | sed -n 's/.* src \([0-9.]*\).*/\1/p')
  local url="http://${ip:-$(hostname)}$([ "$port" = 80 ] || echo ":$port")"
  if [ $ui = 1 ]; then ask --msgbox "Done!\n\nRackPanel is running at\n\n    $url\n\nSign in as: $user\n\nUpdates arrive with apt upgrade or in the panel under Updates." 15 70; fi
  echo "RackPanel is running: $url  (user: $user)"
}

main "$@"
