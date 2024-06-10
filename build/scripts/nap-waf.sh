#!/bin/sh

set -e

mkdir -p /etc/nginx/waf/nac-policies /etc/nginx/waf/nac-logconfs /etc/nginx/waf/nac-usersigs /var/log/app_protect /opt/app_protect
chown -R 101:0 /etc/app_protect /usr/share/ts /var/log/app_protect/ /opt/app_protect/
chmod -R g=u /etc/app_protect /usr/share/ts /var/log/app_protect/ /opt/app_protect/
touch /etc/nginx/waf/nac-usersigs/index.conf
