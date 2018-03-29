openssl genrsa -out gomitmproxy-ca-pk.pem 2048
openssl req -new -x509 -days 36500 -key gomitmproxy-ca-pk.pem -out gomitmproxy-ca-cert.pem