openssl genrsa -out server.key 2048
openssl req -x509 -new -nodes -key server.key -subj "/CN=*" -days 3650 -out server.pem

./httpProxy -cert ./server.pem -key server.key
