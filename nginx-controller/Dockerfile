FROM ubuntu:14.04

MAINTAINER NGINX Docker Maintainers "docker-maint@nginx.com"

ENV NGINX_VERSION 1.9.10-1~trusty

RUN apt-key adv --keyserver hkp://pgp.mit.edu:80 --recv-keys 573BFD6B3D8FBC641079A6ABABF5BD827BD9BF62 \
	&& echo "deb http://nginx.org/packages/mainline/ubuntu/ trusty nginx" >> /etc/apt/sources.list \
	&& apt-get update \
	&& apt-get install -y ca-certificates nginx=${NGINX_VERSION} gettext-base \
	&& rm -rf /var/lib/apt/lists/*

EXPOSE 80 443

COPY nginx-ingress /
COPY nginx/ingress.tmpl /
COPY nginx/nginx.conf /etc/nginx/nginx.conf

RUN rm /etc/nginx/conf.d/*

# This works only if the nginx master process is started from the process which is the Docker entrypoint, and has its
# stdout and stderr set to the stdout and stderr of its parent (entrypoint). nginx must also run with `daemon off`,
# otherwise it will detach its stdout.
RUN ln -sf /dev/stdout /var/log/nginx/access.log
RUN ln -sf /dev/stderr /var/log/nginx/error.log

CMD ["/nginx-ingress"]
