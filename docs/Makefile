HUGO?=hugo
# the officially recommended unofficial docker image
HUGO_IMG?=hugomods/hugo:0.115.3

THEME_MODULE = github.com/nginxinc/nginx-hugo-theme
## Pulls the current theme version from the Netlify settings
THEME_VERSION = $(NGINX_THEME_VERSION)
NETLIFY_DEPLOY_URL = ${DEPLOY_PRIME_URL}

# if there's no local hugo, fallback to docker
ifeq (, $(shell ${HUGO} version 2> /dev/null))
ifeq (, $(shell docker version 2> /dev/null))
    $(error Docker and Hugo are not installed. Hugo (<0.91) or Docker are required to build the local preview.)
else
    HUGO=docker run --rm -it -v ${CURDIR}:/src -p 1313:1313 ${HUGO_IMG} hugo
endif
endif

MARKDOWNLINT?=markdownlint
MARKDOWNLINT_IMG?=ghcr.io/igorshubovych/markdownlint-cli:latest

# if there's no local markdownlint, fallback to docker
ifeq (, $(shell ${MARKDOWNLINT} version 2> /dev/null))
ifeq (, $(shell docker version 2> /dev/null))
ifneq (, $(shell $(NETLIFY) "true"))
    $(error Docker and markdownlint are not installed. markdownlint or Docker are required to lint.)
endif
else
    MARKDOWNLINT=docker run --rm -i -v ${CURDIR}:/src --workdir /src ${MARKDOWNLINT_IMG}
endif
endif

MARKDOWNLINKCHECK?=markdown-link-check
MARKDOWNLINKCHECK_IMG?=ghcr.io/tcort/markdown-link-check:stable
# if there's no local markdown-link-check, fallback to docker
ifeq (, $(shell ${MARKDOWNLINKCHECK} --version 2> /dev/null))
ifeq (, $(shell docker version 2> /dev/null))
ifneq (, $(shell $(NETLIFY) "true"))
    $(error Docker and markdown-link-check are not installed. markdown-link-check or Docker are required to check links.)
endif
else
    MARKDOWNLINKCHECK=docker run --rm -it -v ${CURDIR}:/site --workdir /site ${MARKDOWNLINKCHECK_IMG}
endif
endif

.PHONY: all all-staging all-dev all-local clean hugo-mod build-production build-staging build-dev docs-drafts docs deploy-preview

all: hugo-mod build-production

all-staging: hugo-mod build-staging

all-dev: hugo-mod build-dev

all-local: clean hugo-mod build-production

docs:
	${HUGO}

clean:
	if [[ -d ${PWD}/public ]] ; then rm -rf ${PWD}/public && echo "Removed public directory" ; else echo "Did not find a public directory to remove" ; fi

watch:
	${HUGO} --bind 0.0.0.0 -p 1313 server --disableFastRender

watch-drafts:
	${HUGO} --bind 0.0.0.0 -p 1313 server -D --disableFastRender

link-check:
	${MARKDOWNLINKCHECK} $(shell find content -name '*.md')

lint-markdown:
	${MARKDOWNLINT} -c .markdownlint.json  -- content

# Commands used by Netlify CI
hugo-mod:
	hugo mod get $(THEME_MODULE)@v$(THEME_VERSION)

build-production:
	hugo --gc -e production

build-staging:
	hugo --gc -e staging

build-dev:
	hugo --gc -e development

deploy-preview: hugo-mod
	hugo --gc -b ${NETLIFY_DEPLOY_URL}/nginx-ingress-controller/
