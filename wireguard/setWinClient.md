#安装
https://www.wireguard.com/install/
wireguard-installer.exe

 
[Interface]
PrivateKey = 本地私钥
ListenPort = 51820
Address = 10.0.0.2/24
DNS = 8.8.8.8

[Peer]
PublicKey = 远程公钥
AllowedIPs = 0.0.0.0/0
Endpoint = 远程地址:51820

