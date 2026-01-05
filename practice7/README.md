aruzhan@MacBook-Pro-Aruzhan ~ % chmod 600 /Users/aruzhan/Downloads/ssh-key-1767608738655/ssh-key-1767608738655

aruzhan@MacBook-Pro-Aruzhan ~ % ssh -i /Users/aruzhan/Downloads/ssh-key-1767608738655/ssh-key-1767608738655 -l saruj 94.131.83.2

Welcome to Ubuntu 24.04.3 LTS (GNU/Linux 6.8.0-90-generic x86_64)

 * Documentation:  https://help.ubuntu.com
 * Management:     https://landscape.canonical.com
 * Support:        https://ubuntu.com/pro

 System information as of Mon Jan  5 14:13:00 UTC 2026

  System load:  0.0                Processes:             108
  Usage of /:   12.6% of 16.78GB   Users logged in:       1
  Memory usage: 11%                IPv4 address for eth0: 10.128.0.19
  Swap usage:   0%


Expanded Security Maintenance for Applications is not enabled.

0 updates can be applied immediately.

Enable ESM Apps to receive additional future security updates.
See https://ubuntu.com/esm or run: sudo pro status


Last login: Mon Jan  5 14:02:41 2026 from 37.99.19.57


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


saruj@compute-vm-2-2-18-hdd-1767608579163:/$ kcat -C \
         -b kz1-a-59d8gnlt4r1cl0ev.mdb.yandexcloud.kz:9091 \
         -t orders \
         -X security.protocol=SASL_SSL \
         -X sasl.mechanism=SCRAM-SHA-512 \
         -X sasl.username="user" \
         -X sasl.password="12345678" \
         -X ssl.ca.location=/usr/local/share/ca-certificates/Yandex/YandexInternalRootCA.crt -Z -K:
% Reached end of topic orders [0] at offset 0
% Reached end of topic orders [1] at offset 0
% Reached end of topic orders [2] at offset 0


