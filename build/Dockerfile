# syntax=docker/dockerfile:1.0-experimental
ARG NGINX_VERSION=1.19.7
ARG BUILD_OS=debian

############################################# Base image for Debian #############################################
FROM nginx:$NGINX_VERSION AS debian
ARG IC_VERSION

LABEL version=debian-nginx-$NGINX_VERSION-$IC_VERSION
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

RUN apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y libcap2-bin \
	&& rm -rf /var/lib/apt/lists/*


############################################# Base image for Alpine #############################################
FROM nginx:$NGINX_VERSION-alpine AS alpine
ARG IC_VERSION

LABEL version=alpine-nginx-$NGINX_VERSION-$IC_VERSION
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

RUN apk add --no-cache libcap \
	&& rm -rf /var/cache/apk/*


############################################# Base image for Debian with NGINX Plus #############################################
FROM debian:buster-slim AS debian-plus
ARG IC_VERSION
ENV NGINX_PLUS_VERSION 23-1~buster
ENV NGINX_NJS_VERSION 23+0.5.0-1~buster

LABEL version=debian-nginx-plus-$NGINX_PLUS_VERSION-$IC_VERSION
LABEL maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>"

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	set -x \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y ca-certificates gnupg wget \
	&& wget https://nginx.org/keys/nginx_signing.key \
	&& gpg --no-default-keyring --keyring nginx_keyring.gpg --import nginx_signing.key \
	&& gpg --no-default-keyring --keyring nginx_keyring.gpg --export > /etc/apt/trusted.gpg.d/nginx_signing.gpg \
	&& echo "Acquire::https::plus-pkgs.nginx.com::Verify-Peer \"true\";" >> /etc/apt/apt.conf.d/90nginx \
	&& echo "Acquire::https::plus-pkgs.nginx.com::Verify-Host \"true\";" >> /etc/apt/apt.conf.d/90nginx \
	&& echo "Acquire::https::plus-pkgs.nginx.com::SslCert     \"/etc/ssl/nginx/nginx-repo.crt\";" >> /etc/apt/apt.conf.d/90nginx \
	&& echo "Acquire::https::plus-pkgs.nginx.com::SslKey      \"/etc/ssl/nginx/nginx-repo.key\";" >> /etc/apt/apt.conf.d/90nginx \
	&& echo "Acquire::https::plus-pkgs.nginx.com::User-Agent  \"k8s-ic-$IC_VERSION-apt\";" >> /etc/apt/apt.conf.d/90nginx \
	&& printf "deb https://plus-pkgs.nginx.com/debian buster nginx-plus\n" > /etc/apt/sources.list.d/nginx-plus.list \
	&& apt-get update && apt-get install --no-install-recommends --no-install-suggests -y \
	apt-transport-https libcap2-bin nginx-plus=${NGINX_PLUS_VERSION} nginx-plus-module-njs=${NGINX_NJS_VERSION} \
	&& apt-get purge --auto-remove -y apt-transport-https gnupg wget \
	&& rm -rf /var/lib/apt/lists/*

COPY --chown=nginx:0 internal/configs/oidc/* /etc/nginx/oidc/


############################################# Base image for Debian with NGINX Plus and App Protect #############################################
FROM debian-plus as debian-plus-ap
ARG IC_VERSION

LABEL version=debian-nginx-plus-ap-$NGINX_PLUS_VERSION-$IC_VERSION

ENV APPPROTECT_MODULE_VERSION 23+3.332.0-1~buster
ENV APPPROTECT_PLUGIN_VERSION 3.332.0-1~buster
ENV APPPROTECT_ENGINE_VERSION 6.3.6-1~buster
ENV APPPROTECT_COMPILER_VERSION 6.3.6-1~buster
ENV APPPROTECT_SIG_VERSION 2021.02.26-1~buster
ENV APPPROTECT_THREAT_CAMPAIGNS_VERSION 2021.03.02-1~buster

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	set -x \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y gnupg wget \
	&& wget https://nginx.org/keys/app-protect-sigs.key \
	&& gpg --no-default-keyring --keyring app_protect_keyring.gpg --import app-protect-sigs.key \
	&& gpg --no-default-keyring --keyring app_protect_keyring.gpg --export > /etc/apt/trusted.gpg.d/nginx_app_signing.gpg \
	&& sed -i '$ d' /etc/apt/apt.conf.d/90nginx \
	&& echo "Acquire::https::plus-pkgs.nginx.com::User-Agent  \"k8s-ic-$IC_VERSION-app-$APPPROTECT_VERSION-apt\";" >> /etc/apt/apt.conf.d/90nginx \
	&& echo "deb https://app-protect-sigs.nginx.com/debian/ buster nginx-plus\n" | tee /etc/apt/sources.list.d/app-protect-sigs.list \
	&& echo "Acquire::https::app-protect-sigs.nginx.com::Verify-Peer \"true\";" >> /etc/apt/apt.conf.d/90app-protect-sigs \
	&& echo "Acquire::https::app-protect-sigs.nginx.com::Verify-Host \"true\";" >> /etc/apt/apt.conf.d/90app-protect-sigs \
	&& echo "Acquire::https::app-protect-sigs.nginx.com::SslCert     \"/etc/ssl/nginx/nginx-repo.crt\";" >> /etc/apt/apt.conf.d/90app-protect-sigs \
	&& echo "Acquire::https::app-protect-sigs.nginx.com::SslKey      \"/etc/ssl/nginx/nginx-repo.key\";" >> /etc/apt/apt.conf.d/90app-protect-sigs \
	&& apt-get update && apt-get install -y apt-transport-https nginx-plus-module-appprotect=$APPPROTECT_MODULE_VERSION \
	app-protect-plugin=$APPPROTECT_PLUGIN_VERSION \
	app-protect-engine=$APPPROTECT_ENGINE_VERSION \
	app-protect-compiler=$APPPROTECT_COMPILER_VERSION \
	app-protect=$APPPROTECT_MODULE_VERSION \
	&& apt-get install -y app-protect-attack-signatures${APPPROTECT_SIG_VERSION:+=$APPPROTECT_SIG_VERSION} \
	&& apt-get install -y app-protect-threat-campaigns${APPPROTECT_THREAT_CAMPAIGNS_VERSION:+=$APPPROTECT_THREAT_CAMPAIGNS_VERSION} \
	&& apt-get purge --auto-remove -y apt-transport-https gnupg wget \
	&& rm -rf /var/lib/apt/lists/* \
	&& rm /etc/apt/apt.conf.d/90app-protect-sigs /etc/apt/sources.list.d/app-protect-sigs.list

RUN mkdir -p /etc/nginx/waf /etc/nginx/waf/nac-policies /etc/nginx/waf/nac-logconfs /etc/nginx/waf/nac-usersigs /var/log/app_protect /opt/app_protect \
	&& chown -R nginx:0 /etc/app_protect /usr/share/ts /var/log/app_protect/ /opt/app_protect/ /var/log/nginx/ \
	&& touch /etc/nginx/waf/nac-usersigs/index.conf

RUN printf "MODULE = ALL;\nLOG_LEVEL = TS_CRIT;\nFILE = 2;\n" > /etc/app_protect/bd/logger.cfg \
	&& printf "[config_set_compiler]\nlog_level=fatal\n" >> /etc/app_protect/tools/asm_logging.conf \
	&& for v in \
	asm_config_server \
	lock_factory \
	bd_agent \
	import_export_policy \
	set_active \
	; do sed -i "/\[$v/a log_level=fatal" "/etc/app_protect/tools/asm_logging.conf" \
	; done

COPY --chown=nginx:0 build/log-default.json /etc/nginx

# Uncomment the lines below if you want to install a custom CA certificate
# COPY build/*.crt  /usr/local/share/ca-certificates/
# RUN update-ca-certificates


############################################# Base image for Openshift #############################################
FROM registry.access.redhat.com/ubi8/ubi:8.3 AS openshift-base

LABEL name="NGINX Ingress Controller" \
	description="The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	summary="The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	io.openshift.tags="nginx,ingress-controller,ingress,controller,kubernetes,openshift" \
	maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>" \
	vendor="NGINX Inc <kubernetes@nginx.com>"

RUN set -x \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx


RUN mkdir licenses
COPY --chown=nginx:0 LICENSE /licenses


############################################# Base image for Openshift OSS #############################################
FROM openshift-base AS openshift
ARG NGINX_VERSION
ARG IC_VERSION

LABEL image-version=ubi-nginx-$NGINX_VERSION-$IC_VERSION

RUN echo "[nginx]" >> /etc/yum.repos.d/nginx.repo \
	&& echo "name=nginx repo" >> /etc/yum.repos.d/nginx.repo \
	&& echo "baseurl=https://nginx.org/packages/mainline/rhel/8/\$basearch/" >> /etc/yum.repos.d/nginx.repo \
	&& echo "gpgcheck=0" >> /etc/yum.repos.d/nginx.repo \
	&& echo "enabled=1" >> /etc/yum.repos.d/nginx.repo \
	&& echo "module_hotfixes=true" >> /etc/yum.repos.d/nginx.repo \
	&& yum install -y nginx-${NGINX_VERSION} \
	&& rm /etc/yum.repos.d/nginx.repo

############################################# Base image for Openshift with NGINX Plus #############################################
FROM openshift-base AS openshift-plus
ARG IC_VERSION

ENV NGINX_PLUS_VERSION 23-1.el8.ngx

LABEL image-version=ubi-nginx-$NGINX_PLUS_VERSION-$IC_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	rpm --import https://nginx.org/keys/nginx_signing.key \
	&& echo "[nginx-plus]" >> /etc/yum.repos.d/nginx-plus-8.repo \
	&& echo "name=nginx-plus repo" >> /etc/yum.repos.d/nginx-plus-8.repo \
	&& echo "baseurl=https://plus-pkgs.nginx.com/centos/8/\$basearch/" >> /etc/yum.repos.d/nginx-plus-8.repo \
	&& echo "sslclientcert=/etc/ssl/nginx/nginx-repo.crt" >> /etc/yum.repos.d/nginx-plus-8.repo \
	&& echo "sslclientkey=/etc/ssl/nginx/nginx-repo.key" >> /etc/yum.repos.d/nginx-plus-8.repo \
	&& echo "gpgcheck=1" >> /etc/yum.repos.d/nginx-plus-8.repo \
	&& echo "enabled=1" >> /etc/yum.repos.d/nginx-plus-8.repo \
	&& yum install -y ca-certificates nginx-plus-${NGINX_PLUS_VERSION} \
	&& rm /etc/yum.repos.d/nginx-plus-8.repo

COPY --chown=nginx:0 internal/configs/oidc/* /etc/nginx/oidc/


############################################# Build image for Opentracing Builder #############################################
FROM debian as opentracing-builder
ARG NGINX_VERSION
ARG NGINX_OPENTRACING=0.10.0
ARG OPENTRACING_VERSION=1.6.0

RUN apt-get update && apt-get install -y -q --fix-missing --no-install-recommends \
	autoconf \
	automake \
	build-essential \
	cmake \
	git \
	libcurl4-openssl-dev \
	libcurl4-openssl-dev \
	libgeoip-dev \
	liblmdb-dev \
	libpcre3-dev \
	libprotobuf-dev \
	libssl-dev \
	libtool \
	libxml2-dev \
	libyajl-dev \
	pkgconf \
	protobuf-compiler \
	zlib1g-dev

# Get nginx to build against
RUN curl -sS -O -L http://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz && \
	tar zxvf nginx-${NGINX_VERSION}.tar.gz && rm -f nginx-${NGINX_VERSION}.tar.gz

RUN git clone --branch v${NGINX_OPENTRACING} https://github.com/opentracing-contrib/nginx-opentracing.git
RUN git clone --branch v${OPENTRACING_VERSION} https://github.com/opentracing/opentracing-cpp.git

RUN mkdir -p opentracing-cpp/.build && cd opentracing-cpp/.build && \
	cmake -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=OFF -DBUILD_STATIC_LIBS=OFF -DBUILD_MOCKTRACER=OFF -DENABLE_LINTING=OFF .. && \
	make && make install

RUN cd nginx-${NGINX_VERSION} && \
	./configure \
	--with-compat --with-debug \
	--add-dynamic-module=/nginx-opentracing/opentracing && \
	make modules


############################################# Build image for Trace downloader #############################################
FROM debian AS tracer-downloader

ARG JAEGER_VERSION=v0.4.2

RUN set -x \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y ca-certificates apt-transport-https wget \
	&& wget https://github.com/jaegertracing/jaeger-client-cpp/releases/download/${JAEGER_VERSION}/libjaegertracing_plugin.linux_amd64.so -O /usr/local/lib/libjaegertracing_plugin.so


############################################# Build image for Opentracing #############################################
FROM debian as opentracing
ARG NGINX_VERSION
ARG OPENTRACING_VERSION=1.6.0
ARG IC_VERSION

LABEL version=debian-nginx-opentracing-$NGINX_PLUS_VERSION-$IC_VERSION

COPY --from=opentracing-builder /nginx-${NGINX_VERSION}/objs/ngx_http_opentracing_module.so /usr/lib/nginx/modules/ngx_http_opentracing_module.so
COPY --from=opentracing-builder /usr/local/lib/libopentracing.so.${OPENTRACING_VERSION} /usr/local/lib/libopentracing.so.1
COPY --from=tracer-downloader /usr/local/lib/libjaegertracing_plugin.so /usr/local/lib/libjaegertracing_plugin.so

RUN ldconfig


############################################# Build image for Opentracing with NGINX Plus #############################################
FROM debian-plus as opentracing-plus
ARG IC_VERSION

LABEL version=debian-nginx-plus-opentracing-$NGINX_PLUS_VERSION-$IC_VERSION

ENV NGINX_OPENTRACING_MODULE_VERSION 23+0.9.0-1~buster

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	apt-get update && \
	apt-get install --no-install-recommends --no-install-suggests -y nginx-plus-module-opentracing=${NGINX_OPENTRACING_MODULE_VERSION} && \
	rm -rf /var/lib/apt/lists/*

COPY --from=tracer-downloader /usr/local/lib/libjaegertracing_plugin.so /usr/local/lib/libjaegertracing_plugin.so


############################################# Create common files, permissions and setcap #############################################
FROM $BUILD_OS as files

ARG PLUS

RUN mkdir -p /var/lib/nginx /etc/nginx/secrets /etc/nginx/stream-conf.d \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& chown -R nginx:0 /etc/nginx /var/cache/nginx /var/lib/nginx \
	&& rm -f /etc/nginx/conf.d/* /etc/apt/apt.conf.d/90nginx /etc/apt/sources.list.d/nginx-plus.list

COPY internal/configs/version1/nginx$PLUS.ingress.tmpl \
	internal/configs/version1/nginx$PLUS.tmpl \
	internal/configs/version2/nginx$PLUS.virtualserver.tmpl \
	internal/configs/version2/nginx$PLUS.transportserver.tmpl /

# Uncomment the line below if you would like to add the default.pem to the image
# and use it as a certificate and key for the default server
# ADD default.pem /etc/nginx/secrets/default

EXPOSE 80 443

ENTRYPOINT ["/nginx-ingress"]


############################################# Build nginx-ingress in golang container #############################################
FROM golang:1.16.2-alpine AS builder
ARG VERSION
ARG GIT_COMMIT
WORKDIR /go/src/github.com/nginxinc/kubernetes-ingress/nginx-ingress/cmd/nginx-ingress
COPY . /go/src/github.com/nginxinc/kubernetes-ingress/nginx-ingress/
RUN CGO_ENABLED=0 GOFLAGS='-mod=vendor' \
	go build -installsuffix cgo -ldflags "-w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o /nginx-ingress


############################################# Create image with nginx-ingress built in container #############################################
FROM files AS container
COPY --chown=nginx:0 --from=builder /nginx-ingress /

USER nginx


############################################# Create image with nginx-ingress built locally #############################################
FROM files AS local
COPY --chown=nginx:0 nginx-ingress /

USER nginx
