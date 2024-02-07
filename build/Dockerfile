# syntax=docker/dockerfile:1.6
ARG BUILD_OS=debian
ARG NGINX_PLUS_VERSION=R31
ARG DOWNLOAD_TAG=edge
ARG DEBIAN_FRONTEND=noninteractive


############################################# Base images containing libs for Opentracing and FIPS #############################################
FROM opentracing/nginx-opentracing:nginx-1.25.3@sha256:2e0268d3cd31fe047c2fe566f29731865b0c99cc99b579c6584b23cd3c7830ef as opentracing-lib
FROM opentracing/nginx-opentracing:nginx-1.25.3-alpine@sha256:37c7de3a46ca05428450b1c64bfb2a4d2f9c1835860cef427928fcf11c178f0e as alpine-opentracing-lib
FROM ghcr.io/nginxinc/alpine-fips:0.1.0-alpine3.17@sha256:f00b3f266422feaaac7b733b46903bd19eb1cd1caa6991131576f5f767db76f8 as alpine-fips-3.17
FROM ghcr.io/nginxinc/alpine-fips:0.1.2-alpine3.19@sha256:67595f52053f328fd731bf493d8dc72c1fd956e5f1a2f9a579cc78c5f5f50daa as alpine-fips-3.19


############################################# Base image for Alpine #############################################
FROM nginx:1.25.3-alpine@sha256:156d75f07c59b2fd59d3d1470631777943bb574135214f0a90c7bb82bde916da AS alpine

RUN --mount=type=bind,from=alpine-opentracing-lib,target=/tmp/ot/ \
	apk add --no-cache libcap libstdc++ \
	&& apk upgrade --no-cache -U \
	&& cp -av /tmp/ot/usr/local/lib/libopentracing.so* /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& cp -av /tmp/ot/usr/lib/nginx/modules/ngx_http_opentracing_module.so /usr/lib/nginx/modules/ \
	&& ldconfig /usr/local/lib/ \
	&& apk cache clean


############################################# Base image for Debian #############################################
FROM nginx:1.25.3@sha256:8b4c32060a41e8c07e4b33c2e2695510c729314d84b2b71a1c5d7002aaf0b5ad AS debian

