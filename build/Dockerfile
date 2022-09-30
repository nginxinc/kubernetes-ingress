# syntax=docker/dockerfile:1.4
ARG BUILD_OS=debian
ARG NGINX_PLUS_VERSION=R27
ARG DOWNLOAD_TAG=edge
ARG DEBIAN_VERSION=bullseye-slim


############################################# Base images containing libs for Opentracing #############################################
FROM opentracing/nginx-opentracing:nginx-1.23.1 as opentracing-lib
FROM opentracing/nginx-opentracing:nginx-1.23.1-alpine as alpine-opentracing-lib


############################################# Base image for Debian #############################################
FROM nginx:1.23.1 AS debian

RUN --mount=type=bind,from=opentracing-lib,target=/tmp/ot/ \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y libcap2-bin \
	# temp fix for CVE-2022-40674
	&& apt-get install -y libexpat1 \
	&& rm -rf /var/lib/apt/lists/* \
	&& cp -av /tmp/ot/usr/local/lib/libopentracing.so* /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& cp -av /tmp/ot/usr/lib/nginx/modules/ngx_http_opentracing_module.so /usr/lib/nginx/modules/ \
	&& ldconfig \
	&& echo $NGINX_VERSION > nginx_version


############################################# Base image for Alpine #############################################
FROM nginx:1.23.1-alpine AS alpine

RUN --mount=type=bind,from=alpine-opentracing-lib,target=/tmp/ot/ \
	apk add --no-cache libcap libstdc++ \
	# temp fix for CVE-2022-3209 and CVE-2022-35252
	&& apk upgrade --no-cache libxml2 curl libcurl \
	&& cp -av /tmp/ot/usr/local/lib/libopentracing.so* /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& cp -av /tmp/ot/usr/lib/nginx/modules/ngx_http_opentracing_module.so /usr/lib/nginx/modules/ \
	&& ldconfig /usr/local/lib/


############################################# Base image for Alpine with NGINX Plus #############################################
FROM alpine:3.16 as alpine-plus
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/apk/cert.pem,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/apk/cert.key,mode=0644 \
	--mount=type=bind,from=alpine-opentracing-lib,target=/tmp/ot/ \
	wget -nv -O /etc/apk/keys/nginx_signing.rsa.pub https://cs.nginx.com/static/keys/nginx_signing.rsa.pub \
	&& printf "%s\n" "https://pkgs.nginx.com/plus/${NGINX_PLUS_VERSION}/alpine/v$(grep -E -o '^[0-9]+\.[0-9]+' /etc/alpine-release)/main" >> /etc/apk/repositories \
	&& apk add --no-cache libcap nginx-plus nginx-plus-module-njs nginx-plus-module-opentracing libcurl \
	&& cp -av /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& ldconfig /usr/local/lib/


############################################# Base image for Debian with NGINX Plus #############################################
FROM debian:${DEBIAN_VERSION} AS debian-plus
ARG IC_VERSION
ARG NGINX_PLUS_VERSION
ARG BUILD_OS

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	--mount=type=bind,from=opentracing-lib,target=/tmp/ot/ \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y ca-certificates gnupg curl apt-transport-https libcap2-bin \
	&& curl -fsSL https://cs.nginx.com/static/keys/nginx_signing.key | gpg --dearmor > /etc/apt/trusted.gpg.d/nginx_signing.gpg \
	&& curl -fsSL -o /etc/apt/apt.conf.d/90pkgs-nginx https://cs.nginx.com/static/files/90pkgs-nginx \
	&& DEBIAN_VERSION=$(awk -F '=' '/^VERSION_CODENAME=/ {print $2}' /etc/os-release) \
	&& printf "%s\n" "Acquire::https::pkgs.nginx.com::User-Agent \"k8s-ic-$IC_VERSION${BUILD_OS##debian-plus}-apt\";" >> /etc/apt/apt.conf.d/90pkgs-nginx \
	&& printf "%s\n" "deb https://pkgs.nginx.com/plus/${NGINX_PLUS_VERSION}/debian ${DEBIAN_VERSION} nginx-plus" > /etc/apt/sources.list.d/nginx-plus.list \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y nginx-plus nginx-plus-module-njs nginx-plus-module-opentracing libcurl4 \
	&& apt-get purge --auto-remove -y apt-transport-https gnupg curl \
	&& cp -av /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& ldconfig \
	&& rm -rf /var/lib/apt/lists/*


############################################# Base image for Debian with NGINX Plus and App Protect WAF/DoS #############################################
FROM debian-plus as debian-plus-nap
ARG NGINX_PLUS_VERSION
ARG NAP_MODULES

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y gnupg curl apt-transport-https \
	&& DEBIAN_VERSION=$(awk -F '=' '/^VERSION_CODENAME=/ {print $2}' /etc/os-release) \
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	curl -fsSL https://cs.nginx.com/static/keys/app-protect-security-updates.key | gpg --dearmor > /etc/apt/trusted.gpg.d/nginx_app_signing.gpg \
	&& printf "%s\n" "deb https://pkgs.nginx.com/app-protect/${NGINX_PLUS_VERSION}/debian ${DEBIAN_VERSION} nginx-plus" \
	"deb https://pkgs.nginx.com/app-protect-security-updates/debian ${DEBIAN_VERSION} nginx-plus" > /etc/apt/sources.list.d/nginx-app-protect.list \
	&& apt-get update \
	# temp fix for CVE-2022-37434 and DLA-3112-1
	&& apt-get install zlib1g libbz2-1.0 \
	&& apt-get install --no-install-recommends --no-install-suggests -y app-protect app-protect-attack-signatures app-protect-threat-campaigns \
	&& apt-get purge --auto-remove -y curl; \
	fi \
	&& if [ -z "${NAP_MODULES##*dos*}" ]; then \
	printf "%s\n" "deb https://pkgs.nginx.com/app-protect-dos/${NGINX_PLUS_VERSION}/debian ${DEBIAN_VERSION} nginx-plus" > /etc/apt/sources.list.d/nginx-app-protect-dos.list \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y app-protect-dos; \
	fi \
	&& apt-get purge --auto-remove -y apt-transport-https gnupg \
	&& rm -rf /var/lib/apt/lists/* \
	&& rm /etc/apt/sources.list.d/nginx-app-protect*.list

# Uncomment the lines below if you want to install a custom CA certificate
# COPY build/*.crt  /usr/local/share/ca-certificates/
# RUN update-ca-certificates


############################################# Base image for UBI #############################################
FROM redhat/ubi8 AS ubi-base
ARG IC_VERSION

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
LABEL name="NGINX Ingress Controller" \
	maintainer="kubernetes@nginx.com" \
	vendor="NGINX Inc" \
	version="${IC_VERSION}" \
	release="1" \
	summary="The Ingress Controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	description="The Ingress Controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	io.k8s.description="The NGINX Ingress Controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	io.openshift.tags="nginx,ingress-controller,ingress,controller,kubernetes,openshift"

RUN dnf --nodocs install -y shadow-utils ca-certificates \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx

COPY --link --chown=101:0 LICENSE /licenses/


############################################# Base image for UBI OSS #############################################
FROM ubi-base AS ubi

RUN --mount=type=bind,from=debian,source=/nginx_version,target=/tmp/nginx_version \
	export NGINX_VERSION=$(</tmp/nginx_version) \
	&& rpm --import https://nginx.org/keys/nginx_signing.key \
	&& version=$(grep -E -o '[0-9]+\.[0-9]+' /etc/redhat-release | cut -d"." -f1) \
	&& printf "%s\n" "[nginx]" "name=nginx repo" \
	"baseurl=https://nginx.org/packages/mainline/centos/${version}/\$basearch/" \
	"gpgcheck=1" "enabled=1" "module_hotfixes=true" > /etc/yum.repos.d/nginx.repo \
	&& dnf --nodocs install -y nginx-${NGINX_VERSION} \
	&& rm /etc/yum.repos.d/nginx.repo


############################################# Base image for UBI with NGINX Plus #############################################
FROM ubi-base AS ubi-plus
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	rpm --import https://cs.nginx.com/static/keys/nginx_signing.key \
	&& curl -fsSL "https://cs.nginx.com/static/files/nginx-plus-$(grep -E -o '[0-9]+\.[0-9]+' /etc/redhat-release | cut -d"." -f1).repo" | tr 0 1 > /etc/yum.repos.d/nginx-plus.repo \
	&& sed -i "0,/centos/s;;${NGINX_PLUS_VERSION}/centos;" /etc/yum.repos.d/nginx-plus.repo \
	&& dnf --nodocs install -y nginx-plus nginx-plus-module-njs


############################################# Base image for UBI with NGINX Plus and App Protect WAF/DoS #############################################
FROM ubi-plus as ubi-plus-nap
ARG NGINX_PLUS_VERSION
ARG NAP_MODULES

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	--mount=type=secret,id=rhel_license,dst=/tmp/rhel_license,mode=0644 \
	source /tmp/rhel_license \
	&& subscription-manager register --org=${RHEL_ORGANIZATION} --activationkey=${RHEL_ACTIVATION_KEY} || true \
	&& subscription-manager attach \
	&& dnf config-manager --set-enabled codeready-builder-for-rhel-8-x86_64-rpms \
	&& dnf --nodocs install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm \
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	curl -fsSL https://cs.nginx.com/static/files/app-protect-8.repo > /etc/yum.repos.d/app-protect-8.repo; \
	sed -i "0,/centos/s;;${NGINX_PLUS_VERSION}/centos;" /etc/yum.repos.d/app-protect-8.repo; \
	dnf --nodocs install -y app-protect app-protect-attack-signatures app-protect-threat-campaigns; \
	fi \
	&& if [ -z "${NAP_MODULES##*dos*}" ]; then \
	curl -fsSL https://cs.nginx.com/static/files/app-protect-dos-8.repo > /etc/yum.repos.d/app-protect-dos-8.repo; \
	sed -i "0,/centos/s;;${NGINX_PLUS_VERSION}/centos;" /etc/yum.repos.d/app-protect-dos-8.repo; \
	dnf --nodocs install -y app-protect-dos; \
	fi \
	&& rm /etc/yum.repos.d/app-protect*.repo \
	&& subscription-manager unregister \
	&& dnf clean all && rm -rf /var/cache/dnf

# Uncomment the lines below if you want to install a custom CA certificate
# COPY build/*.crt  /etc/pki/ca-trust/source/anchors/
# RUN update-ca-trust extract


############################################# Create common files, permissions and setcap #############################################
FROM ${BUILD_OS} as common

ARG BUILD_OS
ARG IC_VERSION
ARG GIT_COMMIT
ARG TARGETPLATFORM
ARG NAP_MODULES=none

# copy oidc files on plus build
RUN --mount=type=bind,target=/tmp [ -n "${BUILD_OS##*plus*}" ] && exit 0; mkdir -p /etc/nginx/oidc/ && cp -a /tmp/internal/configs/oidc/* /etc/nginx/oidc/

# run only on nap waf build
RUN --mount=type=bind,target=/tmp [ -n "${NAP_MODULES##*waf*}" ] && exit 0; mkdir -p /etc/nginx/waf/nac-policies /etc/nginx/waf/nac-logconfs /etc/nginx/waf/nac-usersigs /var/log/app_protect /opt/app_protect \
	&& chown -R 101:0 /etc/app_protect /usr/share/ts /var/log/app_protect/ /opt/app_protect/ /var/log/nginx/ \
	&& touch /etc/nginx/waf/nac-usersigs/index.conf \
	&& cp -a /tmp/build/log-default.json /etc/nginx

# run only on nap dos build
RUN [ -n "${NAP_MODULES##*dos*}" ] && exit 0; mkdir -p /root/app_protect_dos /etc/nginx/dos/policies /etc/nginx/dos/logconfs /shared/cores /var/log/adm /var/run/adm \
	&& chmod 777 /shared/cores /var/log/adm /var/run/adm /etc/app_protect_dos

RUN --mount=type=bind,target=/tmp mkdir -p /var/lib/nginx /etc/nginx/secrets /etc/nginx/stream-conf.d \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& setcap -v 'cap_net_bind_service=+ep' /usr/sbin/nginx 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& [ -z "${BUILD_OS##*plus*}" ] && PLUS=-plus; cp -a /tmp/internal/configs/version1/nginx$PLUS.ingress.tmpl /tmp/internal/configs/version1/nginx$PLUS.tmpl \
	/tmp/internal/configs/version2/nginx$PLUS.virtualserver.tmpl /tmp/internal/configs/version2/nginx$PLUS.transportserver.tmpl / \
	&& chown -R 101:0 /etc/nginx /var/cache/nginx /var/lib/nginx /*.tmpl \
	&& rm -f /etc/nginx/conf.d/* /etc/apt/apt.conf.d/90pkgs-nginx /etc/apt/sources.list.d/nginx-plus.list

# Uncomment the line below if you would like to add the default.pem to the image
# and use it as a certificate and key for the default server
# ADD default.pem /etc/nginx/secrets/default

EXPOSE 80 443

STOPSIGNAL SIGTERM
ENTRYPOINT ["/nginx-ingress"]
# 101 is nginx
USER 101

LABEL org.opencontainers.image.version="${IC_VERSION}"
LABEL org.opencontainers.image.revision="${GIT_COMMIT}"
LABEL org.nginx.kic.image.build.target="${TARGETPLATFORM}"
LABEL org.nginx.kic.image.build.os="${BUILD_OS}"
LABEL org.nginx.kic.image.build.nginx.version="${NGINX_PLUS_VERSION}${NGINX_VERSION}"


############################################# Build nginx-ingress in golang container #############################################
FROM golang:1.19-alpine AS builder
ARG IC_VERSION
ARG TARGETARCH

WORKDIR /go/src/github.com/nginxinc/kubernetes-ingress/
RUN apk add --no-cache git
RUN --mount=type=bind,target=/go/src/github.com/nginxinc/kubernetes-ingress/ --mount=type=cache,target=/root/.cache/go-build \
	go mod download
RUN --mount=type=bind,target=/go/src/github.com/nginxinc/kubernetes-ingress/ --mount=type=cache,target=/root/.cache/go-build \
	CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w -X main.version=${IC_VERSION}" -o /nginx-ingress github.com/nginxinc/kubernetes-ingress/cmd/nginx-ingress


############################################# Create image with nginx-ingress built in container #############################################
FROM common AS container

LABEL org.nginx.kic.image.build.version="container"

COPY --link --from=builder --chown=101:0 /nginx-ingress /


############################################# Create image with nginx-ingress built locally #############################################
FROM common AS local

LABEL org.nginx.kic.image.build.version="local"

COPY --link --chown=101:0 nginx-ingress /


############################################# Create image with nginx-ingress built by GoReleaser #############################################
FROM common AS goreleaser
ARG TARGETARCH

LABEL org.nginx.kic.image.build.version="goreleaser"

COPY --link --chown=101:0 dist/kubernetes-ingress_linux_$TARGETARCH*/nginx-ingress /


############################################# Create image with nginx-ingress built by GoReleaser for AWS Marketplace #############################################
FROM common AS aws
ARG TARGETARCH

LABEL org.nginx.kic.image.build.version="aws"

COPY --link --chown=101:0 dist/aws_linux_$TARGETARCH*/nginx-ingress /


############################################# Create image with nginx-ingress extracted from image on Docker Hub #############################################
FROM nginx/nginx-ingress:${DOWNLOAD_TAG} as kic

FROM common as download

LABEL org.nginx.kic.image.build.version="binaries"

COPY --link --from=kic --chown=101:0 /nginx-ingress /
