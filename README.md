# RC Server (sunucu tarafı)

Linux sunucuda çalışan RC Server ajanı: uzaktan dosya yönetimi, dağıtım, sistem/nginx işlemleri, Docker ve terminal erişimi gibi işlemler için HTTPS üzerinden API sunar. Mobil istemci bu sunucuya bağlanır; kimlik doğrulama yapılandırmadaki `hash` değeri ile yapılır.

Detaylı güvenlik notları için [SECURITY.md](SECURITY.md) dosyasına bakın.

## Gereksinimler

- **Go** 1.25 veya üzeri (kaynak koddan derleme için; `go.mod` içindeki `go` satırı ile uyumlu olmalı)
- **Linux** (üretim kurulumu `install.sh` ile test edilmiştir)
- **OpenSSL** (kurulum betiği self-signed TLS sertifikası üretmek için)
- **systemd** (kurulum betiği servis olarak kaydeder)
- **Docker** isteğe bağlı; Docker API kullanan özellikler için sunucuda Docker kurulu olmalı ve servis bağımlılığı tanımlıdır

## Depoyu klonlama

```bash
git clone https://github.com/benahmetcelik/rcserver-server-side.git
cd rcserver-server-side
```

### Go yükseltme (sunucuda eski sürüm varsa)

Dağıtım paketindeki `golang-go` çoğu zaman **çok eski** kalır (ör. 1.18). Bu proje **en az Go 1.25** ister. Resmi ikili paketi kullanın ([indirme listesi](https://go.dev/dl/)); **linux/amd64** için örnek:

```bash
cd /tmp
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz   # sürümü siteden doğrulayın
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc && source ~/.bashrc
go version   # go1.25.x görmelisiniz
```

**ARM64** sunucuda dosya adı `go1.25.0.linux-arm64.tar.gz` olur. `which go` çıktısı `/usr/local/go/bin/go` olmalı; hâlâ eski sürüm görünüyorsa `PATH` sırasını kontrol edin.

### Kaynak koddan derleme

Proje kökünde:

```bash
go mod download   # isteğe bağlı
go build -o rcserver ./cmd/rcserver
```

Çıkan ikiliyi `./rcserver` olarak çalıştırabilir veya aşağıdaki kurulum adımında `/usr/local/bin/rcserver` konumuna kopyalanır.

## Linux’ta üretim kurulumu (`install.sh`)

Kurulum betiği:

- `/opt/rcserver`, `/etc/rcserver`, `/var/lib/rcserver` gibi dizinleri oluşturur
- `rcserver` sistem kullanıcısı ekler
- İkiliyi `/usr/local/bin/rcserver` konumuna kurar
- Yoksa self-signed TLS sertifikası ve varsayılan `config.yaml` üretir
- `hash` üretir ve `systemd` birimi (`rcserver.service`) ekleyip servisi etkinleştirir

**Adımlar:**

1. Depoyu sunucuya alın ve proje dizininde ikiliyi derleyin (yukarıdaki `go build` komutu).
2. Betiği **root** ile çalıştırın:

```bash
sudo ./install.sh
```

Betiğin ikiliyi aradığı varsayılan yol, betiğin bulunduğu dizindeki `./rcserver` dosyasıdır. İkiliyi başka yerde derlediyseniz **gerçek dosya yolunu** verin (örnek):

```bash
sudo BINARY=/root/rcserver-server-side/rcserver ./install.sh
```

### Ortam değişkenleri (isteğe bağlı)

| Değişken | Varsayılan | Açıklama |
|----------|------------|----------|
| `RC_ROOT` | `/opt/rcserver` | Kök dizin |
| `BIN_DST` | `/usr/local/bin/rcserver` | Kurulacak ikili yolu |
| `CFG_DIR` | `/etc/rcserver` | Yapılandırma ve TLS dosyaları |
| `STATE_DIR` | `/var/lib/rcserver` | Durum / deploy dizinleri |
| `RC_USER` | `rcserver` | Servisi çalıştıran sistem kullanıcısı |
| `BINARY` | `<install.sh dizini>/rcserver` | Kurulacak derlenmiş ikili |

### Servis komutları

```bash
sudo systemctl status rcserver
sudo systemctl restart rcserver
sudo journalctl -u rcserver -f
```

Yapılandırma dosyası: `/etc/rcserver/config.yaml`  
Varsayılan dinleme adresi: `:3300` (HTTPS, TLS açık)

## Geliştirme veya manuel çalıştırma

1. `config.example.yaml` dosyasını kopyalayıp düzenleyin (ör. `./config.yaml`).
2. `hash` boş olamaz; üretin:

```bash
export RC_SERVER_CONFIG=./config.yaml
./rcserver generate hash --config ./config.yaml
```

Komut, mobil uygulamada kullanmak için kutulanmış IP/port ve hash bilgisini terminale yazdırır.

3. Sunucuyu başlatın:

```bash
./rcserver serve --config ./config.yaml
```

`RC_SERVER_CONFIG` ortam değişkeni verilirse `--config` ile aynı dosyayı işaret edebilir.

## Yapılandırma özeti

Ana alanlar (`config.yaml`):

| Alan | Açıklama |
|------|----------|
| `listen_addr` | Dinleme adresi (örn. `:3300`) |
| `tls_enabled`, `tls_cert`, `tls_key` | TLS kullanımı ve sertifika yolları |
| `hash` | API kimlik doğrulama anahtarı (`generate hash` ile üretilir) |
| `file_roots` | Dosya API’sinin erişebileceği kök dizinler |
| `nginx_sites_dir`, `www_root`, `deploy_dir` | Nginx ve dağıtım yolları |
| `rate_per_second`, `rate_burst` | İstek hız sınırı |
| `exec_timeout_sec`, `max_output_bytes` | Uzak komut zaman aşımı ve çıktı limiti |

Örnek şablon: [config.example.yaml](config.example.yaml)

## API kimlik doğrulama

Korumalı uçlar için istek başlığı:

- `Authorization: Bearer <HASH>` veya
- `X-RC-Key: <HASH>`

`/health` genelde kimlik doğrulama gerektirmez (uygulama koduna göre doğrulayın).

## Sorun giderme

- **`hash` boş hatası:** `rcserver generate hash --config <yol>` çalıştırın.
- **TLS hatası:** `tls_enabled: true` iken `tls_cert` ve `tls_key` yollarının okunabilir olduğundan emin olun.
- **Nginx yenileme başarısız:** Ajan `rcserver` kullanıcısı ile çalışır; `nginx` komutları için ek `sudoers` veya farklı dağıtım modeli gerekebilir. Ayrıntılar [SECURITY.md](SECURITY.md) içinde.
