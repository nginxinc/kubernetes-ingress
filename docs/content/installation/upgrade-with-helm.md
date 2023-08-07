---
title: Upgrade with Helm
description: "This document describes how to upgrade the NGINX Ingress Controller in your Kubernetes cluster using Helm."
weight: 1800
doctypes: [""]
toc: true
docs: "DOCS-603"
---

This document will walk you through the steps needed to upgrade the NGINX Ingress Controller from version 2.x and 3.0.x to latest available release without any downtime.

## Background

In the NGINX Ingress Controller version 3.1.0, [changes were introduced](https://github.com/nginxinc/kubernetes-ingress/pull/3606) to Helm resource names, labels and annotations to align better with Helm best practices.
However upon running a helm upgrade job from previous installed version to 3.1.0+, some resources like Deploymeny/DaemonSet and Service will be recreated (recreate no rolling update) due to changes in nomenclature and selector labels resulting in undesirable downtime.

## Steps to perform upgrade without downtine
**Note**: Below steps applies to both 2.x and 3.0.x releases.

### If Helm release name is ```nginx-ingress```

1. Describe existing deployment/daemonset to get ```selector```.

    ```shell
    kubectl describe deployments -n <namespace>
    ```
    Copy the key=value under ```selector```, eg:

    ```shell
    Selector:               app=nginx-ingress-nginx-ingress
    ```

2. Checkout the latest available tag using ```git checkout v3.3.0```

3. Update the ```selectorLabels: {}``` field in ```values.yaml``` file located at       ``` /kubernates-ingress/deployments/helm-chart``` with the ```selector``` from Step 1.
    ```shell
    selectorLabels: {app: nginx-ingress-nginx-ingress}
    ```

4. Run helm upgrade with following arguments set:
    ```shell
    --set controller.serviceNameOverride=“nginx-ingress-nginx-ingress”
    --set controller.name=""
    --set fullnameOverride=“nginx-ingress-nginx-ingress”
    ```
    eg: ```helm upgrade nginx-ingress --set controller.kind=deployment --set controller.nginxplus=false --set controller.image.pullPolicy=Always --set controller.serviceNameOverride=“nginx-ingress-nginx-ingress” --set controller.name=“” --set fullnameOverride=“nginx-ingress-nginx-ingress” .```

5. Once upgrade process is finished, verify it by running a ```kubectl describe... ``` on deployment (daemonset will have no such events) and that events section should have rolling update workflow:
    eg:
    ```shell
        Type    Reason             Age    From                   Message
    ----    ------             ----   ----                   -------
    Normal  ScalingReplicaSet  9m11s  deployment-controller  Scaled up replica set nginx-ingress-nginx-ingress-<old_version> to 1
    Normal  ScalingReplicaSet  101s   deployment-controller  Scaled up replica set nginx-ingress-nginx-ingress-<new_version> to 1
    Normal  ScalingReplicaSet  98s    deployment-controller  Scaled down replica set nginx-ingress-nginx-ingress-<old_version> to 0 from 1


### If Helm release name is not ```nginx-ingress```

1. Describe existing deployment/daemonset to get ```selector```.

    ```shell
    kubectl describe deployments -n <namespace>
    ```
    Copy the key=value under ```selector```, eg:

    ```shell
    Selector:               app=<helm_release_name>-nginx-ingress
    ```

2. Checkout the latest available tag using ```git checkout v3.3.0```

3. Update the ```selectorLabels: {}``` field in ```values.yaml``` file located at       ``` /kubernates-ingress/deployments/helm-chart``` with the ```selector``` from Step 1.
    ```shell
    selectorLabels: {app: <helm_release_name>-nginx-ingress}
    ```

4. Run helm upgrade with following arguments set:
    ```shell
      --set controller.serviceNameOverride=“<helm_release_name>-nginx-ingress”,
      --set controller.name=""
    ```
    eg: ```helm upgrade test-release --set controller.kind=deployment --set controller.nginxplus=false --set controller.image.pullPolicy=Always --set controller.serviceNameOverride=“test-release-nginx-ingress” --set controller.name="" .```

5. Once upgrade process is finished, verify it by running a ```kubectl describe... ``` on deployment and that events section should have rolling update workflow:
    eg:
    ```shell
        Type    Reason             Age    From                   Message
    ----    ------             ----   ----                   -------
    Normal  ScalingReplicaSet  9m11s  deployment-controller  Scaled up replica set test-release-nginx-ingress-<old_version> to 1
    Normal  ScalingReplicaSet  101s   deployment-controller  Scaled up replica set test-release-nginx-ingress-<new_version> to 1
    Normal  ScalingReplicaSet  98s    deployment-controller  Scaled down replica set test-release-nginx-ingress-<old_version> to 0 from 1
    ```
