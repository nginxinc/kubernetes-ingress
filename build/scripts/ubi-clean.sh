#!/bin/sh

set -e

microdnf remove -y shadow-utils subscription-manager
microdnf clean all && rm -rf /var/cache/dnf
