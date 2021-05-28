# syntax=docker/dockerfile:1.2
ARG BUILD_OS=debian
ARG NGINX_PLUS_VERSION=r24

############################################# Base image for Debian #############################################
FROM nginx:1.21.0 AS debian

RUN apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y libcap2-bin \
	&& rm -rf /var/lib/apt/lists/*


############################################# Base image for Alpine #############################################
# docker.io/library/nginx is a temporary workaround for Dependabot to see this as different from the one used in Debian
FROM docker.io/library/nginx:1.21.0-alpine AS alpine

RUN apk add --no-cache libcap


############################################# Base image for Alpine with NGINX Plus #############################################
FROM alpine:3.13 as alpine-plus
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/apk/cert.pem,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/apk/cert.key,mode=0644 \
	wget -nv -O /etc/apk/keys/nginx_signing.rsa.pub https://cs.nginx.com/static/keys/nginx_signing.rsa.pub \
	&& printf "%s\n" "https://pkgs.nginx.com/plus/alpine/v$(grep -E -o '^[0-9]+\.[0-9]+' /etc/alpine-release)/main" >> /etc/apk/repositories \
	&& apk add --no-cache libcap nginx-plus~${NGINX_PLUS_VERSION#r} nginx-plus-module-njs~${NGINX_PLUS_VERSION#r}

COPY --chown=nginx:0 internal/configs/oidc/* /etc/nginx/oidc/


############################################# Base image for Debian with NGINX Plus #############################################
FROM debian:buster-slim AS debian-plus
ARG IC_VERSION
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y ca-certificates gnupg wget apt-transport-https libcap2-bin \
	&& wget -nv https://cs.nginx.com/static/keys/nginx_signing.key \
	&& gpg --no-default-keyring --keyring nginx_keyring.gpg --import nginx_signing.key \
	&& gpg --no-default-keyring --keyring nginx_keyring.gpg --export > /etc/apt/trusted.gpg.d/nginx_signing.gpg \
	&& wget -nv -P /etc/apt/apt.conf.d https://cs.nginx.com/static/files/90pkgs-nginx \
	&& echo "Acquire::https::pkgs.nginx.com::User-Agent  \"k8s-ic-$IC_VERSION-apt\";" >> /etc/apt/apt.conf.d/90pkgs-nginx \
	&& printf "%s\n" "deb https://pkgs.nginx.com/plus/debian buster nginx-plus" > /etc/apt/sources.list.d/nginx-plus.list \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y \
	nginx-plus-${NGINX_PLUS_VERSION} nginx-plus-module-njs-${NGINX_PLUS_VERSION} \
	&& apt-get purge --auto-remove -y apt-transport-https gnupg wget \
	&& rm -rf /var/lib/apt/lists/*

COPY --chown=nginx:0 internal/configs/oidc/* /etc/nginx/oidc/


############################################# Base image for Debian with NGINX Plus and App Protect #############################################
FROM debian-plus as debian-plus-nap
ARG IC_VERSION
ARG NGINX_PLUS_VERSION

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y gnupg wget apt-transport-https \
	&& wget -nv https://cs.nginx.com/static/keys/app-protect-security-updates.key \
	&& gpg --no-default-keyring --keyring app_protect_keyring.gpg --import app-protect-security-updates.key \
	&& gpg --no-default-keyring --keyring app_protect_keyring.gpg --export > /etc/apt/trusted.gpg.d/nginx_app_signing.gpg \
	&& sed -i "$ d" /etc/apt/apt.conf.d/90pkgs-nginx \
	&& echo "Acquire::https::pkgs.nginx.com::User-Agent  \"k8s-ic-$IC_VERSION-nap-apt\";" >> /etc/apt/apt.conf.d/90pkgs-nginx \
	&& printf "%s\n" "deb https://pkgs.nginx.com/app-protect/debian buster nginx-plus" \
	"deb https://pkgs.nginx.com/app-protect-security-updates/debian buster nginx-plus" > /etc/apt/sources.list.d/nginx-app-protect.list \
	&& apt-get update \
	# searching apt-cache for the latest version of NAP packages compatible with the $NGINX_PLUS_VERSION
	&& module_version=$(apt-cache showpkg nginx-plus-module-appprotect | awk -v ver="nginx-plus-$NGINX_PLUS_VERSION" '{ if ($6 == ver) {print $1; exit}}') \
	engine_version=$(apt-cache showpkg app-protect | awk -v ver="$module_version" '{ if ($1 == ver && $3 == "nginx-plus-module-appprotect") {print substr($NF, 1, length($NF)-1); exit}}') \
	plugin=$(apt-cache showpkg nginx-plus-module-appprotect | awk -v ver="nginx-plus-$NGINX_PLUS_VERSION" '{ if ($6 == ver) {print substr($NF, 1, length($NF)-1); exit}}') \
	&& apt-get install --no-install-recommends --no-install-suggests -y \
	nginx-plus-module-appprotect=${module_version} \
	app-protect-plugin=${plugin} \
	app-protect-engine=${engine_version} \
	app-protect-compiler=${engine_version} \
	app-protect=${module_version} \
	&& apt-get install --no-install-recommends --no-install-suggests -y app-protect-attack-signatures app-protect-threat-campaigns \
	&& apt-get purge --auto-remove -y apt-transport-https gnupg wget \
	&& rm -rf /var/lib/apt/lists/* \
	&& rm /etc/apt/sources.list.d/nginx-app-protect.list \
	&& mkdir -p /etc/nginx/waf/nac-policies /etc/nginx/waf/nac-logconfs /etc/nginx/waf/nac-usersigs /var/log/app_protect /opt/app_protect \
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


############################################# Base image for UBI #############################################
FROM registry.access.redhat.com/ubi8-minimal AS ubi-base

LABEL name="NGINX Ingress Controller" \
	description="The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	summary="The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	io.openshift.tags="nginx,ingress-controller,ingress,controller,kubernetes,openshift"

RUN microdnf --nodocs --enablerepo=ubi-8-baseos install -y shadow-utils \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx

COPY --chown=nginx:0 LICENSE /licenses/


############################################# Base image for UBI OSS #############################################
FROM ubi-base AS ubi

ARG NGINX_VERSION

RUN rpm --import https://nginx.org/keys/nginx_signing.key \
	&& printf "%s\n" "[nginx]" "name=nginx repo" \
	"baseurl=https://nginx.org/packages/mainline/centos/8/\$basearch/" \
	"gpgcheck=1" "enabled=1" "module_hotfixes=true" > /etc/yum.repos.d/nginx.repo \
	&& microdnf --setopt=install_weak_deps=0 --nodocs install -y nginx-${NGINX_VERSION} \
	&& rm /etc/yum.repos.d/nginx.repo


############################################# Base image for UBI with NGINX Plus #############################################
FROM ubi-base AS ubi-plus
ARG NGINX_PLUS_VERSION

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	rpm --import https://cs.nginx.com/static/keys/nginx_signing.key \
	&& curl -sS https://cs.nginx.com/static/files/nginx-plus-8.repo | tr 0 1 > /etc/yum.repos.d/nginx-plus-8.repo \
	&& microdnf --setopt=install_weak_deps=0 --nodocs install -y nginx-plus-${NGINX_PLUS_VERSION} nginx-plus-module-njs-${NGINX_PLUS_VERSION} \
	&& rm /etc/yum.repos.d/nginx-plus-8.repo

COPY --chown=nginx:0 internal/configs/oidc/* /etc/nginx/oidc/


############################################# Build image for Opentracing Builder #############################################
FROM debian as opentracing-builder
ARG NGINX_OPENTRACING=0.10.0
ARG OPENTRACING_VERSION=1.6.0

RUN apt-get update && apt-get install -y -q --fix-missing --no-install-recommends \
	autoconf \
	automake \
	build-essential \
	cmake \
	git \
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
RUN curl -sS -O -L http://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz \
	&& tar zxvf nginx-${NGINX_VERSION}.tar.gz && rm -f nginx-${NGINX_VERSION}.tar.gz \
	&& git clone --branch v${NGINX_OPENTRACING} https://github.com/opentracing-contrib/nginx-opentracing.git \
	&& git clone --branch v${OPENTRACING_VERSION} https://github.com/opentracing/opentracing-cpp.git

WORKDIR /opentracing-cpp/.build
RUN	cmake -DCMAKE_BUILD_TYPE=Release -DBUILD_TESTING=OFF -DBUILD_STATIC_LIBS=OFF -DBUILD_MOCKTRACER=OFF -DENABLE_LINTING=OFF .. && \
	make && make install

WORKDIR /nginx-${NGINX_VERSION}
RUN ./configure \
	--with-compat --with-debug \
	--add-dynamic-module=/nginx-opentracing/opentracing && \
	make modules


############################################# Build image for Trace downloader #############################################
FROM debian AS tracer-downloader

ARG JAEGER_VERSION=v0.4.2

RUN apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y ca-certificates apt-transport-https wget \
	&& wget -nv https://github.com/jaegertracing/jaeger-client-cpp/releases/download/${JAEGER_VERSION}/libjaegertracing_plugin.linux_amd64.so -O /usr/local/lib/libjaegertracing_plugin.so


############################################# Build image for Opentracing #############################################
FROM debian as opentracing
ARG OPENTRACING_VERSION=1.6.0

COPY --from=opentracing-builder /nginx-${NGINX_VERSION}/objs/ngx_http_opentracing_module.so /usr/lib/nginx/modules/ngx_http_opentracing_module.so
COPY --from=opentracing-builder /usr/local/lib/libopentracing.so.${OPENTRACING_VERSION} /usr/local/lib/libopentracing.so.1
COPY --from=tracer-downloader /usr/local/lib/libjaegertracing_plugin.so /usr/local/lib/libjaegertracing_plugin.so

RUN ldconfig


############################################# Build image for Opentracing with NGINX Plus #############################################
FROM debian-plus as opentracing-plus
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y nginx-plus-module-opentracing-${NGINX_PLUS_VERSION} \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=tracer-downloader /usr/local/lib/libjaegertracing_plugin.so /usr/local/lib/libjaegertracing_plugin.so


############################################# Create common files, permissions and setcap #############################################
FROM $BUILD_OS as common

ARG PLUS
ARG BUILD_OS
ARG IC_VERSION
ARG GIT_COMMIT
ARG DATE

RUN mkdir -p /var/lib/nginx /etc/nginx/secrets /etc/nginx/stream-conf.d \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& setcap -v 'cap_net_bind_service=+ep' /usr/sbin/nginx 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& chown -R nginx:0 /etc/nginx /etc/nginx/secrets /var/cache/nginx /var/lib/nginx \
	&& rm -f /etc/nginx/conf.d/* /etc/apt/apt.conf.d/90pkgs-nginx /etc/apt/sources.list.d/nginx-plus.list

COPY --chown=nginx:0 internal/configs/version1/nginx$PLUS.ingress.tmpl \
	internal/configs/version1/nginx$PLUS.tmpl \
	internal/configs/version2/nginx$PLUS.virtualserver.tmpl \
	internal/configs/version2/nginx$PLUS.transportserver.tmpl /

# Uncomment the line below if you would like to add the default.pem to the image
# and use it as a certificate and key for the default server
# ADD default.pem /etc/nginx/secrets/default

EXPOSE 80 443

ENTRYPOINT ["/nginx-ingress"]
USER nginx

LABEL org.opencontainers.image.title="NGINX Ingress Controller for Kubernetes"
LABEL org.opencontainers.image.documentation="https://docs.nginx.com/nginx-ingress-controller"
LABEL org.opencontainers.image.source="https://github.com/nginxinc/kubernetes-ingress"
LABEL org.opencontainers.image.vendor="NGINX Inc <kubernetes@nginx.com>"
LABEL org.opencontainers.image.licenses="Apache-2.0"
LABEL org.opencontainers.image.version="${IC_VERSION}-${BUILD_OS}-${NGINX_PLUS_VERSION}${NGINX_VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${DATE}"


############################################# Build nginx-ingress in golang container #############################################
FROM golang:1.16-alpine AS builder
ARG IC_VERSION
ARG GIT_COMMIT
ARG DATE
ARG TARGETARCH

WORKDIR /go/src/github.com/nginxinc/kubernetes-ingress/nginx-ingress/cmd/nginx-ingress
COPY go.mod go.sum /go/src/github.com/nginxinc/kubernetes-ingress/nginx-ingress/
RUN go mod download
COPY . /go/src/github.com/nginxinc/kubernetes-ingress/nginx-ingress/
RUN CGO_ENABLED=0 GO111MODULE=on GOOS=linux GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -X main.version=${IC_VERSION} -X main.commit=${GIT_COMMIT} -X main.date=${DATE}" -o /nginx-ingress


############################################# Create image with nginx-ingress built in container #############################################
FROM common AS container
COPY --chown=nginx:0 --from=builder /nginx-ingress /


############################################# Create image with nginx-ingress built locally #############################################
FROM common AS local
COPY --chown=nginx:0 nginx-ingress /


############################################# Create image with nginx-ingress built by GoReleaser #############################################
FROM common AS goreleaser
ARG TARGETARCH
ARG TARGETVARIANT

COPY --chown=nginx:0 dist/kubernetes-ingress_linux_$TARGETARCH${TARGETVARIANT:+_7}/nginx-ingress /
