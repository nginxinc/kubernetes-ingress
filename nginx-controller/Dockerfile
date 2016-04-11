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

CMD ["/nginx-ingress"]
