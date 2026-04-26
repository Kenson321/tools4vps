#!/bin/bash

#安装
wget https://git.zx2c4.com/wireguard-go/snapshot/wireguard-go-0.0.20230223.tar.xz
tar xf wireguard-go-0.0.20230223.tar.xz
cd wireguard-go-0.0.20230223/
make

wget https://git.zx2c4.com/wireguard-tools/snapshot/wireguard-tools-1.0.20210914.tar.xz
cd src
make
make install

#添加虚拟网卡
wireguard-go wg0
ps aux | grep wireguard
 
#指定ip
ip addr add 10.0.0.1/24 dev wg0
 
#密钥
umask 077
wg genkey > privatekey
wg pubkey < privatekey > publickey

#配置
wg set wg0 listen-port 51820 private-key ./privatekey
wg set wg0 peer 客户端公钥 remove
wg set wg0 peer 客户端公钥 allowed-ips 10.0.0.2/32
 
#启动网卡
ip link set wg0 up
 
#查看设置
wg

#服务端设置网关
sudo sysctl -w net.ipv4.ip_forward=1
sudo iptables -A FORWARD -i wg0 -j ACCEPT
sudo iptables -A FORWARD -o wg0 -j ACCEPT
sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
 
