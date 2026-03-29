#!/usr/bin/env bash
set -euo pipefail

RC_ROOT="${RC_ROOT:-/opt/rcserver}"
BIN_DST="${BIN_DST:-/usr/local/bin/rcserver}"
CFG_DIR="${CFG_DIR:-/etc/rcserver}"
STATE_DIR="${STATE_DIR:-/var/lib/rcserver}"
USER_NAME="${RC_USER:-rcserver}"

die() { echo "install.sh: $*" >&2; exit 1; }

[[ "${EUID:-0}" -eq 0 ]] || die "root olarak çalıştırın (sudo)"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Örnek: BINARY=/root/rcserver-server-side/rcserver (gerçek dosya yolu; /tam/yol gibi yer tutucu kullanmayın)
BINARY="${BINARY:-$SCRIPT_DIR/rcserver}"
[[ -f "$BINARY" ]] || die "ikili bulunamadı: $BINARY — önce proje kökünde 'go build -o rcserver ./cmd/rcserver' çalıştırın veya BINARY= ile gerçek rcserver dosya yolunu verin"

install -d -m 0755 "$RC_ROOT" "$CFG_DIR" "$STATE_DIR"

if ! id "$USER_NAME" &>/dev/null; then
  useradd --system --home "$STATE_DIR" --shell /usr/sbin/nologin "$USER_NAME" || true
fi
chown -R "$USER_NAME:$USER_NAME" "$STATE_DIR" || true

install -m 0755 "$BINARY" "$BIN_DST"

TLS_CRT="$CFG_DIR/tls.crt"
TLS_KEY="$CFG_DIR/tls.key"
if [[ ! -f "$TLS_CRT" || ! -f "$TLS_KEY" ]]; then
  openssl req -x509 -newkey rsa:4096 -keyout "$TLS_KEY" -out "$TLS_CRT" -days 825 -nodes \
    -subj "/CN=rcserver/O=rcservers" >/dev/null 2>&1
  chmod 0640 "$TLS_KEY" "$TLS_CRT"
fi

CFG_FILE="$CFG_DIR/config.yaml"
if [[ ! -f "$CFG_FILE" ]]; then
  cat >"$CFG_FILE" <<EOF
listen_addr: ":3300"
tls_enabled: true
tls_cert: $TLS_CRT
tls_key: $TLS_KEY
hash: ""
file_roots:
  - /var/www
  - $STATE_DIR/sites
  - $STATE_DIR
nginx_sites_dir: /etc/nginx/sites-available
www_root: /var/www
deploy_dir: $STATE_DIR/sites
rate_per_second: 20
rate_burst: 40
exec_timeout_sec: 120
max_output_bytes: 2097152
EOF
  chmod 0600 "$CFG_FILE"
fi

export RC_SERVER_CONFIG="$CFG_FILE"
"$BIN_DST" generate hash --config "$CFG_FILE" || true

UNIT="/etc/systemd/system/rcserver.service"
cat >"$UNIT" <<EOF
[Unit]
Description=RC Server agent
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
User=$USER_NAME
Group=$USER_NAME
Environment=RC_SERVER_CONFIG=$CFG_FILE
ExecStart=$BIN_DST serve --config $CFG_FILE
Restart=on-failure
AmbientCapabilities=CAP_NET_BIND_SERVICE
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable rcserver.service
systemctl restart rcserver.service || systemctl start rcserver.service

echo "Kurulum tamam. Yapılandırma: $CFG_FILE"
echo "Servis: systemctl status rcserver"
echo "Mobil uygulama için HASH çıktısı için: RC_SERVER_CONFIG=$CFG_FILE $BIN_DST generate hash --config $CFG_FILE"
