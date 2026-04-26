#!/bin/bash

go install github.com/shadowsocks/go-shadowsocks2@latest
go-shadowsocks2 -s 'ss://AEAD_CHACHA20_POLY1305:密码@:端口' -verbose

