ARG GOLANG_CONTAINER=golang:latest

FROM nginx:1.19.2 AS base

# forward nginx access and error logs to stdout and stderr of the ingress
# controller process
RUN ln -sf /proc/1/fd/1 /var/log/nginx/access.log \
	&& ln -sf /proc/1/fd/1 /var/log/nginx/stream-access.log \
	&& ln -sf /proc/1/fd/2 /var/log/nginx/error.log

RUN mkdir -p /var/lib/nginx \
	&& mkdir -p /etc/nginx/secrets \
	&& mkdir -p /etc/nginx/stream-conf.d \
	&& apt-get update \
	&& apt-get install -y libcap2-bin \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx \
	&& setcap 'cap_net_bind_service=+ep' /usr/sbin/nginx-debug \
	&& chown -R nginx:0 /etc/nginx \
	&& chown -R nginx:0 /var/cache/nginx \
	&& chown -R nginx:0 /var/lib/nginx \
	&& apt-get remove --purge -y libcap2-bin \
	&& rm /etc/nginx/conf.d/* \
	&& rm -rf /var/lib/apt/lists/*

COPY internal/configs/version1/nginx.ingress.tmpl \
	internal/configs/version1/nginx.tmpl \
	internal/configs/version2/nginx.virtualserver.tmpl \
	internal/configs/version2/nginx.transportserver.tmpl /

# Uncomment the line below if you would like to add the default.pem to the image
# and use it as a certificate and key for the default server
# ADD default.pem /etc/nginx/secrets/default

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
	go build -installsuffix cgo -ldflags "-w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o /nginx-ingress


FROM base AS container
COPY --from=builder /nginx-ingress /
