---
title: Upgrading NGINX Ingress Controller with Helm
description: "This document describes how to use Helm to upgrade NGINX Ingress Controller from 2.x and 3.0.x to the latest available release."
weight: 1800
doctypes: [""]
toc: true
docs: "DOCS-603"
---

## Background

In NGINX Ingress Controller version 3.1.0, [changes were introduced](https://github.com/nginxinc/kubernetes-ingress/pull/3606) to Helm resource names, labels and annotations to fit with Helm best practices.
When using Helm to upgrade from a version prior to to 3.1.0, certain resources like Deployment, DaemonSet and Service will be recreated due to the aforementioned changes, which will result in downtime. 
Although the advisory is to update all resources in accordance with new naming convention, to avoid the downtime please follow the steps listed in this page.

## Upgrade Steps
{{<note>}} The following steps apply to both 2.x and 3.0.x releases.{{</note>}}

The steps you should follow depend on the Helm release name:

{{<tabs name="upgrade-helm">}}

{{%tab name="Release name is nginx-ingress"%}}

1. Use `kubectl describe deployments` to get the `Selector` value:

    ```shell
    kubectl describe deployments -n <namespace>
    ```
    Copy the key=value under `Selector`, such as:

    ```shell
    Selector:               app=nginx-ingress-nginx-ingress
    ```

1. Checkout the latest available tag using `git checkout v3.3.0`

1. Update the `selectorLabels: {}` field in the `values.yaml` file located at `/kubernates-ingress/deployments/helm-chart` with the copied `selector` value.
    ```shell
    selectorLabels: {app: nginx-ingress-nginx-ingress}
    ```

1. Run `helm upgrade` with following arguments set:
    ```shell
    --set controller.serviceNameOverride="nginx-ingress-nginx-ingress"
    --set controller.name=""
    --set fullnameOverride="nginx-ingress-nginx-ingress"
    ```
    It could look as follows:

     `helm upgrade nginx-ingress --set controller.kind=deployment --set controller.nginxplus=false --set controller.image.pullPolicy=Always --set controller.serviceNameOverride="nginx-ingress-nginx-ingress" --set controller.name="" --set fullnameOverride="nginx-ingress-nginx-ingress"`

1. Once the upgrade process has finished, use `kubectl describe` on the deployment to verify the change by reviewing its events:
    ```shell
        Type    Reason             Age    From                   Message
    ----    ------             ----   ----                   -------
    Normal  ScalingReplicaSet  9m11s  deployment-controller  Scaled up replica set nginx-ingress-nginx-ingress-<old_version> to 1
    Normal  ScalingReplicaSet  101s   deployment-controller  Scaled up replica set nginx-ingress-nginx-ingress-<new_version> to 1
    Normal  ScalingReplicaSet  98s    deployment-controller  Scaled down replica set nginx-ingress-nginx-ingress-<old_version> to 0 from 1

{{%/tab%}}

{{%tab name="Release name is not nginx-ingress"%}}

1. Use `kubectl describe deployments` to get the `Selector` value:

    ```shell
    kubectl describe deployments -n <namespace>
    ```
    Copy the key=value under ```Selector```, such as:

    ```shell
    Selector:               app=<helm_release_name>-nginx-ingress
    ```

1. Checkout the latest available tag using `git checkout v3.3.0`

1. Update the `selectorLabels: {}` field in the `values.yaml` file located at `/kubernates-ingress/deployments/helm-chart` with the copied `selector` value.
    ```shell
    selectorLabels: {app: <helm_release_name>-nginx-ingress}
    ```

1. Run `helm upgrade` with following arguments set:
    ```shell
      --set controller.serviceNameOverride="<helm_release_name>-nginx-ingress",
      --set controller.name=""
    ```
    It could look as follows:

    `helm upgrade test-release --set controller.kind=deployment --set controller.nginxplus=false --set controller.image.pullPolicy=Always --set controller.serviceNameOverride="test-release-nginx-ingress" --set controller.name=""`

1. Once the upgrade process has finished, use `kubectl describe` on the deployment to verify the change by reviewing its events:
    ```shell
        Type    Reason             Age    From                   Message
    ----    ------             ----   ----                   -------
    Normal  ScalingReplicaSet  9m11s  deployment-controller  Scaled up replica set test-release-nginx-ingress-<old_version> to 1
    Normal  ScalingReplicaSet  101s   deployment-controller  Scaled up replica set test-release-nginx-ingress-<new_version> to 1
    Normal  ScalingReplicaSet  98s    deployment-controller  Scaled down replica set test-release-nginx-ingress-<old_version> to 0 from 1
    ```
{{%/tab%}}

{{</tabs>}}
