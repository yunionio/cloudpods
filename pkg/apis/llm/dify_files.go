package llm

const (
	DIFY_SSRF_ENTRYPINT_SHELL = `
mkdir -p /etc/squid

cat > /etc/squid/squid.conf.template <<'EOF'
%s
EOF

echo "[ENTRYPOINT] re-create snakeoil self-signed certificate removed in the build process"
if [ ! -f /etc/ssl/private/ssl-cert-snakeoil.key ]; then
    /usr/sbin/make-ssl-cert generate-default-snakeoil --force-overwrite > /dev/null 2>&1
fi

tail -F /var/log/squid/access.log 2>/dev/null &
tail -F /var/log/squid/error.log 2>/dev/null &
tail -F /var/log/squid/store.log 2>/dev/null &
tail -F /var/log/squid/cache.log 2>/dev/null &

echo "[ENTRYPOINT] replacing environment variables in the template"
awk '{
    while(match($0, /\${[A-Za-z_][A-Za-z_0-9]*}/)) {
        var = substr($0, RSTART+2, RLENGTH-3)
        val = ENVIRON[var]
        $0 = substr($0, 1, RSTART-1) val substr($0, RSTART+RLENGTH)
    }
    print
}' /etc/squid/squid.conf.template > /etc/squid/squid.conf

chown -R "$SQUID_USER":"$SQUID_USER" /etc/squid

/usr/sbin/squid -Nz
echo "[ENTRYPOINT] starting squid"
/usr/sbin/squid -f /etc/squid/squid.conf -NYC 1
`
	DIFY_SSRF_SQUID_CONFIGURATION_FILE = `visible_hostname localhost # set visible_hostname to avoid WARNING: Could not determine this machines public hostname. 
acl localnet src 0.0.0.1-0.255.255.255	# RFC 1122 "this" network (LAN)
acl localnet src 10.0.0.0/8		# RFC 1918 local private network (LAN)
acl localnet src 100.64.0.0/10		# RFC 6598 shared address space (CGN)
acl localnet src 169.254.0.0/16 	# RFC 3927 link-local (directly plugged) machines
acl localnet src 172.16.0.0/12		# RFC 1918 local private network (LAN)
acl localnet src 192.168.0.0/16		# RFC 1918 local private network (LAN)
acl localnet src fc00::/7       	# RFC 4193 local private network range
acl localnet src fe80::/10      	# RFC 4291 link-local (directly plugged) machines
acl SSL_ports port 443
# acl SSL_ports port 1025-65535   # Enable the configuration to resolve this issue: https://github.com/langgenius/dify/issues/12792
acl Safe_ports port 80		# http
acl Safe_ports port 21		# ftp
acl Safe_ports port 443		# https
acl Safe_ports port 70		# gopher
acl Safe_ports port 210		# wais
acl Safe_ports port 1025-65535	# unregistered ports
acl Safe_ports port 280		# http-mgmt
acl Safe_ports port 488		# gss-http
acl Safe_ports port 591		# filemaker
acl Safe_ports port 777		# multiling http
acl CONNECT method CONNECT
acl allowed_domains dstdomain .marketplace.dify.ai
http_access allow allowed_domains
http_access deny !Safe_ports
http_access deny CONNECT !SSL_ports
http_access allow localhost manager
http_access deny manager
http_access allow localhost
include /etc/squid/conf.d/*.conf
http_access deny all

################################## Proxy Server ################################
http_port ${HTTP_PORT}
coredump_dir ${COREDUMP_DIR}
refresh_pattern ^ftp:		1440	20%	10080
refresh_pattern ^gopher:	1440	0%	1440
refresh_pattern -i (/cgi-bin/|\?) 0	0%	0
refresh_pattern \/(Packages|Sources)(|\.bz2|\.gz|\.xz)$ 0 0% 0 refresh-ims
refresh_pattern \/Release(|\.gpg)$ 0 0% 0 refresh-ims
refresh_pattern \/InRelease$ 0 0% 0 refresh-ims
refresh_pattern \/(Translation-.*)(|\.bz2|\.gz|\.xz)$ 0 0% 0 refresh-ims
refresh_pattern .		0	20%	4320


# cache_dir ufs /var/spool/squid 100 16 256
# upstream proxy, set to your own upstream proxy IP to avoid SSRF attacks
# cache_peer 172.1.1.1 parent 3128 0 no-query no-digest no-netdb-exchange default

################################## Reverse Proxy To Sandbox ################################
http_port ${REVERSE_PROXY_PORT} accel vhost
cache_peer ${SANDBOX_HOST} parent ${SANDBOX_PORT} 0 no-query originserver
acl src_all src all
http_access allow src_all

# Unless the option's size is increased, an error will occur when uploading more than two files.
client_request_buffer_max_size 100 MB
`
)