RUN --mount=type=bind,from=opentracing-lib,target=/tmp/ot/ \
	apt-get update \
	&& apt-get upgrade -y \
	&& apt-get install --no-install-recommends --no-install-suggests -y libcap2-bin \
	&& rm -rf /var/lib/apt/lists/* \
	&& cp -av /tmp/ot/usr/local/lib/libopentracing.so* /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& cp -av /tmp/ot/usr/lib/nginx/modules/ngx_http_opentracing_module.so /usr/lib/nginx/modules/ \
	&& ldconfig


############################################# Base image for UBI #############################################
FROM nginxcontrib/nginx:1.25.3-ubi@sha256:4a3e891705687db11a02a3ba37a1ce42b01349d49198a956576787ab4a3a7a0c AS ubi
ARG IC_VERSION

LABEL name="NGINX Ingress Controller" \
	maintainer="kubernetes@nginx.com" \
	vendor="NGINX Inc" \
	version="${IC_VERSION}" \
	release="1" \
	summary="The Ingress Controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	description="The Ingress Controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	io.k8s.description="The NGINX Ingress Controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
	io.openshift.tags="nginx,ingress-controller,ingress,controller,kubernetes,openshift"

COPY --link --chown=101:0 LICENSE /licenses/
RUN microdnf update -y \
	&& microdnf clean all


############################################# NGINX files for NGINX Plus #############################################
FROM scratch as nginx-files
ARG IC_VERSION
ARG BUILD_OS
ARG NGINX_PLUS_VERSION

# the following links can be replaced with local files if needed, i.e. ADD --chown=101:0 <local_file> <container_file>
ADD --link --chown=101:0 https://cs.nginx.com/static/files/90pkgs-nginx 90pkgs-nginx
ADD --link --chown=101:0 https://cs.nginx.com/static/keys/nginx_signing.key nginx_signing.key
ADD --link --chown=101:0 https://cs.nginx.com/static/keys/nginx_signing.rsa.pub nginx_signing.rsa.pub
ADD --link --chown=101:0 https://cs.nginx.com/static/keys/app-protect-security-updates.key app-protect-security-updates.key
ADD --link --chown=101:0 https://cs.nginx.com/static/keys/app-protect-security-updates.rsa.pub app-protect-security-updates.rsa.pub
ADD --link --chown=101:0 https://cs.nginx.com/static/files/nginx-plus-8.repo nginx-plus-8.repo
ADD --link --chown=101:0 https://cs.nginx.com/static/files/plus-9.repo nginx-plus-9.repo
ADD --link --chown=101:0 https://cs.nginx.com/static/files/app-protect-8.repo app-protect-8.repo
ADD --link --chown=101:0 https://cs.nginx.com/static/files/app-protect-9.repo app-protect-9.repo
ADD --link --chown=101:0 https://cs.nginx.com/static/files/app-protect-dos-8.repo app-protect-dos-8.repo
ADD --link --chown=101:0 https://raw.githubusercontent.com/nginxinc/k8s-common/main/files/plus-debian-11.repo debian-plus-11.sources
ADD --link --chown=101:0 https://raw.githubusercontent.com/nginxinc/k8s-common/main/files/plus-debian-12.repo debian-plus-12.sources
ADD --link --chown=101:0 https://raw.githubusercontent.com/nginxinc/k8s-common/main/files/nap-waf-debian-11.repo nap-waf-11.sources
ADD --link --chown=101:0 https://raw.githubusercontent.com/nginxinc/k8s-common/main/files/nap-dos-debian-11.repo nap-dos-11.sources

RUN --mount=from=busybox:musl,src=/bin/,dst=/bin/ printf "%s\n" "Acquire::https::pkgs.nginx.com::User-Agent k8s-ic-$IC_VERSION${BUILD_OS##debian-plus}-apt;" >> 90pkgs-nginx \
	&& printf "%s\n" "user_agent=k8s-ic-$IC_VERSION${BUILD_OS##ubi*plus}-dnf" | tee -a nginx-plus-*.repo \
	&& sed -i -e "s;%VERSION%;${NGINX_PLUS_VERSION};g" *.sources \
	&& sed -i -e "y/0/1/" -e "1,8s;/centos;/${NGINX_PLUS_VERSION}/centos;" *.repo \
	&& echo HTTP_USER_AGENT="k8s-ic-$IC_VERSION${BUILD_OS##alpine-plus}-apk" > user_agent


############################################# Base image for Alpine with NGINX Plus #############################################
FROM alpine:3.19@sha256:c5b1261d6d3e43071626931fc004f70149baeba2c8ec672bd4f27761f8e1ad6b as alpine-plus
ARG NGINX_PLUS_VERSION

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/apk/cert.pem,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/apk/cert.key,mode=0644 \
	--mount=type=bind,from=alpine-opentracing-lib,target=/tmp/ot/ \
	--mount=type=bind,from=nginx-files,src=nginx_signing.rsa.pub,target=/etc/apk/keys/nginx_signing.rsa.pub \
	--mount=type=bind,from=nginx-files,src=user_agent,target=/tmp/user_agent \
	export $(cat /tmp/user_agent) \
	&& printf "%s\n" "https://pkgs.nginx.com/plus/${NGINX_PLUS_VERSION}/alpine/v$(grep -E -o '^[0-9]+\.[0-9]+' /etc/alpine-release)/main" >> /etc/apk/repositories \
	&& apk upgrade --no-cache -U \
	&& apk add --no-cache nginx-plus nginx-plus-module-njs nginx-plus-module-opentracing nginx-plus-module-fips-check libcap libcurl \
	&& cp -av /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& ldconfig /usr/local/lib/ \
	&& apk cache clean


############################################# Base image for Alpine with NGINX Plus and FIPS #############################################
FROM alpine-plus as alpine-plus-fips

RUN --mount=type=bind,from=alpine-fips-3.19,target=/tmp/fips/ \
	mkdir -p /usr/ssl \
	&& cp -av /tmp/fips/usr/lib/ossl-modules/fips.so /usr/lib/ossl-modules/fips.so \
	&& cp -av /tmp/fips/usr/ssl/fipsmodule.cnf /usr/ssl/fipsmodule.cnf \
	&& cp -av /tmp/fips/etc/ssl/openssl.cnf /etc/ssl/openssl.cnf


############################################# Base image for Alpine with NGINX Plus, App Protect WAF and FIPS #############################################
FROM alpine:3.17@sha256:6e94b5cda2d6fd57d85abf81e81dabaea97a5885f919da676cc19d3551da4061 as alpine-plus-nap-fips
ARG NGINX_PLUS_VERSION

RUN --mount=type=bind,from=alpine-fips-3.17,target=/tmp/fips/ \
	--mount=type=secret,id=nginx-repo.crt,dst=/etc/apk/cert.pem,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/apk/cert.key,mode=0644 \
	--mount=type=bind,from=alpine-opentracing-lib,target=/tmp/ot/ \
	--mount=type=bind,from=nginx-files,src=app-protect-security-updates.rsa.pub,target=/etc/apk/keys/app-protect-security-updates.rsa.pub \
	--mount=type=bind,from=nginx-files,src=nginx_signing.rsa.pub,target=/etc/apk/keys/nginx_signing.rsa.pub \
	printf "%s\n" "https://pkgs.nginx.com/plus/${NGINX_PLUS_VERSION}/alpine/v$(grep -E -o '^[0-9]+\.[0-9]+' /etc/alpine-release)/main" >> /etc/apk/repositories \
	&& printf "%s\n" "https://pkgs.nginx.com/app-protect/${NGINX_PLUS_VERSION}/alpine/v$(grep -E -o '^[0-9]+\.[0-9]+' /etc/alpine-release)/main" >> /etc/apk/repositories \
	&& printf "%s\n" "https://pkgs.nginx.com/app-protect-security-updates/alpine/v$(grep -E -o '^[0-9]+\.[0-9]+' /etc/alpine-release)/main" >> /etc/apk/repositories \
	&& apk upgrade --no-cache -U \
	&& apk add --no-cache libcap-utils libcurl nginx-plus nginx-plus-module-njs nginx-plus-module-opentracing nginx-plus-module-fips-check \
	&& mkdir -p /usr/ssl \
	&& cp -av /tmp/fips/usr/lib/ossl-modules/fips.so /usr/lib/ossl-modules/fips.so \
	&& cp -av /tmp/fips/usr/ssl/fipsmodule.cnf /usr/ssl/fipsmodule.cnf \
	&& cp -av /tmp/fips/etc/ssl/openssl.cnf /etc/ssl/openssl.cnf \
	&& cp -av /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& ldconfig /usr/local/lib/ \
	&& apk add --no-cache app-protect app-protect-attack-signatures app-protect-threat-campaigns


############################################# Base image for Debian with NGINX Plus #############################################
FROM debian:12-slim@sha256:7802002798b0e351323ed2357ae6dc5a8c4d0a05a57e7f4d8f97136151d3d603 AS debian-plus

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	--mount=type=bind,from=opentracing-lib,target=/tmp/ot/ \
	--mount=type=bind,from=nginx-files,src=nginx_signing.key,target=/tmp/nginx_signing.key \
	--mount=type=bind,from=nginx-files,src=90pkgs-nginx,target=/etc/apt/apt.conf.d/90pkgs-nginx \
	--mount=type=bind,from=nginx-files,src=debian-plus-12.sources,target=/etc/apt/sources.list.d/nginx-plus.sources \
	apt-get update \
	&& apt-get upgrade -y \
	&& apt-get install --no-install-recommends --no-install-suggests -y sq ca-certificates libcap2-bin libcurl4 \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx \
	&& sq dearmor -o /usr/share/keyrings/nginx-archive-keyring.gpg /tmp/nginx_signing.key \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y nginx-plus nginx-plus-module-njs nginx-plus-module-opentracing nginx-plus-module-fips-check \
	&& apt-get purge --auto-remove -y sq \
	&& cp -av /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& ldconfig \
	&& rm -rf /var/lib/apt/lists/*


############################################# Base image for Debian with NGINX Plus and App Protect WAF/DoS #############################################
FROM debian:12-slim@sha256:7802002798b0e351323ed2357ae6dc5a8c4d0a05a57e7f4d8f97136151d3d603 as debian-plus-nap
ARG NAP_MODULES

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	--mount=type=bind,from=opentracing-lib,target=/tmp/ot/ \
	--mount=type=bind,from=nginx-files,src=nginx_signing.key,target=/tmp/nginx_signing.key \
	--mount=type=bind,from=nginx-files,src=app-protect-security-updates.key,target=/tmp/app-protect-security-updates.key \
	--mount=type=bind,from=nginx-files,src=90pkgs-nginx,target=/etc/apt/apt.conf.d/90pkgs-nginx \
	--mount=type=bind,from=nginx-files,src=debian-plus-11.sources,target=/etc/apt/sources.list.d/nginx-plus.sources \
	--mount=type=bind,from=nginx-files,src=nap-waf-11.sources,target=/tmp/app-protect.sources \
	--mount=type=bind,from=nginx-files,src=nap-dos-11.sources,target=/tmp/app-protect-dos.sources \
	## the code below is duplicated from the debian-plus image because NAP doesn't support debian 12
	if [ -z "${NAP_MODULES##*waf*}" ]; then \
	cp /tmp/app-protect.sources /etc/apt/sources.list.d/app-protect.sources; \
	fi \
	&& if [ -z "${NAP_MODULES##*dos*}" ]; then \
	cp /tmp/app-protect-dos.sources /etc/apt/sources.list.d/app-protect-dos.sources; \
	fi \
	&& apt-get update \
	&& apt-get upgrade -y \
	&& apt-get install --no-install-recommends --no-install-suggests -y ca-certificates sq \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx \
	&& sq dearmor -o /usr/share/keyrings/nginx-archive-keyring.gpg /tmp/nginx_signing.key \
	&& sq dearmor -o /usr/share/keyrings/app-protect-archive-keyring.gpg /tmp/app-protect-security-updates.key \
	&& apt-get update \
	&& apt-get install --no-install-recommends --no-install-suggests -y nginx-plus nginx-plus-module-njs nginx-plus-module-opentracing nginx-plus-module-fips-check libcap2-bin libcurl4 \
	## end of duplicated code
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	apt-get install --no-install-recommends --no-install-suggests -y app-protect app-protect-attack-signatures app-protect-threat-campaigns; \
	fi \
	&& if [ -z "${NAP_MODULES##*dos*}" ]; then \
	apt-get install --no-install-recommends --no-install-suggests -y app-protect-dos; \
	fi \
	&& apt-get purge --auto-remove -y sq \
	## the code below is duplicated from the debian-plus image because NAP doesn't support debian 12
	&& cp -av /tmp/ot/usr/local/lib/libjaegertracing*so* /tmp/ot/usr/local/lib/libzipkin*so* /tmp/ot/usr/local/lib/libdd*so* /tmp/ot/usr/local/lib/libyaml*so* /usr/local/lib/ \
	&& ldconfig \
	## end of duplicated code
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	rm -f /etc/apt/sources.list.d/app-protect.sources; \
	fi \
	&& if [ -z "${NAP_MODULES##*dos*}" ]; then \
	rm -f /etc/apt/sources.list.d/app-protect-dos.sources; \
	fi \
	&& rm -rf /var/lib/apt/lists/*

# Uncomment the lines below if you want to install a custom CA certificate
# COPY build/*.crt /usr/local/share/ca-certificates/
# RUN update-ca-certificates


############################################# Base image for UBI with NGINX Plus #############################################
FROM redhat/ubi9-minimal@sha256:582e18f13291d7c686ec4e6e92d20b24c62ae0fc72767c46f30a69b1a6198055 AS ubi-plus

SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	--mount=type=bind,from=nginx-files,src=nginx_signing.key,target=/tmp/nginx_signing.key \
	--mount=type=bind,from=nginx-files,src=nginx-plus-9.repo,target=/etc/yum.repos.d/nginx-plus.repo \
	microdnf --nodocs install -y shadow-utils \
	&& microdnf update -y \
	&& cat /etc/yum.repos.d/nginx-plus.repo \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx \
	&& rpm --import /tmp/nginx_signing.key \
	&& microdnf --nodocs install -y nginx-plus nginx-plus-module-njs nginx-plus-module-fips-check \
	&& microdnf remove -y shadow-utils \
	&& microdnf clean all


############################################# Base image for UBI with NGINX Plus and App Protect WAF #############################################
FROM redhat/ubi9@sha256:1fafb0905264413501df60d90a92ca32df8a2011cbfb4876ddff5ceb20c8f165 as ubi-9-plus-nap
ARG NAP_MODULES

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	--mount=type=secret,id=rhel_license,dst=/tmp/rhel_license,mode=0644 \
	--mount=type=bind,from=nginx-files,src=nginx_signing.key,target=/tmp/nginx_signing.key \
	--mount=type=bind,from=nginx-files,src=nginx-plus-9.repo,target=/etc/yum.repos.d/nginx-plus.repo,rw \
	--mount=type=bind,from=nginx-files,src=app-protect-security-updates.key,target=/tmp/app-protect-security-updates.key \
	--mount=type=bind,from=nginx-files,src=app-protect-9.repo,target=/tmp/app-protect-9.repo \
	source /tmp/rhel_license \
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	cp /tmp/app-protect-9.repo /etc/yum.repos.d/app-protect-9.repo; \
	fi \
	## the code below is duplicated from the ubi-plus image because NAP WAF doesn't support UBI minimal versions
	&& dnf --nodocs install -y shadow-utils ca-certificates \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx \
	&& rpm --import /tmp/nginx_signing.key \
	&& dnf --nodocs install -y nginx-plus nginx-plus-module-njs nginx-plus-module-fips-check \
	## end of duplicated code
	## fix for CVEs
	&& dnf upgrade -y curl dbus libcap libssh platform-python python3-requests libxml2 systemd sqlite-libs dnf-plugin-subscription-manager dmidecode subscription-manager-rhsm-certificates glibc subscription-manager \
	&& sed -i 's/\(def in_container():\)/\1\n    return False/g' /usr/lib64/python*/*-packages/rhsm/config.py \
	&& subscription-manager register --org=${RHEL_ORGANIZATION} --activationkey=${RHEL_ACTIVATION_KEY} || true \
	&& subscription-manager attach \
	&& dnf config-manager --set-enabled codeready-builder-for-rhel-9-x86_64-rpms \
	&& dnf --nodocs install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-9.noarch.rpm \
	&& rpm --import /tmp/app-protect-security-updates.key \
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	dnf --nodocs install -y app-protect app-protect-attack-signatures app-protect-threat-campaigns; \
	fi \
	# fix for CVEs
	&& dnf upgrade -y curl ncurses \
	&& subscription-manager unregister \
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	rm -f /etc/yum.repos.d/app-protect-9.repo; \
	fi \
	&& dnf clean all

