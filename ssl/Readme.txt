openssl req -newkey rsa:2048 \
  -new -nodes -x509 \
  -days 3650 \
  -out cert.pem \
  -keyout key.pem \
  -subj "/C=HC/ST=HttpCtl/L=HttpCtl/O=HttpCtl/OU=HttpCtl/CN=httpctl"


https://stackoverflow.com/questions/22666163/golang-tls-with-selfsigned-certificate

http://pro-tips-dot-com.tumblr.com/post/65472594329/golang-establish-secure-http-connections-with
