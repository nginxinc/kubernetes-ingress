FROM nginx:1.9.15

# forward nginx access and error logs to stdout and stderr of the ingress
# controller process
RUN ln -sf /proc/1/fd/1 /var/log/nginx/access.log \
	&& ln -sf /proc/1/fd/2 /var/log/nginx/error.log

COPY nginx-ingress /
COPY nginx/ingress.tmpl /
COPY nginx/nginx.conf /etc/nginx/nginx.conf

RUN rm /etc/nginx/conf.d/*

CMD ["/nginx-ingress"]
