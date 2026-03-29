# RC Server güvenlik notları

## Kimlik doğrulama

- Tüm `/api/v1/*` uçları ( `/health` hariç) `Authorization: Bearer <HASH>` veya `X-RC-Key: <HASH>` ile korunur.
- `HASH` değerini güçlü tutun; paylaşmayın ve düzenli olarak `rcserver generate hash` ile yenileyin.

## TLS

- Kurulum betiği self-signed sertifika üretir. Mobil uygulamada ilk bağlantıda “kendi imzalı sertifikaya güven” seçeneği test içindir.
- Üretimde mümkünse geçerli bir sertifika (Let’s Encrypt vb.) kullanın ve mobilde güvenilir CA ile doğrulama yapın.

## Ağ

- Ajanı yalnızca güvendiğiniz ağlarda veya VPN üzerinden dinletebilirsiniz; güvenlik duvarında yalnızca gerekli IP’lere izin verin.

## İşletim kullanıcısı

- `install.sh` servisi `rcserver` sistem kullanıcısı ile çalıştırır. **Kök olarak çalıştırmayın.**

## Uzak komut ve Docker

- Uzak komut çalıştırma tam yetki anlamına gelebilir; kara liste basit bir katmandır, tam güvenlik sağlamaz.
- Docker soketi (`/var/run/docker.sock`) kök benzeri yetki verir; `rcserver` kullanıcısını `docker` grubuna eklemek güvenlik riskini artırır — yalnızca güvenilen ortamlarda kullanın.

## Nginx yeniden yükleme

- `nginx -t` ve `systemctl reload nginx` genelde root veya sudo gerektirir. Varsayılan kurulumda `rcserver` kullanıcısı bu komutları başarısız tamamlayabilir. Çözüm seçenekleri:
  - Nginx yapılandırma dizinine yazma ve `nginx` komutları için sınırlı `sudoers` kuralları, veya
  - Ajanı özel bir kilitli ortamda ve bilinçli şekilde daha yetkili bir kullanıcıyla çalıştırma (riski kabul ederek).

## Hız sınırlama

- Varsayılan: istemci IP başına saniyede ~20 istek, burst 40. `config.yaml` içinde `rate_per_second` ve `rate_burst` ile değiştirilebilir.

## Dosya kökleri

- `file_roots` dışına çıkan yollar reddedilir. Üretimde yalnızca gerekli dizinleri listeleyin.
