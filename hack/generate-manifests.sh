#!/usr/bin/env bash

# Generate deployment files using Helm. This script uses the Helm chart examples in examples/helm-chart

charts=$(find examples/helm-chart -maxdepth 1 -mindepth 1 -type d -exec basename {} \;)

for chart in $charts; do
    manifest=deploy/$chart/deploy.yaml
    helm template nginx-ingress --namespace nginx-ingress --values examples/helm-chart/$chart/values.yaml --skip-crds charts/nginx-ingress >$manifest 2>/dev/null
    sed -i.bak '/app.kubernetes.io\/managed-by: Helm/d' $manifest
    sed -i.bak '/helm.sh/d' $manifest
    cp $manifest config/base
    if [ "$chart" == "app-protect-dos" ]; then
        kustomize build config/overlays/app-protect-dos >$manifest
    else
        kustomize build config/base >$manifest
    fi
    rm -f config/base/deploy.yaml
    rm -f $manifest.bak
done