const (
	DIFY_NGINX_ENTRYPINT_SHELL = `
mkdir -p /etc/nginx
cat > /etc/nginx/nginx.conf.template <<'EOF'
%s
EOF

cat > /etc/nginx/proxy.conf.template <<'EOF'
%s
EOF

cat > /etc/nginx/conf.d/default.conf.template <<'EOF'
%s
EOF

env_vars=$(printenv | cut -d= -f1 | sed 's/^/$/g' | paste -sd, -)
echo "Substituting variables: $env_vars"

envsubst "$env_vars" < /etc/nginx/nginx.conf.template > /etc/nginx/nginx.conf
envsubst "$env_vars" < /etc/nginx/proxy.conf.template > /etc/nginx/proxy.conf
envsubst "$env_vars" < /etc/nginx/conf.d/default.conf.template > /etc/nginx/conf.d/default.conf

exec nginx -g 'daemon off;'`
	DIFY_NGINX_NGINX_CONF_FILE = `
user  nginx;
worker_processes  ${NGINX_WORKER_PROCESSES};

error_log  /var/log/nginx/error.log notice;
pid        /var/run/nginx.pid;


events {
    worker_connections  1024;
}


http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    #tcp_nopush     on;

    keepalive_timeout  ${NGINX_KEEPALIVE_TIMEOUT};

    #gzip  on;
    client_max_body_size ${NGINX_CLIENT_MAX_BODY_SIZE};

    include /etc/nginx/conf.d/*.conf;
}
`
	DIFY_NGINX_PROXY_CONF_FILE = `
proxy_set_header Host $host;
proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
proxy_set_header X-Forwarded-Proto $scheme;
proxy_set_header X-Forwarded-Port $server_port;
proxy_http_version 1.1;
proxy_set_header Connection "";
proxy_buffering off;
proxy_read_timeout ${NGINX_PROXY_READ_TIMEOUT};
proxy_send_timeout ${NGINX_PROXY_SEND_TIMEOUT};
`
	DIFY_NGINX_DEFAULT_CONF_FILE = `# Please do not directly edit this file. Instead, modify the .env variables related to NGINX configuration.

server {
    listen ${NGINX_PORT};
    server_name ${NGINX_SERVER_NAME};

    location /console/api {
      proxy_pass http://localhost:5001;
      include proxy.conf;
    }

    location /api {
      proxy_pass http://localhost:5001;
      include proxy.conf;
    }

    location /v1 {
      proxy_pass http://localhost:5001;
      include proxy.conf;
    }

    location /files {
      proxy_pass http://localhost:5001;
      include proxy.conf;
    }

    location /explore {
      proxy_pass http://localhost:3000;
      include proxy.conf;
    }

    location /e/ {
      proxy_pass http://localhost:5002;
      proxy_set_header Dify-Hook-Url $scheme://$host$request_uri;
      include proxy.conf;
    }

    location / {
      proxy_pass http://localhost:3000;
      include proxy.conf;
    }
    location /mcp {
      proxy_pass http://localhost:5001;
      include proxy.conf;
    }
    # placeholder for acme challenge location
    # ${ACME_CHALLENGE_LOCATION}

    # placeholder for https config defined in https.conf.template
    # ${HTTPS_CONFIG}
}
`
)

const (
	DIFY_SANDBOX_WRITE_CONF_SHELL = `
cat > /conf/config.yaml <<'EOF'
%s
EOF

cat > /conf/config.yaml.template <<'EOF'
%s
EOF

touch /dependencies/python-requirements.txt

# decompress nodejs
tar -xvf $NODE_TAR_XZ -C /opt
ln -s $NODE_DIR/bin/node /usr/local/bin/node
rm -f $NODE_TAR_XZ

# start main
/main
  `
	DIFY_SANDBOX_CONF_FILE = `app:
  port: 8194
  debug: True
  key: dify-sandbox
max_workers: 4
max_requests: 50
worker_timeout: 5
python_path: /usr/local/bin/python3
enable_network: True # please make sure there is no network risk in your environment
allowed_syscalls: # please leave it empty if you have no idea how seccomp works
proxy:
  socks5: ''
  http: ''
  https: ''
`
	DIFY_SANDBOX_CONF_TEMP_FILE = `app:
  port: 8194
  debug: True
  key: dify-sandbox
max_workers: 4
max_requests: 50
worker_timeout: 5
python_path: /usr/local/bin/python3
python_lib_path:
  - /usr/local/lib/python3.10
  - /usr/lib/python3.10
  - /usr/lib/python3
  - /usr/lib/x86_64-linux-gnu
  - /etc/ssl/certs/ca-certificates.crt
  - /etc/nsswitch.conf
  - /etc/hosts
  - /etc/resolv.conf
  - /run/systemd/resolve/stub-resolv.conf
  - /run/resolvconf/resolv.conf
  - /etc/localtime
  - /usr/share/zoneinfo
  - /etc/timezone
  # add more paths if needed
python_pip_mirror_url: https://pypi.tuna.tsinghua.edu.cn/simple
nodejs_path: /usr/local/bin/node
enable_network: True
allowed_syscalls:
  - 1
  - 2
  - 3
  # add all the syscalls which you require
proxy:
  socks5: ''
  http: ''
  https: ''
`
)
