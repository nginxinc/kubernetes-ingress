ARG GOLANG_CONTAINER=golang:latest

FROM registry.access.redhat.com/ubi8/ubi:8.1 AS base

LABEL name="NGINX Ingress Controller" \
      description="The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
      summary="The Ingress controller is an application that runs in a cluster and configures an HTTP load balancer according to Ingress resources." \
      io.openshift.tags="nginx,ingress-controller,ingress,controller,kubernetes,openshift" \
      maintainer="NGINX Docker Maintainers <docker-maint@nginx.com>" \
      vendor="NGINX Inc <kubernetes@nginx.com>"

ENV NGINX_VERSION=1.19.2

RUN set -x \
	&& groupadd --system --gid 101 nginx \
	&& useradd --system --gid nginx --no-create-home --home-dir /nonexistent --comment "nginx user" --shell /bin/false --uid 101 nginx \
	&& echo "[nginx]" >> /etc/yum.repos.d/nginx.repo \
	&& echo "name=nginx repo" >> /etc/yum.repos.d/nginx.repo \
	&& echo "baseurl=https://nginx.org/packages/mainline/rhel/8/\$basearch/" >> /etc/yum.repos.d/nginx.repo \
	&& echo "gpgcheck=0" >> /etc/yum.repos.d/nginx.repo \
	&& echo "enabled=1" >> /etc/yum.repos.d/nginx.repo \
	&& echo "module_hotfixes=true" >> /etc/yum.repos.d/nginx.repo \
	&& yum install -y nginx-${NGINX_VERSION} \
	&& mkdir -p /var/lib/nginx \
	&& mkdir -p /etc/nginx/secrets \
	&& mkdir -p /etc/nginx/stream-conf.d \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& chown -R nginx:0 /etc/nginx \
	&& chown -R nginx:0 /var/cache/nginx \
	&& chown -R nginx:0 /var/lib/nginx \
	&& rm /etc/yum.repos.d/nginx.repo \
	&& rm /etc/nginx/conf.d/*

# forward nginx access and error logs to stdout and stderr of the ingress
# controller process
RUN ln -sf /proc/1/fd/1 /var/log/nginx/access.log \
	&& ln -sf /proc/1/fd/1 /var/log/nginx/stream-access.log \
	&& ln -sf /proc/1/fd/2 /var/log/nginx/error.log

COPY internal/configs/version1/nginx.ingress.tmpl \
	internal/configs/version1/nginx.tmpl \
	internal/configs/version2/nginx.virtualserver.tmpl \
	internal/configs/version2/nginx.transportserver.tmpl /

# Uncomment the line below if you would like to add the default.pem to the image
# and use it as a certificate and key for the default server
# ADD default.pem /etc/nginx/secrets/default

RUN mkdir licenses
COPY LICENSE /licenses

USER nginx

ENTRYPOINT ["/nginx-ingress"]


FROM base AS local
COPY nginx-ingress /


FROM $GOLANG_CONTAINER AS builder
ARG VERSION
ARG GIT_COMMIT
WORKDIR /go/src/github.com/nginxinc/kubernetes-ingress/nginx-ingress/cmd/nginx-ingress
COPY . /go/src/github.com/nginxinc/kubernetes-ingress/nginx-ingress/
RUN CGO_ENABLED=0 GOFLAGS='-mod=vendor' \
	go build -installsuffix cgo -ldflags "-w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o /tmp/nginx-ingress


FROM base AS container
COPY --from=builder /tmp/nginx-ingress /
