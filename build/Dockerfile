FROM nginx:1.17.5

# forward nginx access and error logs to stdout and stderr of the ingress
# controller process
RUN ln -sf /proc/1/fd/1 /var/log/nginx/access.log \
	&& ln -sf /proc/1/fd/1 /var/log/nginx/stream-access.log \
	&& ln -sf /proc/1/fd/2 /var/log/nginx/error.log

RUN mkdir -p /var/lib/nginx \
	&& mkdir -p /etc/nginx/secrets \
	&& apt-get update \
	&& apt-get install -y libcap2-bin \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx \
	&& chown -R nginx:0 /etc/nginx \
	&& chown -R nginx:0 /var/cache/nginx \
	&& chown -R nginx:0 /var/lib/nginx \
	&& apt-get remove --purge -y libcap2-bin \
	&& rm /etc/nginx/conf.d/* \
	&& rm -rf /var/lib/apt/lists/*

COPY nginx-ingress internal/configs/version1/nginx.ingress.tmpl internal/configs/version1/nginx.tmpl internal/configs/version2/nginx.virtualserver.tmpl /

# Uncomment the line below if you would like to add the default.pem to the image
# and use it as a certificate and key for the default server
# ADD default.pem /etc/nginx/secrets/default

USER nginx

ENTRYPOINT ["/nginx-ingress"]
