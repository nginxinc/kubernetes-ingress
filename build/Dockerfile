# syntax=docker/dockerfile:1.3
ARG BUILD_OS=debian
ARG NGINX_PLUS_VERSION=r25
ARG UBI_VERSION=8
ARG DOWNLOAD_TAG=edge

############################################# Base image for Debian #############################################
FROM nginx:1.21.3 AS debian

RUN apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y libcap2-bin \
	# temporary fix for CVE-2021-37750
	&& apt-get install -y libkrb5-3 libk5crypto3 \
	&& rm -rf /var/lib/apt/lists/* \
	&& echo $NGINX_VERSION > nginx_version


############################################# Base image for Alpine #############################################
# docker.io/library/nginx is a temporary workaround for Dependabot to see this as different from the one used in Debian
FROM docker.io/library/nginx:1.21.3-alpine AS alpine

RUN apk add --no-cache libcap \
	# temporary fix for CVE-2021-22945
	&& apk upgrade --no-cache curl libcurl


############################################# Base image for Alpine with NGINX Plus #############################################
FROM alpine:3.14 as alpine-plus
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/apk/cert.pem,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/apk/cert.key,mode=0644 \
	wget -nv -O /etc/apk/keys/nginx_signing.rsa.pub https://cs.nginx.com/static/keys/nginx_signing.rsa.pub \
	&& printf "%s\n" "https://pkgs.nginx.com/plus/alpine/v$(grep -E -o '^[0-9]+\.[0-9]+' /etc/alpine-release)/main" >> /etc/apk/repositories \
	&& apk add --no-cache libcap nginx-plus~${NGINX_PLUS_VERSION#r} nginx-plus-module-njs~${NGINX_PLUS_VERSION#r}


############################################# Base image for Debian with NGINX Plus #############################################
FROM debian:buster-slim AS debian-plus
ARG IC_VERSION
ARG NGINX_PLUS_VERSION
ARG BUILD_OS

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y ca-certificates gnupg curl apt-transport-https libcap2-bin \
	&& curl -sSL https://cs.nginx.com/static/keys/nginx_signing.key | gpg --dearmor > /etc/apt/trusted.gpg.d/nginx_signing.gpg \
	&& curl -sSL -o /etc/apt/apt.conf.d/90pkgs-nginx https://cs.nginx.com/static/files/90pkgs-nginx \
	&& printf "%s\n" "Acquire::https::pkgs.nginx.com::User-Agent \"k8s-ic-$IC_VERSION${BUILD_OS##debian-plus}-apt\";" >> /etc/apt/apt.conf.d/90pkgs-nginx \
	&& printf "%s\n" "deb https://pkgs.nginx.com/plus/${NGINX_PLUS_VERSION^^}/debian buster nginx-plus" > /etc/apt/sources.list.d/nginx-plus.list \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y nginx-plus nginx-plus-module-njs \
	&& apt-get purge --auto-remove -y apt-transport-https gnupg curl \
	&& rm -rf /var/lib/apt/lists/*


############################################# Base image for Debian with NGINX Plus and App Protect #############################################
FROM debian-plus as debian-plus-nap
ARG IC_VERSION
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y gnupg curl apt-transport-https \
	&& curl -sSL https://cs.nginx.com/static/keys/app-protect-security-updates.key | gpg --dearmor > /etc/apt/trusted.gpg.d/nginx_app_signing.gpg \
	&& printf "%s\n" "deb https://pkgs.nginx.com/app-protect/${NGINX_PLUS_VERSION^^}/debian buster nginx-plus" \
	"deb https://pkgs.nginx.com/app-protect-security-updates/debian buster nginx-plus" > /etc/apt/sources.list.d/nginx-app-protect.list \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y \
	nginx-plus-module-appprotect app-protect app-protect-attack-signatures app-protect-threat-campaigns \
	&& apt-get purge --auto-remove -y apt-transport-https gnupg curl \
	&& rm -rf /var/lib/apt/lists/* \
	&& rm /etc/apt/sources.list.d/nginx-app-protect.list

# Uncomment the lines below if you want to install a custom CA certificate
# COPY build/*.crt  /usr/local/share/ca-certificates/
# RUN update-ca-certificates


############################################# Base image for UBI 8 #############################################
FROM redhat/ubi8-minimal AS ubi-base-8


############################################# Base image for UBI 7 #############################################
FROM registry.access.redhat.com/ubi7/ubi AS ubi-base-7

RUN yum install -y microdnf


############################################# Base image for UBI #############################################
FROM ubi-base-${UBI_VERSION} AS ubi-base

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
LABEL name="NGINX Ingress Controller" \
	description="The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	summary="The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	io.openshift.tags="nginx,ingress-controller,ingress,controller,kubernetes,openshift"

RUN microdnf --nodocs install -y shadow-utils ca-certificates \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx

COPY --chown=nginx:0 LICENSE /licenses/


############################################# Base image for UBI OSS #############################################
FROM ubi-base AS ubi

RUN --mount=type=bind,from=debian,source=/nginx_version,target=/tmp/nginx_version \
	export NGINX_VERSION=$(</tmp/nginx_version) \
	&& rpm --import https://nginx.org/keys/nginx_signing.key \
	&& version=$(grep -E -o '[0-9]+\.[0-9]+' /etc/redhat-release | cut -d"." -f1) \
	&& printf "%s\n" "[nginx]" "name=nginx repo" \
	"baseurl=https://nginx.org/packages/mainline/centos/${version}/\$basearch/" \
	"gpgcheck=1" "enabled=1" "module_hotfixes=true" > /etc/yum.repos.d/nginx.repo \
	&& microdnf --nodocs install -y nginx-${NGINX_VERSION} \
	&& rm /etc/yum.repos.d/nginx.repo


############################################# Base image for UBI with NGINX Plus #############################################
FROM ubi-base AS ubi-plus
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	rpm --import https://cs.nginx.com/static/keys/nginx_signing.key \
	&& curl -sS "https://cs.nginx.com/static/files/nginx-plus-$(grep -E -o '[0-9]+\.[0-9]+' /etc/redhat-release | cut -d"." -f1).repo" | tr 0 1 > /etc/yum.repos.d/nginx-plus.repo \
	&& microdnf --nodocs install -y nginx-plus-${NGINX_PLUS_VERSION} nginx-plus-module-njs-${NGINX_PLUS_VERSION}


############################################# Base image for UBI with NGINX Plus and App Protect #############################################
FROM ubi-plus as ubi-plus-nap
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	--mount=type=secret,id=rhel_license,dst=/tmp/rhel_license,mode=0644 \
	source /tmp/rhel_license \
	&& subscription-manager register --org=${RHEL_ORGANIZATION} --activationkey=${RHEL_ACTIVATION_KEY} || true \
	&& subscription-manager attach \
	&& set -x \
	&& curl -sS https://cs.nginx.com/static/files/app-protect-7.repo > /etc/yum.repos.d/app-protect-7.repo \
	&& subscription-manager repos --enable rhel-7-server-optional-rpms --enable rhel-7-server-extras-rpms \
	&& rpm -ivh https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm \
	&& yum clean all \
	&& yum install -y nginx-plus-module-appprotect-${NGINX_PLUS_VERSION} $(repoquery app-protect-${NGINX_PLUS_VERSION#r}*) app-protect-attack-signatures app-protect-threat-campaigns \
	&& rm /etc/yum.repos.d/app-protect-7.repo \
	&& subscription-manager unregister

# Uncomment the lines below if you want to install a custom CA certificate
# COPY build/*.crt  /etc/pki/ca-trust/source/anchors/
# RUN update-ca-trust extract


############################################# Base image containing libs for Opentracing #############################################
FROM opentracing/nginx-opentracing:nginx-1.21.3 as opentracing-lib


############################################# Build image for Opentracing #############################################
FROM debian as opentracing

RUN --mount=type=bind,from=opentracing-lib,target=/tmp/ cp -av /tmp/usr/local/lib/libopentracing.so* /tmp/usr/local/lib/libjaegertracing*so* /tmp/usr/local/lib/libzipkin*so* /tmp/usr/local/lib/libdd*so* /tmp/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& cp -av /tmp/usr/lib/nginx/modules/ngx_http_opentracing_module.so /usr/lib/nginx/modules/ \
	&& ldconfig


############################################# Build image for Opentracing with NGINX Plus #############################################
FROM debian-plus as opentracing-plus
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y nginx-plus-module-opentracing-${NGINX_PLUS_VERSION} \
	&& rm -rf /var/lib/apt/lists/*

RUN --mount=type=bind,from=opentracing-lib,target=/tmp/ cp -av /tmp/usr/local/lib/libjaegertracing*so* /tmp/usr/local/lib/libzipkin*so* /tmp/usr/local/lib/libdd*so* /tmp/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& ldconfig


############################################# Create common files, permissions and setcap #############################################
FROM ${BUILD_OS} as common

ARG BUILD_OS
ARG IC_VERSION
ARG GIT_COMMIT
ARG DATE
ARG TARGETPLATFORM

# copy oidc files on plus build
RUN --mount=target=/tmp [ -n "${BUILD_OS##*plus*}" ] && exit 0; mkdir -p etc/nginx/oidc/ && cp -a /tmp/internal/configs/oidc/* /etc/nginx/oidc/

# run only on nap build
RUN --mount=target=/tmp [ -n "${BUILD_OS##*nap*}" ] && exit 0; mkdir -p /etc/nginx/waf/nac-policies /etc/nginx/waf/nac-logconfs /etc/nginx/waf/nac-usersigs /var/log/app_protect /opt/app_protect \
	&& chown -R nginx:0 /etc/app_protect /usr/share/ts /var/log/app_protect/ /opt/app_protect/ /var/log/nginx/ \
	&& touch /etc/nginx/waf/nac-usersigs/index.conf \
	&& printf "%s\n" "MODULE = ALL;" "LOG_LEVEL = TS_CRIT;" "FILE = 2;" > /etc/app_protect/bd/logger.cfg \
	&& printf "%s\n" "[config_set_compiler]" "log_level=fatal" >> /etc/app_protect/tools/asm_logging.conf \
	&& for v in \
	asm_config_server \
	lock_factory \
	bd_agent \
	import_export_policy \
	set_active \
	; do sed -i "/\[$v/a log_level=fatal" "/etc/app_protect/tools/asm_logging.conf" \
	; done \
	&& cp -a /tmp/build/log-default.json /etc/nginx

RUN --mount=target=/tmp mkdir -p /var/lib/nginx /etc/nginx/secrets /etc/nginx/stream-conf.d \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& setcap -v 'cap_net_bind_service=+ep' /usr/sbin/nginx 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& [ -z "${BUILD_OS##*plus*}" ] && PLUS=-plus; cp -a /tmp/internal/configs/version1/nginx$PLUS.ingress.tmpl /tmp/internal/configs/version1/nginx$PLUS.tmpl \
	/tmp/internal/configs/version2/nginx$PLUS.virtualserver.tmpl /tmp/internal/configs/version2/nginx$PLUS.transportserver.tmpl / \
	&& chown -R nginx:0 /etc/nginx /etc/nginx/secrets /var/cache/nginx /var/lib/nginx /*.tmpl \
	&& rm -f /etc/nginx/conf.d/* /etc/apt/apt.conf.d/90pkgs-nginx /etc/apt/sources.list.d/nginx-plus.list

# Uncomment the line below if you would like to add the default.pem to the image
# and use it as a certificate and key for the default server
# ADD default.pem /etc/nginx/secrets/default

EXPOSE 80 443

STOPSIGNAL SIGTERM
ENTRYPOINT ["/nginx-ingress"]
USER nginx

LABEL org.opencontainers.image.version="${IC_VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.opencontainers.image.created="${DATE}"
LABEL org.nginx.kic.image.build.target="${TARGETPLATFORM}"
LABEL org.nginx.kic.image.build.os="${BUILD_OS}"
LABEL org.nginx.kic.image.build.nginx.vesion="${NGINX_PLUS_VERSION}${NGINX_VERSION}"


############################################# Build nginx-ingress in golang container #############################################
FROM golang:1.17-alpine AS builder
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

LABEL org.nginx.kic.image.build.version="container"

COPY --chown=nginx:0 --from=builder /nginx-ingress /


############################################# Create image with nginx-ingress built locally #############################################
FROM common AS local

LABEL org.nginx.kic.image.build.version="local"

COPY --chown=nginx:0 ./nginx-ingress /


############################################# Create image with nginx-ingress built by GoReleaser #############################################
FROM common AS goreleaser
ARG TARGETARCH
ARG TARGETVARIANT

LABEL org.nginx.kic.image.build.version="goreleaser"

COPY --chown=nginx:0 ./dist/kubernetes-ingress_linux_$TARGETARCH${TARGETVARIANT:+_7}/nginx-ingress /


############################################# Create image with nginx-ingress built by GoReleaser for AWS Marketplace #############################################
FROM common AS aws
ARG TARGETARCH

LABEL org.nginx.kic.image.build.version="aws"

COPY --chown=nginx:0 ./dist/aws_linux_$TARGETARCH/nginx-ingress /


############################################# Create image with nginx-ingress extracted from image on Docker Hub #############################################
FROM nginx/nginx-ingress:${DOWNLOAD_TAG} as kic

FROM common as download

LABEL org.nginx.kic.image.build.version="binaries"

COPY --from=kic /nginx-ingress /
