NETLIFY_ROOT = /opt/build/repo
SITE_ROOT = ${NETLIFY_ROOT}/site
BUILDDIR = ${NETLIFY_ROOT}/docs-web/build
THEME_REPO = ${THEME_REPO_URL}

.PHONY: all build clean setup

all: clean setup build

clean:
	rm -rf ${SITE_ROOT}

setup:
	mkdir ${SITE_ROOT}
	@git clone -q --single-branch --branch master https://$(GITHUB_PAT)@${THEME_REPO} ${SITE_ROOT}/docs-nginx-com
	pip install -r ${SITE_ROOT}/docs-nginx-com/requirements.txt
	rsync -avOz ${NETLIFY_ROOT}/docs-web/ ${SITE_ROOT}/docs-nginx-com/source/nginx-ingress-controller/ --exclude ${NETLIFY_ROOT}/docs-web/Makefile --exclude ${NETLIFY_ROOT}/docs-web/_source
	cp ${NETLIFY_ROOT}/docs-web/_source/home_page_body.html ${SITE_ROOT}/docs-nginx-com/_themes/docs-theme/includes/home_page_body.html
	cp ${NETLIFY_ROOT}/docs-web/_source/dev-home.txt ${SITE_ROOT}/docs-nginx-com/source/index.rst
	cp ${NETLIFY_ROOT}/docs-web/_source/dev-home.txt ${SITE_ROOT}/docs-nginx-com/source/home.rst

build:
	sphinx-build -b dirhtml -d ${BUILDDIR}/doctrees ${SITE_ROOT}/docs-nginx-com/source ${BUILDDIR}/dirhtml
	@echo "Build finished. The HTML pages are in ${BUILDDIR}/dirhtml."