saruj@compute-vm-2-2-18-hdd-1767608579163:/$ sudo mkdir -p /usr/local/share/ca-certificates/Yandex
sudo wget "https://storage.yandexcloud.kz/cloud-certs/CA.pem" \
     -O /usr/local/share/ca-certificates/Yandex/YandexInternalRootCA.crt
sudo chmod 0644 /usr/local/share/ca-certificates/Yandex/YandexInternalRootCA.crt
--2026-01-05 14:43:39--  https://storage.yandexcloud.kz/cloud-certs/CA.pem
Resolving storage.yandexcloud.kz (storage.yandexcloud.kz)... 5.35.110.123, 2a07:aa40:2f::138
Connecting to storage.yandexcloud.kz (storage.yandexcloud.kz)|5.35.110.123|:443... connected.
HTTP request sent, awaiting response... 200 OK
Length: 5313 (5.2K) [application/x-x509-ca-cert]
Saving to: ‘/usr/local/share/ca-certificates/Yandex/YandexInternalRootCA.crt’

/usr/local/share/ca-certifica 100%[===============================================>]   5.19K  --.-KB/s    in 0s      

2026-01-05 14:43:39 (814 MB/s) - ‘/usr/local/share/ca-certificates/Yandex/YandexInternalRootCA.crt’ saved [5313/5313]

saruj@compute-vm-2-2-18-hdd-1767608579163:/$ sudo update-ca-certificates
Updating certificates in /etc/ssl/certs...
rehash: warning: skipping YandexCA-bundle.pem,it does not contain exactly one certificate or CRL
rehash: warning: skipping YandexInternalRootCA.pem,it does not contain exactly one certificate or CRL
rehash: warning: skipping ca-certificates.crt,it does not contain exactly one certificate or CRL
rehash: warning: skipping yandex-ca.pem,it does not contain exactly one certificate or CRL
4 added, 0 removed; done.
Running hooks in /etc/ca-certificates/update.d...
Processing triggers for ca-certificates-java (20240118) ...
Replacing debian:YandexInternalIntermediateCA.pem
Replacing debian:YandexInternalRootCA.pem
Replacing debian:YandexInternalIntermediateCA.pem
Replacing debian:YandexInternalRootCA.pem
done.
done.