############################################# Base image for UBI with NGINX Plus and App Protect WAF & DoS #############################################
FROM redhat/ubi8@sha256:23d8dfd08024fdfa34b168c297d8c74a1dc58675b02b3418925932df123b755c as ubi-8-plus-nap
ARG NAP_MODULES

RUN --mount=type=secret,id=nginx-repo.crt,dst=/etc/ssl/nginx/nginx-repo.crt,mode=0644 \
	--mount=type=secret,id=nginx-repo.key,dst=/etc/ssl/nginx/nginx-repo.key,mode=0644 \
	--mount=type=secret,id=rhel_license,dst=/tmp/rhel_license,mode=0644 \
	--mount=type=bind,from=nginx-files,src=nginx_signing.key,target=/tmp/nginx_signing.key \
	--mount=type=bind,from=nginx-files,src=nginx-plus-8.repo,target=/etc/yum.repos.d/nginx-plus.repo,rw \
	--mount=type=bind,from=nginx-files,src=app-protect-security-updates.key,target=/tmp/app-protect-security-updates.key \
	--mount=type=bind,from=nginx-files,src=app-protect-8.repo,target=/tmp/app-protect-8.repo \
	--mount=type=bind,from=nginx-files,src=app-protect-dos-8.repo,target=/tmp/app-protect-dos-8.repo \
	source /tmp/rhel_license \
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	cp /tmp/app-protect-8.repo /etc/yum.repos.d/app-protect-8.repo; \
	fi \
	&& if [ -z "${NAP_MODULES##*dos*}" ]; then \
	cp /tmp/app-protect-dos-8.repo /etc/yum.repos.d/app-protect-dos-8.repo; \
	fi \
	## the code below is duplicated from the ubi-plus image because NAP DOS doesn't support UBI 9 and minimal versions
	&& dnf --nodocs install -y shadow-utils ca-certificates \
	&& dnf update -y \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx \
	&& rpm --import /tmp/nginx_signing.key \
	&& dnf --nodocs install -y nginx-plus nginx-plus-module-njs nginx-plus-module-fips-check \
	## end of duplicated code
	## fix for CVEs
	&& dnf upgrade -y curl dbus libcap libssh platform-python python3-requests libxml2 systemd sqlite-libs dnf-plugin-subscription-manager dmidecode subscription-manager-rhsm-certificates glibc subscription-manager \
	&& sed -i 's/\(def in_container():\)/\1\n    return False/g' /usr/lib64/python*/*-packages/rhsm/config.py \
	&& subscription-manager register --org=${RHEL_ORGANIZATION} --activationkey=${RHEL_ACTIVATION_KEY} || true \
	&& subscription-manager attach \
	&& dnf config-manager --set-enabled codeready-builder-for-rhel-8-x86_64-rpms \
	&& dnf --nodocs install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-8.noarch.rpm \
	&& rpm --import /tmp/app-protect-security-updates.key \
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	dnf --nodocs install -y app-protect app-protect-attack-signatures app-protect-threat-campaigns; \
	fi \
	&& if [ -z "${NAP_MODULES##*dos*}" ]; then \
	dnf --nodocs install -y app-protect-dos; \
	fi \
	# fix for CVEs
	&& dnf upgrade -y curl ncurses \
	&& subscription-manager unregister \
	&& if [ -z "${NAP_MODULES##*waf*}" ]; then \
	rm -f /etc/yum.repos.d/app-protect-8.repo; \
	fi \
	&& if [ -z "${NAP_MODULES##*dos*}" ]; then \
	rm -f /etc/yum.repos.d/app-protect-dos-8.repo; \
	fi \
	&& dnf clean all

# Uncomment the lines below if you want to install a custom CA certificate
# COPY build/*.crt /etc/pki/ca-trust/source/anchors/
# RUN update-ca-trust extract


############################################# Create common files, permissions and setcap #############################################
FROM ${BUILD_OS} as common

ARG BUILD_OS
ARG IC_VERSION
ARG TARGETPLATFORM
ARG NAP_MODULES=none

# copy oidc files on plus build
RUN --mount=type=bind,target=/tmp [ -n "${BUILD_OS##*plus*}" ] && exit 0; mkdir -p /etc/nginx/oidc/ && cp -a /tmp/internal/configs/oidc/* /etc/nginx/oidc/

# run only on nap waf build
RUN --mount=type=bind,target=/tmp [ -n "${NAP_MODULES##*waf*}" ] && exit 0; mkdir -p /etc/nginx/waf/nac-policies /etc/nginx/waf/nac-logconfs /etc/nginx/waf/nac-usersigs /var/log/app_protect /opt/app_protect \
	&& chown -R 101:0 /etc/app_protect /usr/share/ts /var/log/app_protect/ /opt/app_protect/ \
	&& chmod -R g=u /etc/app_protect /usr/share/ts /var/log/app_protect/ /opt/app_protect/ \
	&& touch /etc/nginx/waf/nac-usersigs/index.conf \
	&& cp -a /tmp/build/log-default.json /etc/nginx

# run only on nap dos build
RUN [ -n "${NAP_MODULES##*dos*}" ] && exit 0; mkdir -p /root/app_protect_dos /etc/nginx/dos/policies /etc/nginx/dos/logconfs /shared/cores /var/log/adm /var/run/adm \
	&& chmod 777 /shared/cores /var/log/adm /var/run/adm /etc/app_protect_dos

RUN --mount=type=bind,target=/tmp mkdir -p /var/lib/nginx /etc/nginx/secrets /etc/nginx/stream-conf.d \
	&& setcap 'cap_net_bind_service=+eip' /usr/sbin/nginx 'cap_net_bind_service=+eip' /usr/sbin/nginx-debug \
	&& setcap -v 'cap_net_bind_service=+eip' /usr/sbin/nginx 'cap_net_bind_service=+eip' /usr/sbin/nginx-debug \
	&& [ -z "${BUILD_OS##*plus*}" ] && PLUS=-plus; cp -a /tmp/internal/configs/version1/nginx$PLUS.ingress.tmpl /tmp/internal/configs/version1/nginx$PLUS.tmpl \
	/tmp/internal/configs/version2/nginx$PLUS.virtualserver.tmpl /tmp/internal/configs/version2/nginx$PLUS.transportserver.tmpl / \
	&& chown -R 101:0 /etc/nginx /var/cache/nginx /var/lib/nginx /var/log/nginx /*.tmpl \
	&& chmod -R g=u /etc/nginx /var/cache/nginx /var/lib/nginx /var/log/nginx /*.tmpl \
	&& rm -f /etc/nginx/conf.d/*

# Uncomment the line below if you would like to add the default.pem to the image
# and use it as a certificate and key for the default server
# ADD default.pem /etc/nginx/secrets/default

EXPOSE 80 443

STOPSIGNAL SIGTERM
ENTRYPOINT ["/nginx-ingress"]
# 101 is nginx
USER 101

LABEL org.opencontainers.image.version="${IC_VERSION}" \
	org.opencontainers.image.documentation=https://docs.nginx.com/nginx-ingress-controller \
	org.opencontainers.image.vendor="NGINX Inc <kubernetes@nginx.com>" \
	org.nginx.kic.image.build.target="${TARGETPLATFORM}" \
	org.nginx.kic.image.build.os="${BUILD_OS}" \
	org.nginx.kic.image.build.nginx.version="${NGINX_PLUS_VERSION}${NGINX_VERSION}"


############################################# Build nginx-ingress in golang container #############################################
FROM golang:1.22-alpine@sha256:8e96e6cff6a388c2f70f5f662b64120941fcd7d4b89d62fec87520323a316bd9 AS builder
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
# root is required for `setcap` invocation
USER 0
RUN setcap 'cap_net_bind_service=+ep' /nginx-ingress && setcap -v 'cap_net_bind_service=+ep' /nginx-ingress
# 101 is nginx, defined above
USER 101


############################################# Create image with nginx-ingress built locally #############################################
FROM common AS local

LABEL org.nginx.kic.image.build.version="local"

COPY --link --chown=101:0 nginx-ingress /
# root is required for `setcap` invocation
USER 0
RUN setcap 'cap_net_bind_service=+ep' /nginx-ingress && setcap -v 'cap_net_bind_service=+ep' /nginx-ingress
# 101 is nginx, defined above
USER 101


############################################# Create image with nginx-ingress built by GoReleaser #############################################
FROM common AS goreleaser
ARG TARGETARCH

LABEL org.nginx.kic.image.build.version="goreleaser"

COPY --link --chown=101:0 dist/kubernetes-ingress_linux_${TARGETARCH}*/nginx-ingress /
# root is required for `setcap` invocation
USER 0
RUN setcap 'cap_net_bind_service=+ep' /nginx-ingress && setcap -v 'cap_net_bind_service=+ep' /nginx-ingress
# 101 is nginx, defined above
USER 101


############################################# Create image with nginx-ingress built by GoReleaser for AWS Marketplace #############################################
FROM common AS aws
ARG TARGETARCH
ARG NAP_MODULES_AWS

LABEL org.nginx.kic.image.build.version="aws"

COPY --link --chown=101:0 dist/aws*${NAP_MODULES_AWS}_linux_${TARGETARCH}*/nginx-ingress /
# root is required for `setcap` invocation
USER 0
RUN setcap 'cap_net_bind_service=+ep' /nginx-ingress && setcap -v 'cap_net_bind_service=+ep' /nginx-ingress
# 101 is nginx, defined above
USER 101


############################################# Create image with nginx-ingress extracted from image on Docker Hub #############################################
FROM nginx/nginx-ingress:${DOWNLOAD_TAG} as kic

FROM common as download

LABEL org.nginx.kic.image.build.version="binaries"

COPY --link --from=kic --chown=101:0 /nginx-ingress /
