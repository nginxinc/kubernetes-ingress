LOCAL_ROOT = $(git rev-parse --show-toplevel)
SHELL = /bin/bash
THEME_MODULE = gitlab.com/f5/nginx/controller/poc/f5-hugo
# export THEME_PATH="</local/path/to/theme>"
THEME_PATH = $(LOCAL_THEME_PATH)
THEME_BRANCH = development

.PHONY: all all-local clean hugo-mod build-production build-staging hugo-server-drafts hugo-server

all: hugo-mod build-production

all-local: clean hugo-mod build-production

clean:
	rm -rf ${LOCAL_ROOT}/public

hugo-mod:
	hugo mod clean
	rm -rf _vendor
	hugo mod get
	hugo mod vendor

build-production:
	hugo --gc -e production
	
build-staging:
	hugo --gc -e staging

hugo-server-drafts:
	hugo server -e production -b 127.0.0.1/nginx-ingress-controller/ -D --disableFastRender

hugo-server:
	hugo server -e production -b 127.0.0.1/nginx-ingress-controller/ --disableFastRender

netlify:
	netlify build --context=branch-deploy
	netlify deploy -d public

replace-theme:
	go mod edit -replace "$(THEME_MODULE)"="$(THEME_PATH)"