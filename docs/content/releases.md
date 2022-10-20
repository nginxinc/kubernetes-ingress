---
title: Releases
description: "NGINX Ingress Controller Release Notes."
weight: 1900
doctypes: ["concept"]
toc: true
docs: "DOCS-616"
---
## NGINX Ingress Controller 2.4.0

04 Oct 2022

OVERVIEW:

* Added support for enabling [proxy_protocol](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/shared-examples/proxy-protocol) when port 443 is being used for both HTTPS traffic and [TLS Passthrough traffic](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/custom-resources/tls-passthrough).
* Updates to the TransportServer resource to support using [ExternalName services](https://kubernetes.io/docs/concepts/services-networking/service/#externalname). For examples, see [externalname-services](https://github.com/nginxinc/kubernetes-ingress/tree/v2.4.0/examples/custom-resources/externalname-services).
* VirtualServer resource now supports [wildcard hostname](https://kubernetes.io/docs/concepts/services-networking/ingress/#hostname-wildcards).
* NGINX Ingress Controller images including the combined NGINX AppProtect WAF and NGINX AppProtect DoS solutions are now published to our registry. See [Images with NGINX Plus](https://docs.nginx.com/nginx-ingress-controller/technical-specifications/#images-with-nginx-plus) for a detailed list of images in our registry.
* Added support for watching multiple namespaces using the [-watch-namespace](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#-watch-namespace-string) cli argument. This can configured by passing a comma-separated list of namespaces to the `-watch-namespace` CLI argument (e.g. `-watch-namespace=ns-1,ns-2`).
* A new cli argument has been added: [-include-year](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#-include-year). This appends the current year to the log output from the ingress controller. Example output: `I20220512 09:20:42.345457`.
* Post-startup configuration reloads have been optimized to reduce traffic impacts. When many resources are modified at the same time, changes are combined to reduce the number of data plane reloads.

FEATURES:

* [2986](https://github.com/nginxinc/kubernetes-ingress/pull/2986) Batch reloads at runtime.
* [2914](https://github.com/nginxinc/kubernetes-ingress/pull/2914) Support watching multiple namespaces.
* [2884](https://github.com/nginxinc/kubernetes-ingress/pull/2884) Include year in logs.
* [2993](https://github.com/nginxinc/kubernetes-ingress/pull/2993) Accept proxy protocol when TLS passthrough enabled.
* [3041](https://github.com/nginxinc/kubernetes-ingress/pull/3041) Support external name service for TansportServer.
* [2939](https://github.com/nginxinc/kubernetes-ingress/pull/2939) Add support for wildcard hostname in VirtualServer.

IMPROVEMENTS:

* [3040](https://github.com/nginxinc/kubernetes-ingress/pull/3040) Add command line argument to manually disable IPV6 listeners for unsupported clusters.
* [3088](https://github.com/nginxinc/kubernetes-ingress/pull/3088) Filter secrets of type helm.sh/release.v1.

FIXES:

* [2971](https://github.com/nginxinc/kubernetes-ingress/pull/2971) fix: Correct error message on missing path in path validation. Thanks to [Zachary Seguin](https://github.com/zachomedia).
* [3095](https://github.com/nginxinc/kubernetes-ingress/pull/3095) do not create configmap if customConfigMap is used. Thanks to [Bryan Hendryx](https://github.com/coolbry95).

HELM CHART:

* [3087](https://github.com/nginxinc/kubernetes-ingress/pull/3087) Allow omitting the default server secret from Helm installs.
* [2831](https://github.com/nginxinc/kubernetes-ingress/pull/2831) Add ServiceMonitor to Helm Chart. Thanks to [araineUnity](https://github.com/araineUnity).
* [2855](https://github.com/nginxinc/kubernetes-ingress/pull/2854) Add initialDelaySeconds to helm charts. Thanks to [Daniel Edgar](https://github.com/aknot242).
* [2979](https://github.com/nginxinc/kubernetes-ingress/pull/2979) Allow to specify image with digest in helm chart. Thanks to [Hans Feldt](https://github.com/hafe).
* [3031](https://github.com/nginxinc/kubernetes-ingress/pull/3031) Adding automountServiceAccountToken to helm chart.

UPGRADE:
* For NGINX, use the 2.4.0 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.4.0), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress) or [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress).
* For NGINX Plus, use the 2.4.0 images from the F5 Container registry or the [AWS Marketplace](https://aws.amazon.com/marketplace/search/?CREATOR=741df81b-dfdc-4d36-b8da-945ea66b522c&FULFILLMENT_OPTION_TYPE=CONTAINER&filters=CREATOR%2CFULFILLMENT_OPTION_TYPE) or build your own image using the 2.4.0 source code.
* For Helm, use version 0.15.0 of the chart. If you're using custom resources like VirtualServer and TransportServer (`controller.enableCustomResources` is set to `true`), after you run the `helm upgrade` command, the CRDs will not be upgraded. After running the `helm upgrade` command, run `kubectl apply -f deployments/helm-chart/crds` to upgrade the CRDs.

SUPPORTED PLATFORMS:

We will provide technical support for NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and that passes the Kubernetes conformance tests. This release was fully tested on the following Kubernetes versions: 1.19-1.25.

## NGINX Ingress Controller 2.3.1

16 Sep 2022

CHANGES:
* [3048](https://github.com/nginxinc/kubernetes-ingress/pull/3048) Bump NGINX to 1.23.1
* [3049](https://github.com/nginxinc/kubernetes-ingress/pull/3049) Update Go dependencies.
* [3031](https://github.com/nginxinc/kubernetes-ingress/pull/3031) Add automountServiceAccountToken to helm chart

HELM CHART:
* The version of the Helm chart is now 0.14.1.

UPGRADE:
* For NGINX, use the 2.3.1 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.3.1), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress), [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress) or [Quay.io](https://quay.io/repository/nginx/nginx-ingress).
* For NGINX Plus, use the 2.3.1 images from the F5 Container registry or build your own image using the 2.3.1 source code.
* For Helm, use version 0.14.1 of the chart.

## NGINX Ingress Controller 2.3.0

12 July 2022

OVERVIEW:

* Support making VirtualServer resources discoverable via public DNS servers using [external-dns](https://kubernetes-sigs.github.io/external-dns). Examples for configuring external-dns with NGINX Ingress Controller can be found [here](https://github.com/nginxinc/kubernetes-ingress/tree/v2.3.0/examples/custom-resources/external-dns).
* Resolves [CVE-2022-30535](https://support.f5.com/csp/article/K52125139). This vulnerability impacted the visibility of secrets accessible by NGINX Ingress Controller. In some cases, secrets visible to NGINX Ingress Controller could be exposed to any authenticated user with permission to create and update Ingress objects. This vulnerability affected Ingress objects only - our Custom Resources (VirtualServer and TransportServer) were not affected. Customers unable to upgrade should migrate any Ingress resources to VirtualServer resources where possible, and use RBAC to restrict write access for users for Ingress objects.
* Support using HTTP basic authentication with [VirtualServer](https://github.com/nginxinc/kubernetes-ingress/tree/v2.3.1/examples/custom-resources/basic-auth) and [Ingress](https://github.com/nginxinc/kubernetes-ingress/tree/v2.3.0/examples/basic-auth) resources. Special thanks to [Simon Wachter](https://github.com/svvac).
* Support HTTP01 type ACME Issuers for use with VirtualServer resources with [cert-manager](https://cert-manager.io/docs/).

FEATURES:

* [2581](https://github.com/nginxinc/kubernetes-ingress/pull/2581) Add OpenTracing to all Debian and Alpine based images.
* [2328](https://github.com/nginxinc/kubernetes-ingress/pull/2328) Add handling of multiple log destinations.
* [2691](https://github.com/nginxinc/kubernetes-ingress/pull/2691) AP: log-conf escaping chars.
* [2759](https://github.com/nginxinc/kubernetes-ingress/pull/2759) Add support for HTTP01 Challenges on VirtualServer resources.
* [2762](https://github.com/nginxinc/kubernetes-ingress/pull/2762) Add DNSEndpoint CRD for integration with ExternalDNS.
* [2801](https://github.com/nginxinc/kubernetes-ingress/pull/2801) Add SBOMs to release.
* [2269](https://github.com/nginxinc/kubernetes-ingress/pull/2269) HTTP basic auth support. Thanks to [Simon Wachter](https://github.com/svvac).
* [2800](https://github.com/nginxinc/kubernetes-ingress/pull/2800) Integrate external-dns with VirtualServer resources.

IMPROVEMENTS:

* [2583](https://github.com/nginxinc/kubernetes-ingress/pull/2583) Add runAsNonRoot in deployments.
* [2484](https://github.com/nginxinc/kubernetes-ingress/pull/2484) Add container resource requests.
* [2627](https://github.com/nginxinc/kubernetes-ingress/pull/2627) Update InternalRoute server_name.
* [2742](https://github.com/nginxinc/kubernetes-ingress/pull/2742) Add additional unit tests to confirm special characters can't be used in the lb-method annotation.
* [2730](https://github.com/nginxinc/kubernetes-ingress/pull/2730) Add string sanitisation for proxy-pass-headers & proxy-hide-headers.
* [2733](https://github.com/nginxinc/kubernetes-ingress/pull/2733) Add string validation to server-tokens annotation.
* [2734](https://github.com/nginxinc/kubernetes-ingress/pull/2734) Validate rewrite annotation.
* [2754](https://github.com/nginxinc/kubernetes-ingress/pull/2754) Validate JWT key, realm and login url for ingress resources annotations.
* [2751](https://github.com/nginxinc/kubernetes-ingress/pull/2751) Add string validation to sticky-cookie-services annotation.
* [2775](https://github.com/nginxinc/kubernetes-ingress/pull/2775) Add validation to Ingress path.
* [2774](https://github.com/nginxinc/kubernetes-ingress/pull/2774) Sanitize nginx.com/jwt-token.
* [2783](https://github.com/nginxinc/kubernetes-ingress/pull/2783) Update validation regex for path spec.
* [2781](https://github.com/nginxinc/kubernetes-ingress/pull/2781) Report Hostname in ExternalEndpoint for VS and VSR resources.

FIXES:

* [2617](https://github.com/nginxinc/kubernetes-ingress/pull/2617) Fix Dockerfile for amd64 microarchitectures.
* [2637](https://github.com/nginxinc/kubernetes-ingress/pull/2637) Add terminationGracePeriodSeconds to deployment. Thanks to [Maksym Iv](https://github.com/maksym-iv).
* [2654](https://github.com/nginxinc/kubernetes-ingress/pull/2654) Sync changes from OIDC repo, add field in policy.
* [2673](https://github.com/nginxinc/kubernetes-ingress/pull/2673) Fix status.loadbalancer.hostname deletion on OOMKill. Thanks to [Heiko Voigt](https://github.com/hvoigt).
* [2718](https://github.com/nginxinc/kubernetes-ingress/pull/2718) Fix cases where CM enabled but no TLS block specified in VS.

HELM CHART:

* [2418](https://github.com/nginxinc/kubernetes-ingress/pull/2418) Add support for allocateLoadBalancerNodePorts, ipFamilyPolicy and ipFamilies. Thanks to [centromere](https://github.com/centromere).
* [2672](https://github.com/nginxinc/kubernetes-ingress/pull/2672) Add minReadySeconds & strategy support. Thanks to [Ciaran](https://github.com/cmk-pcs).
* [2625](https://github.com/nginxinc/kubernetes-ingress/pull/2625) allow configuring topologySpreadConstraints in Helm chart. Thanks to [Kamil Domański](https://github.com/kdomanski).


UPGRADE:
* For NGINX, use the 2.3.0 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.3.0), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress) or [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress).
* For NGINX Plus, use the 2.3.0 images from the F5 Container registry or the [AWS Marketplace](https://aws.amazon.com/marketplace/search/?CREATOR=741df81b-dfdc-4d36-b8da-945ea66b522c&FULFILLMENT_OPTION_TYPE=CONTAINER&filters=CREATOR%2CFULFILLMENT_OPTION_TYPE) or build your own image using the 2.3.0 source code.
* For Helm, use version 0.14.0 of the chart. If you're using custom resources like VirtualServer and TransportServer (`controller.enableCustomResources` is set to `true`), after you run the `helm upgrade` command, the CRDs will not be upgraded. After running the `helm upgrade` command, run `kubectl apply -f deployments/helm-chart/crds` to upgrade the CRDs.
* When upgrading using the [manifests](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/), make sure to update the [ClusterRole](https://github.com/nginxinc/kubernetes-ingress/blob/v2.3.1/deployments/rbac/rbac.yaml). This is required to enable the ExternalDNS for VirtualServer resources integration.

SUPPORTED PLATFORMS:

We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the Kubernetes conformance tests. This release was fully tested on the following Kubernetes versions: 1.19-1.24.

## NGINX Ingress Controller 2.2.2

23 May 2022

FIXES:
* [2627](https://github.com/nginxinc/kubernetes-ingress/pull/2627) Update InternalRoute server_name.

HELM CHART:
* The version of the Helm chart is now 0.13.2.

UPGRADE:
* For NGINX, use the 2.2.2 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.2.2), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress) or [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress).
* For NGINX Plus, use the 2.2.2 images from the F5 Container registry or build your own image using the 2.2.2 source code.
* For Helm, use version 0.13.2 of the chart.

## NGINX Ingress Controller 2.2.1

17 May 2022

CHANGES:
* Update Go dependencies.

FIXES:
* [2654](https://github.com/nginxinc/kubernetes-ingress/pull/2654) Sync changes from [nginx-openid-connect](https://github.com/nginxinc/nginx-openid-connect) repo, add zoneSyncLeeway field in policy. For more information on the fixes, see [52](https://github.com/nginxinc/nginx-openid-connect/pull/52).

HELM CHART:
* The version of the Helm chart is now 0.13.1.

UPGRADE:
* For NGINX, use the 2.2.1 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.2.1), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress) or [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress).
* For NGINX Plus, use the 2.2.1 images from the F5 Container registry or build your own image using the 2.2.1 source code.
* For Helm, use version 0.13.1 of the chart.

## NGINX Ingress Controller 2.2.0

12 Apr 2022

OVERVIEW:

* Support for automatic provisioning and management of Certificate resources for VirtualServer resources using [cert-manager](https://cert-manager.io/docs/). Examples for configuring cert-manager with NGINX Ingress Controller can be found [here](https://github.com/nginxinc/kubernetes-ingress/tree/v2.2.0/examples/custom-resources/certmanager). Please note that HTTP01 type ACME Issuers are not yet supported for use with VirtualServer resources.

* Full support for IPv6 using the NGINX Ingress Controller [VirtualServer and VirtualServerRoute](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources) custom resources, and Ingress resources.

* The [-enable-preview-policies](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#-enable-preview-policies) cli argument has been deprecated and is no longer required for the usage of any Policy resource type. This argument will be removed completely in v2.6.0.

* A new [-enable-oidc](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/command-line-arguments/#-enable-oidc) cli argument has been added to enable OIDC policies. Previously, this behaviour was achieved through the usage of the `-enable-preview-policies` cli argument.

FEATURES:
* [2576](https://github.com/nginxinc/kubernetes-ingress/pull/2576) Add support for IPv6.
* [2572](https://github.com/nginxinc/kubernetes-ingress/pull/2572) Automate provisioning of Certificate resources for VirtualServer resources using cert-manager.

IMPROVEMENTS:
* [2346](https://github.com/nginxinc/kubernetes-ingress/pull/2346) Use os.ReadDir for lightweight directory reading. Thanks to [Eng Zer Jun](https://github.com/Juneezee).
* [2360](https://github.com/nginxinc/kubernetes-ingress/pull/2360) Add NGINX App Protect reconnect period directive.
* [2479](https://github.com/nginxinc/kubernetes-ingress/pull/2479) Add cli argument to configure NGINX App Protect log level.
* [2455](https://github.com/nginxinc/kubernetes-ingress/pull/2455) Increase memory available for NGINX App Protect xml parser.

CHANGES:
* [2580](https://github.com/nginxinc/kubernetes-ingress/pull/2580) Create -enable-oidc command line argument for OIDC policy.
* [2566](https://github.com/nginxinc/kubernetes-ingress/pull/2566) Unbind policy from preview policies.
* [2582](https://github.com/nginxinc/kubernetes-ingress/pull/2582) Rename Make targets from `openshift` to `ubi`.

FIXES:
* [2378](https://github.com/nginxinc/kubernetes-ingress/pull/2378) Fix healthcheck ports.
* [2404](https://github.com/nginxinc/kubernetes-ingress/pull/2404) Start nginx with -e stderr parameter.
* [2414](https://github.com/nginxinc/kubernetes-ingress/pull/2414) Fix in file nginx-plus.virtualserver.tmpl ApDosMonitor->ApDosMonitorURI.

HELM CHART:
* [2525](https://github.com/nginxinc/kubernetes-ingress/pull/2525) Extend helm chart to include NGINX Service Mesh fields.
* [2294](https://github.com/nginxinc/kubernetes-ingress/pull/2294) Add extra containers to helm chart. Thanks to [Márk Sági-Kazár](https://github.com/sagikazarmark).

UPGRADE:
* For NGINX, use the 2.2.0 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.2.0), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress) or [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress).
* For NGINX Plus, use the 2.2.0 images from the F5 Container registry or the [AWS Marketplace](https://aws.amazon.com/marketplace/search/?CREATOR=741df81b-dfdc-4d36-b8da-945ea66b522c&FULFILLMENT_OPTION_TYPE=CONTAINER&filters=CREATOR%2CFULFILLMENT_OPTION_TYPE) or build your own image using the 2.2.0 source code.
* For Helm, use version 0.13.0 of the chart. If you're using custom resources like VirtualServer and TransportServer (`controller.enableCustomResources` is set to `true`), after you run the `helm upgrade` command, the CRDs will not be upgraded. After running the `helm upgrade` command, run `kubectl apply -f deployments/helm-chart/crds` to upgrade the CRDs.
* When upgrading using the [manifests](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/), make sure to update the [ClusterRole](https://github.com/nginxinc/kubernetes-ingress/blob/v2.2.0/deployments/rbac/rbac.yaml). This is required to enable the cert-manager for VirtualServer resources integration.
* The -enable-preview-policies cli argument has been deprecated, and is no longer required for any Policy resources.
* Enabling OIDC Policies now requires the use of -enable-oidc cli argument instead of the -enable-preview-policies cli argument.

SUPPORTED PLATFORMS:

We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the Kubernetes conformance tests. This release was fully tested on the following Kubernetes versions: 1.19-1.23.

## NGINX Ingress Controller 2.1.2

29 Mar 2022

CHANGES:
* Update UBI based images to 8.

HELM CHART:
* The version of the Helm chart is now 0.12.2.

UPGRADE:
* For NGINX, use the 2.1.2 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.1.2), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress) or [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress).
* For NGINX Plus, use the 2.1.2 images from the F5 Container registry or the [AWS Marketplace](https://aws.amazon.com/marketplace/search/?CREATOR=741df81b-dfdc-4d36-b8da-945ea66b522c&FULFILLMENT_OPTION_TYPE=CONTAINER&filters=CREATOR%2CFULFILLMENT_OPTION_TYPE) or build your own image using the 2.1.2 source code.
* For Helm, use version 0.12.2 of the chart.

## NGINX Ingress Controller 1.12.4

23 March 2022

CHANGES:
* Update NGINX version to 1.21.6.
* Update NGINX Plus version to R26.
* Update Debian to Bullseye.
* Update Alpine to 3.15.
* Update UBI to 8.
* Update Go to 1.17 and Go dependencies.

FIXES:
* Fix OpenTracing not working with NGINX Plus.

HELM CHART:
* The version of the Helm chart is now 0.10.4.

UPGRADE:
* For NGINX, use the 1.12.4 image from our DockerHub: `nginx/nginx-ingress:1.12.4`, `nginx/nginx-ingress:1.12.4-alpine` or `nginx/nginx-ingress:1.12.4-ubi`
* For NGINX Plus, please build your own image using the 1.12.4 source code.
* For Helm, use version 0.10.4 of the chart.

## NGINX Ingress Controller 2.1.1

17 Feb 2022

CHANGES:
* Update NGINX version to 1.21.6.
* Update NGINX Plus version to R26.

HELM CHART:
* The version of the Helm chart is now 0.12.1.

UPGRADE:
* For NGINX, use the 2.1.1 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.1.1), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress) or [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress).
* For NGINX Plus, use the 2.1.1 images from the F5 Container registry or build your own image using the 2.1.1 source code.
* For Helm, use version 0.12.1 of the chart.

## NGINX Ingress Controller 2.1.0

06 Jan 2022

OVERVIEW:

* Support for NGINX App Protect Denial of Service protection with NGINX Ingress Controller. More information about [NGINX App Protect DoS](https://www.nginx.com/products/nginx-app-protect/denial-of-service/). Examples for configuring NGINX App Protect DoS with NGINX Ingress Controller can be found [here](https://github.com/nginxinc/kubernetes-ingress/tree/v2.1.1/examples/appprotect-dos).

* Full support for gRPC services using the NGINX Ingress Controller [VirtualServer and VirtualServerRoute](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources) custom resource definitions.  This makes configuring and supporting gRPC services much easier, giving a simple YAML configuration and removing the need for snippets. Resource definition examples for gRPC can be found [here](https://github.com/nginxinc/kubernetes-ingress/tree/v2.1.1/examples/custom-resources/grpc-upstreams).

* Implementation of NGINX mandatory and persistent health checks in VirtualServer and VirtualServerRoute to further reduce interruptions to your service traffic as configuration changes continuously happen in your dynamic Kubernetes environment(s). Health checks have been extended to include `mandatory` and `persistent` fields. Mandatory health checks ensures that a new upstream server starts receiving traffic only after the health check passes. Mandatory health checks can be marked as persistent, so that the previous state is remembered when the Ingress Controller reloads NGINX Plus configuration. When combined with the slow-start parameter, the mandatory health check give a new upstream server more time to connect to databases and “warm up” before being asked to handle their full share of traffic. See the settings [here](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#upstreamhealthcheck). More about the [NGINX Plus mandatory and persistent health check features](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/#mandatory-health-checks).
Mandatory health checks can be marked as persistent, so that the previous state is remembered when reloading configuration. When combined with the slow-start parameter, it gives a new service pod more time to connect to databases and “warm up” before being asked to handle their full share of traffic.
See the settings [here](https://docs.nginx.com/nginx-ingress-controller/configuration/virtualserver-and-virtualserverroute-resources/#upstreamhealthcheck).
More about the [NGINX Plus mandatory and persistent health check features](https://docs.nginx.com/nginx/admin-guide/load-balancer/http-health-check/#mandatory-health-checks)

FEATURES:
* [2251](https://github.com/nginxinc/kubernetes-ingress/pull/2251) Enable setting mandatory and persistent in upstream healthchecks in VS and VSR.
* [2241](https://github.com/nginxinc/kubernetes-ingress/pull/2241) Add support for NGINX App Protect DoS.
* [2200](https://github.com/nginxinc/kubernetes-ingress/pull/2200) Add Alpine image with OpenTracing.
* [2178](https://github.com/nginxinc/kubernetes-ingress/pull/2178) Support healthchecks in gRPC upstreams.
* [2110](https://github.com/nginxinc/kubernetes-ingress/pull/2110) Support gRPC in the Upstreams of the virtual server resources. Particular thanks to [Chiyu Zhong](https://github.com/CatTail) for all their work.
* [2149](https://github.com/nginxinc/kubernetes-ingress/pull/2149) Add metric about total number of TransportServers.
* [2100](https://github.com/nginxinc/kubernetes-ingress/pull/2100) Add support for initContainers. Thanks to [Gunnar Scherf](https://github.com/g10f).
* [1827](https://github.com/nginxinc/kubernetes-ingress/pull/1827) Add support for wildcard cert in VirtualServer resources. Thanks to [Simon Wachter](https://github.com/svvac).
* [2107](https://github.com/nginxinc/kubernetes-ingress/pull/2107) Add option to download the NGINX Ingress Controller binary. Introduced a new `TARGET` `download` in the `Makefile` which can be used when building the NGINX Ingress Controller Docker image. With this option the Ingress Controller binary will be downloaded instead of built from source.

IMPROVEMENTS:
* [2044](https://github.com/nginxinc/kubernetes-ingress/pull/2044) Upload NGINX Ingress Controller binaries to release.
* [2094](https://github.com/nginxinc/kubernetes-ingress/pull/2094) AP: update appolicies crd.
* [2216](https://github.com/nginxinc/kubernetes-ingress/pull/2216) Add grpc_status to the logs.
* [2237](https://github.com/nginxinc/kubernetes-ingress/pull/2237) Unbind app-protect from -preview-policies.
* [2273](https://github.com/nginxinc/kubernetes-ingress/pull/2273) Make the resource comparison more informative in case of an error. Thanks to [Andrey Karpov](https://github.com/ndk)

FIXES:
* [2267](https://github.com/nginxinc/kubernetes-ingress/pull/2267) Fix URI rewrite in VirtualServers and VirtualServerRoutes.
* [2260](https://github.com/nginxinc/kubernetes-ingress/pull/2260) Check if refresh token is `undefined` and do not store it in this case. Thanks to [tippexs](https://github.com/tippexs) for the fix.
* [2215](https://github.com/nginxinc/kubernetes-ingress/pull/2215) enableSnippets should not depend from enableCustomResources. Thanks to [Alessio Casco](https://github.com/AlessioCasco) for the fix.
* [1934](https://github.com/nginxinc/kubernetes-ingress/pull/1934) AP: fix watch-namespace for NAP resources.
* [2125](https://github.com/nginxinc/kubernetes-ingress/pull/2125) Allow empty string in server-tokens annotation for NGINX Plus.
* [2042](https://github.com/nginxinc/kubernetes-ingress/pull/2042) Use release specific repo for NGINX Plus on Debian.

CHANGES:
* [2124](https://github.com/nginxinc/kubernetes-ingress/pull/2124) Apply -enable-snippets cli arg to Ingresses. This PR extends the existing -enable-snippets cli argument to apply to Ingress resources. If snippets are not enabled, the Ingress Controller will reject any Ingress resources with snippets annotations. Previously, the argument only applied to VirtualServer, VirtualServerRoute and TransportServer resources. Please Note: this is a breaking change. See the `UPGRADE` instructions below.
* [2173](https://github.com/nginxinc/kubernetes-ingress/pull/2173) Update Debian to Bullseye.
* Update NGINX Plus version to R25.
* Update NGINX version to 1.21.5.

HELM CHART:
* The version of the Helm chart is now 0.12.0.

UPGRADE:
* For NGINX, use the 2.1.0 images from our [DockerHub](https://hub.docker.com/r/nginx/nginx-ingress/tags?page=1&ordering=last_updated&name=2.1.0), [GitHub Container](https://github.com/nginxinc/kubernetes-ingress/pkgs/container/kubernetes-ingress) or [Amazon ECR Public Gallery](https://gallery.ecr.aws/nginx/nginx-ingress).
* For NGINX Plus, use the 2.1.0 images from the F5 Container registry or build your own image using the 2.1.0 source code.
* For Helm, use version 0.12.0 of the chart.
* We changed the behaviour of snippets in Ingress resources by extending the existing -enable-snippets cli argument to apply to Ingress resources as well as VirtualServer, VirtualServerRoute and TransportServer resources. Because the default value of -enable-snippets is false, if you are using snippets in Ingress resources, you must explicitly set the -enable-snippets to true before upgrading the Ingress Controller, so that the new version of the Ingress Controller doesn't reject Ingresses with snippets annotations.

SUPPORTED PLATFORMS:

We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the Kubernetes conformance tests. This release was fully tested on the following Kubernetes versions: 1.19-1.23.

## NGINX Ingress Controller 2.0.3

28 Oct 2021

CHANGES:
* [2124](https://github.com/nginxinc/kubernetes-ingress/pull/2124) Apply -enable-snippets cli arg to Ingresses. This PR extends the existing -enable-snippets cli argument to apply to Ingress resources. If snippets are not enabled, the Ingress Controller will reject any Ingress resources with snippets annotations. Previously, the argument only applied to VirtualServer, VirtualServerRoute and TransportServer resources. Please Note: this is a breaking change. See the `UPGRADE` instructions below.
* [2132](https://github.com/nginxinc/kubernetes-ingress/pull/2132) Install libcurl on OpenTracing for NGINX Plus.

HELM CHART:
* The version of the Helm chart is now 0.11.3.

UPGRADE:
* For NGINX, use the 2.0.3 image from our DockerHub: `nginx/nginx-ingress:2.0.3`, `nginx/nginx-ingress:2.0.3-alpine` or `nginx/nginx-ingress:2.0.3-ubi`
* For NGINX Plus, please build your own image using the 2.0.3 source code.
* For Helm, use version 0.11.3 of the chart.
* We changed the behaviour of snippets in Ingress resources by extending the existing -enable-snippets cli argument to apply to Ingress resources as well as VirtualServer, VirtualServerRoute and TransportServer resources. Because the default value of -enable-snippets is false, if you are using snippets in Ingress resources, you must explicitly set the -enable-snippets to true before upgrading the Ingress Controller, so that the new version of the Ingress Controller doesn't reject Ingresses with snippets annotations.

## NGINX Ingress Controller 1.12.3

28 October 2021

FIXES:
* [2133](https://github.com/nginxinc/kubernetes-ingress/pull/2133) Use release specific repo for the App Protect packages on Debian. This fixes an error when building Debian-based images with NGINX Plus with App Protect: previously, building an image would fail with the error `nginx-plus-module-appprotect : Depends: app-protect-plugin (= 3.639.0-1~buster) but 3.671.0-1~buster is to be installed`. The bug first appeared when NGINX App Protect version 3.6 was released on 13 October 2021.
* [2134](https://github.com/nginxinc/kubernetes-ingress/pull/2134) Apply -enable-snippets cli arg to Ingresses. This PR extends the existing -enable-snippets cli argument to apply to Ingress resources. If snippets are not enabled, the Ingress Controller will reject any Ingress resources with snippets annotations. Previously, the argument only applied to VirtualServer, VirtualServerRoute and TransportServer resources. Please Note: this is a breaking change. See the `UPGRADE` instructions below.

HELM CHART:
* The version of the Helm chart is now 0.10.3.

UPGRADE:
* For NGINX, use the 1.12.3 image from our DockerHub: `nginx/nginx-ingress:1.12.3`, `nginx/nginx-ingress:1.12.3-alpine` or `nginx/nginx-ingress:1.12.3-ubi`
* For NGINX Plus, please build your own image using the 1.12.3 source code.
* For Helm, use version 0.10.3 of the chart.
* We changed the behaviour of snippets in Ingress resources by extending the existing -enable-snippets cli argument to apply to Ingress resources as well as VirtualServer, VirtualServerRoute and TransportServer resources. Because the default value of -enable-snippets is false, if you are using snippets in Ingress resources, you must explicitly set the -enable-snippets to true before upgrading the Ingress Controller, so that the new version of the Ingress Controller doesn't reject Ingresses with snippets annotations.

## NGINX Ingress Controller 2.0.2

13 Oct 2021

CHANGES:
* Update NGINX App Protect version to 3.6.
* Update NGINX Plus version to R25 in NGINX App Protect enabled images.
* [2074](https://github.com/nginxinc/kubernetes-ingress/pull/2074) Update JWT library to `golang-jwt/jwt`. Previously, the Ingress Controller used `dgrijalva/jwt-go`, which has a vulnerability [CVE-2020-26160](https://nvd.nist.gov/vuln/detail/CVE-2020-26160). Note that the Ingress Controller wasn’t affected by this vulnerability, and the jwt library was used only in the NGINX Plus images from AWS Marketplace for Containers.

HELM CHART:
* The version of the Helm chart is now 0.11.2.

UPGRADE:
* For NGINX, use the 2.0.2 image from our DockerHub.
* For NGINX Plus, use the 2.0.2 from the F5 Container registry or build your own image using the 2.0.2 source code.
* For Helm, use version 0.11.2 of the chart.

## NGINX Ingress Controller 2.0.1

07 Oct 2021

FIXES:
* [2051](https://github.com/nginxinc/kubernetes-ingress/pull/2051) Use release specific repo for NGINX Plus on Debian. This fixes an error when building the Debian-based image with NGINX Plus and App Protect: previously, building the image would fail with the error `Package 'nginx-plus-r24' has no installation candidate`.

DOCUMENTATION IMPROVEMENTS:
* [2059](https://github.com/nginxinc/kubernetes-ingress/pull/2059) fixes issues with the 404 and robots.txts redirects.
* [2049](https://github.com/nginxinc/kubernetes-ingress/pull/2049) Remove note from operator installation. A version of the Operator compatible with 2.0.x is now available.

HELM CHART:
* The version of the Helm chart is now 0.11.1.

UPGRADE:
* For NGINX, use the 2.0.1 image from our DockerHub.
* For NGINX Plus, use the 2.0.1 from the F5 Container registry or build your own image using the 2.0.1 source code.
* For Helm, use version 0.11.1 of the chart.

## NGINX Ingress Controller 1.12.2

7 October 2021

FIXES:
* [2048](https://github.com/nginxinc/kubernetes-ingress/pull/2048) Use release specific repo for NGINX Plus on Debian. This fixes an error when building Debian-based images with NGINX Plus: previously, building an image would fail with the error `Package 'nginx-plus-r24' has no installation candidate`. The bug first appeared when NGINX Plus R25 was released on 28 September 2021.

HELM CHART:
* The version of the Helm chart is now 0.10.2.

UPGRADE:
* For NGINX, use the 1.12.2 image from our DockerHub: `nginx/nginx-ingress:1.12.2`, `nginx/nginx-ingress:1.12.2-alpine` or `nginx/nginx-ingress:1.12.2-ubi`
* For NGINX Plus, please build your own image using the 1.12.2 source code.
* For Helm, use version 0.10.2 of the chart.

## NGINX Ingress Controller 2.0.0

28 September 2021

OVERVIEW:

Release 2.0.0 includes:
* *Support for Ingress networking.k8s.io/v1*. Kubernetes 1.22 removes support for networking.k8s.io/v1beta1. To support Kubernetes 1.22, NGINX Ingress Controller 2.0 is also compatible with only the networking.k8s.io/v1 version of the Ingress and IngressClass resources.  This has the following implications:
  1. The minimum supported version of Kubernetes is now 1.19. For older Kubernetes versions, use the 1.12.x release of the Ingress Controller.
  2. For Kubernetes versions 1.19-1.21, you can continue using the `networking.k8s.io/v1beta1` of the Ingress and IngressClass resources.
  3. For Kubernetes 1.22, you need to migrate your Ingress and IngressClass resources to `networking.k8s.io/v1`.
  4. If you are using the deprecated `kubernetes.io/ingress.class` annotation in your Ingress resources, it is recommended to switch to the `ingressClassName` field.

     We migrated all our documentation and examples to use `networking.k8s.io/v1` and the `ingressClassName` field of the Ingress resource.
* *Scalability improvements*. We improved the time for an Ingress Controller pod to become ready and start receiving traffic. This is especially noticeable when you have hundreds of Ingress or other configuration resources like VirtualServers: instead of several minutes or more in rare cases, a pod will become ready within a few minutes.
* *Documentation improvements* We changed the look and feel of our documentation at https://docs.nginx.com/nginx-ingress-controller as well as the underlying publishing technology, which will allow us to bring even more improvements in the next releases.
* *Upgrade path for k8s.nginx.org/v1alpha1 Policy resource* If you’re running release 1.9.0 and using the k8s.nginx.org/v1alpha1 Policy, the Ingress Controller now supports an upgrade path from v1alpha1 to v1 Policy version without downtime. See UPDATING POLICIES section below.

You will find the complete changelog for release 2.0.0, including bug fixes, improvements, and changes below.

FEATURES:
* [1908](https://github.com/nginxinc/kubernetes-ingress/pull/1908) Add NTLM support to VirtualServer and VirtualServerRoute upstreams.
* [1850](https://github.com/nginxinc/kubernetes-ingress/pull/1850) Support Ingress and IngressClass v1.
* [1746](https://github.com/nginxinc/kubernetes-ingress/pull/1746) Add ingressClassName field to Policy.

IMPROVEMENTS:
* [1956](https://github.com/nginxinc/kubernetes-ingress/pull/1956) Add v1alpha1 version back to policy CRD.
* [1907](https://github.com/nginxinc/kubernetes-ingress/pull/1907) Remove libs compilation for OpenTracing in Dockerfile; add Zipkin and Datadog in addition to the already supported Jaeger tracer; additionally, for NGINX we now publish a Docker image with the tracers and the OpenTracing module on DockerHub: `nginx-ic/nginx-plus-ingress:1.12.0-ot`. Also thanks to [MatyRi](https://github.com/MatyRi) for upgrading OpenTracing in [1883](https://github.com/nginxinc/kubernetes-ingress/pull/1883).
* [1788](https://github.com/nginxinc/kubernetes-ingress/pull/1788) Reload only once during the start. This significantly reduces the time it takes for an Ingress Controller pod to become ready when hundreds of Ingress or other supported resources are created in the cluster.

FIXES:
* [1926](https://github.com/nginxinc/kubernetes-ingress/pull/1926) Fix increased IC pod startup time when hundreds of VirtualServerRoutes are used
* [1712](https://github.com/nginxinc/kubernetes-ingress/pull/1712) Allow `make` to build image when .git directory is missing.

DOCUMENTATION IMPROVEMENTS:
* [1932](https://github.com/nginxinc/kubernetes-ingress/pull/1932) Add IAM instructions for NGINX Plus AWS Marketplace images.
* [1927](https://github.com/nginxinc/kubernetes-ingress/pull/1927) Fix function name comments typo. Thanks to [Sven Nebel](https://github.com/snebel29).
* [1898](https://github.com/nginxinc/kubernetes-ingress/pull/1898) Add instructions for configuring MyF5 JWT as a Docker registry secret for the F5 Container registry for NGINX Plus images.
* [1851](https://github.com/nginxinc/kubernetes-ingress/pull/1851) Update docs and examples to use networking.k8s.io/v1.
* [1765](https://github.com/nginxinc/kubernetes-ingress/pull/1765) Create documentation for pulling NGINX Plus images from the F5 Container registry.
* [1740](https://github.com/nginxinc/kubernetes-ingress/pull/1740) Publish docs using Hugo and Netlify.
* [1702](https://github.com/nginxinc/kubernetes-ingress/pull/1702) Add security recommendations documentation.

HELM CHART:
* The version of the helm chart is now 0.11.0.
* Add new parameters to the Chart: `controller.pod.extraLabels`. Added in [1884](https://github.com/nginxinc/kubernetes-ingress/pull/1884).

CHANGES:
* [1855](https://github.com/nginxinc/kubernetes-ingress/pull/1855) Update minimum Kubernetes version to 1.19; remove the `-use-ingress-class-only` command-line argument, which doesn't work with Kubernetes >= 1.19.
* [1721](https://github.com/nginxinc/kubernetes-ingress/pull/1721) Increase default reload timeout to 60s: the Ingress Controller will wait for 60s for NGINX to start or reload. Previously, the default was 4 seconds.
* [2009](https://github.com/nginxinc/kubernetes-ingress/pull/2009) Increase default upstream zone size for NGINX Plus. See the INCREASED UPSTREAM ZONES section below.
* Update NGINX Plus version to R25. **Note**: images with NGINX App Protect will continue to use R24 until App Protect 3.6 is released.
* Update NGINX version to 1.21.3.

UPGRADE:
* For NGINX, use the 2.0.0 image from our DockerHub.
* For NGINX Plus, use the 2.0.0 from the F5 Container registry or build your own image using the 2.0.0 source code.
* For Helm, use version 0.11.0 of the chart.

See the complete list of supported images for NGINX and NGINX Plus on the [Technical Specifications](https://docs.nginx.com/nginx-ingress-controller/technical-specifications/#supported-docker-images) page.

INCREASED UPSTREAM ZONES

We increased the default size of an upstream zone from 256K to 512K to accommodate a change in NGINX Plus R25. The change makes NGINX Plus allocate more memory for storing upstream server (peer) data, which means upstream server zones will use more memory to account for that new data.

The increase in the zone size is to prevent NGINX Plus configuration reload failures after an upgrade to release 1.13.0. Note that If a zone becomes full, NGINX Plus will fail to reload and fail to add more upstream servers via the API.

The new 512K default value will be able to hold ~270 upstream servers per upstream, similarly to how the old 256K value was able to hold the same number of upstream servers in the previous Ingress Controller releases. You can understand the utilization of the upstream zones via [NGINX Plus API](http://nginx.org/en/docs/http/ngx_http_api_module.html#slabs) and the [NGINX Plus dashboard](https://docs.nginx.com/nginx-ingress-controller/logging-and-monitoring/status-page/#accessing-live-activity-monitoring-dashboard) (the shared zones tab).

If you have a large number of upstream in the NGINX Plus configuration of the Ingress Controller, expect that after an upgrade NGINX Plus will consume more memory: +256K per upstream. If you don’t have upstreams with huge number of upstream serves and you’d like to reduce the memory usage of NGINX Plus, you can configure the `upstream-zone-size` [ConfigMap key](https://docs.nginx.com/nginx-ingress-controller/configuration/global-configuration/configmap-resource/#backend-services-upstreams) with a lower value. Additionally, the Ingress resource supports `nginx.org/upstream-zone-size` [annotation](https://docs.nginx.com/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations/#backend-services-upstreams) to configure zone sizes for the upstreams of an Ingress resource rather than globally.

UPDATING POLICIES

This section is only relevant if you’re running release 1.9.0 and planning to upgrade to release 2.0.0.

Release 1.10 removed the `k8s.nginx.org/v1alpha1 version` of the Policy resource and introduced the `k8s.nginx.org/v1` version. This means that to upgrade to release 1.10 users had to re-create v1alpha1 Policies with the v1 version, which caused downtime for their applications. Release 2.0.0 brings back the support for the v1alpha1 Policy, which makes it possible to upgrade from 1.9.0 to 2.0.0 release without causing downtime:

* If the Policy is marked as a preview feature in the [documentation](https://docs.nginx.com/nginx-ingress-controller/configuration/policy-resource/), make sure the -enable-preview-policies command-line argument is set in 2.0.0 Ingress Controller.
* During the upgrade, the existing Policies will not be removed.
* After the upgrade, make sure to update the Policy manifests to k8s.nginx.org/v1 version.

Please also read the [release 1.10 changelog](https://docs.nginx.com/nginx-ingress-controller/releases/#nginx-ingress-controller-1100) for the instructions on how to update Secret resources, which is also necessary since some of the Policies reference Secrets.

Note that 2.1.0 will remove support for the v1alpha1 version of the Policy.

SUPPORTED PLATFORMS:

We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the Kubernetes conformance tests. This release was fully tested on the following Kubernetes versions: 1.19-1.22.


## NGINX Ingress Controller 1.12.1

8 September 2021

CHANGES:
* Update NGINX App Protect version to 3.5.

HELM CHART:
* The version of the Helm chart is now 0.10.1.

UPGRADE:
* For NGINX, use the 1.12.1 image from our DockerHub: `nginx/nginx-ingress:1.12.1`, `nginx/nginx-ingress:1.12.1-alpine` or `nginx/nginx-ingress:1.12.1-ubi`
* For NGINX Plus, use the 1.12.1 image from the F5 Container Registry - see [the documentation here](https://docs.nginx.com/nginx-ingress-controller/installation/pulling-ingress-controller-image/)
* Alternatively, you can also build your own image using the 1.12.1 source code.
* For Helm, use version 0.10.1 of the chart.

## NGINX Ingress Controller 1.12.0

30 June 2021

OVERVIEW:

Release 1.12.0 includes:
* The introduction of pre-built containers for advanced capabilities with NGINX Plus through the F5 Container Registry.
* TransportServer supports TCP/UDP connections through the NGINX streams module adding support for matching specific health check response patterns for granular availability testing of your application, maximum connections to protect your applications from overload, supporting fine tuning of load balancing behavior, and snippets for advanced capability support as soon as you are ready to implement.
* Availability through the AWS Container marketplace supporting Elastic Kubernetes Service.
* NGINX App Protect capabilities have been extended to support the latest version and its capabilities.

You will find the complete changelog for release 1.12.0, including bug fixes, improvements, and changes below.

FEATURES:
* [1633](https://github.com/nginxinc/kubernetes-ingress/pull/1633) Support match in TransportServer health checks.
* [1619](https://github.com/nginxinc/kubernetes-ingress/pull/1619) Add AWS Marketplace Entitlement verification.
* [1480](https://github.com/nginxinc/kubernetes-ingress/pull/1480) Add max connections to TransportServer.
* [1479](https://github.com/nginxinc/kubernetes-ingress/pull/1479) Add load balancing method to TransportServer.
* [1466](https://github.com/nginxinc/kubernetes-ingress/pull/1466) Support snippets in TransportServer.

FEATURES FOR NGINX APP PROTECT:
* [1578](https://github.com/nginxinc/kubernetes-ingress/pull/1578) Add support for CSRF protection in APPolicy.
* [1513](https://github.com/nginxinc/kubernetes-ingress/pull/1513) Support multiple log security configs in Ingresses.
* [1481](https://github.com/nginxinc/kubernetes-ingress/pull/1481) Add support for user defined browsers in APPolicy.
* [1411](https://github.com/nginxinc/kubernetes-ingress/pull/1411) Add unary gRPC support in APPolicy.

IMPROVEMENTS:
* [1671](https://github.com/nginxinc/kubernetes-ingress/pull/1671) Simplify Dockerfile stages for Debian.
* [1652](https://github.com/nginxinc/kubernetes-ingress/pull/1652) Add HTTPS option to Prometheus endpoint.
* [1646](https://github.com/nginxinc/kubernetes-ingress/pull/1646) Improve Dockerfile.
* [1574](https://github.com/nginxinc/kubernetes-ingress/pull/1574) Add Docker image for Alpine with NGINX Plus.
* [1512](https://github.com/nginxinc/kubernetes-ingress/pull/1512) Don't require default server TLS secret.
* [1500](https://github.com/nginxinc/kubernetes-ingress/pull/1500) Support ssl_reject_handshake in Ingress and VS.
* [1494](https://github.com/nginxinc/kubernetes-ingress/pull/1494) Add logs around NGINX Plus binary/flag mismatch.
* [1492](https://github.com/nginxinc/kubernetes-ingress/pull/1492) Update the IC so that GlobalConfiguration is not mandatory when configured.
* Documentation improvements: [1649](https://github.com/nginxinc/kubernetes-ingress/pull/1649).

FIXES:
* [1658](https://github.com/nginxinc/kubernetes-ingress/pull/1658) Add missing njs module to the openshift-image-nap-plus image.
* [1654](https://github.com/nginxinc/kubernetes-ingress/pull/1654) Fix incorrect configuration and unexpected warnings about Secrets at the IC start.
* [1501](https://github.com/nginxinc/kubernetes-ingress/pull/1501) Fix ungraceful shutdown of NGINX.
* Documentation fixes: [1668](https://github.com/nginxinc/kubernetes-ingress/pull/1668), [1594](https://github.com/nginxinc/kubernetes-ingress/pull/1594) thanks to [shaggy245](https://github.com/shaggy245), [1563](https://github.com/nginxinc/kubernetes-ingress/pull/1563), [1551](https://github.com/nginxinc/kubernetes-ingress/pull/1551).

HELM CHART:
* The version of the helm chart is now 0.10.0.
* Add new parameters to the Chart: `prometheus.scheme`, `prometheus.secret`. Added in [1652](https://github.com/nginxinc/kubernetes-ingress/pull/1652).

CHANGES:
* [1604](https://github.com/nginxinc/kubernetes-ingress/pull/1604) Update NGINX Plus to R24. Previously, the Dockerfile had a fixed NGINX Plus version. Now the Dockerfile has a floating version that corresponds to the latest major NGINX Plus version. In the event of a patch version of NGINX Plus being released, make sure to rebuild your image to get the latest version (previously, we released a new Ingress Controller release in that case). Additionally, the AppProtect related packages are no longer fixed -- the Dockerfile will always install the latest version of the packages that work with the latest NGINX Plus version.
* [1500](https://github.com/nginxinc/kubernetes-ingress/pull/1500) Support ssl_reject_handshake in Ingress and VS. Previously, to handle missing or invalid TLS Secrets in Ingress and VirtualServer resources, the Ingress Controller would configure NGINX to break any attempts for clients to establish TLS connections to the affected hosts using `ssl_ciphers NULL;` in the NGINX configuration. The method didn't work for TLS v1.3. Now the Ingress Controller uses `ssl_reject_handshake on;`, which works for TLS v1.3.
* Update NGINX Plus version to R24.
* Update NGINX version to 1.21.0.

UPGRADE:
* For NGINX, use the 1.12.0 image from our DockerHub: `nginx/nginx-ingress:1.12.0`, `nginx/nginx-ingress:1.12.0-alpine` or `nginx-ingress:1.12.0-ubi`
* For NGINX Plus, please build your own image using the 1.12.0 source code.
* For Helm, use version 0.10.0 of the chart.

SUPPORTED PLATFORMS:

We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the Kubernetes conformance tests.  This release was fully tested on the following Kubernetes versions: 1.16-1.21.

## NGINX Ingress Controller 1.11.3

25 May 2021

CHANGES:
* Update NGINX version to 1.21.0.

HELM CHART:
* The version of the Helm chart is now 0.9.3.

UPGRADE:
* For NGINX, use the 1.11.3 image from our DockerHub: `nginx/nginx-ingress:1.11.3`, `nginx/nginx-ingress:1.11.3-alpine` or `nginx/nginx-ingress:1.11.3-ubi`
* For NGINX Plus, please build your own image using the 1.11.3 source code.
* For Helm, use version 0.9.3 of the chart.

## NGINX Ingress Controller 1.11.2

19 May 2021

CHANGES:
* Update NGINX Plus version to R23 P1.

HELM CHART:
* The version of the Helm chart is now 0.9.2.

UPGRADE:
* For NGINX, use the 1.11.2 image from our DockerHub: `nginx/nginx-ingress:1.11.2`, `nginx/nginx-ingress:1.11.2-alpine` or `nginx/nginx-ingress:1.11.2-ubi`
* For NGINX Plus, please build your own image using the 1.11.2 source code.
* For Helm, use version 0.9.2 of the chart.

## NGINX Ingress Controller 1.11.1

7 April 2021

CHANGES:
* Update NGINX version to 1.19.9.
* Update the OpenSSL libraries used in the UBI images
* Fixed documentation in the Helm README

HELM CHART:
* The version of the Helm chart is now 0.9.1.

UPGRADE:
* For NGINX, use the 1.11.1 image from our DockerHub: `nginx/nginx-ingress:1.11.1`, `nginx/nginx-ingress:1.11.1-alpine` or `nginx/nginx-ingress:1.11.1-ubi`
* For NGINX Plus, please build your own image using the 1.11.1 source code.
* For Helm, use version 0.9.1 of the chart.

## NGINX Ingress Controller 1.11.0

31 March 2021

OVERVIEW:

Release 1.11.0 includes:
* Native NGINX Ingress Controller App Protect (WAF) policy
* TransportServer improvements in terms of reliability, added features and operational aspects
* Integration of NGINX Ingress Controller with Istio service mesh

You will find the complete changelog for release 1.11.0, including bug fixes, improvements, and changes below.

FEATURES:
* [1317](https://github.com/nginxinc/kubernetes-ingress/pull/1317) Add status field to Policy resource.
* [1449](https://github.com/nginxinc/kubernetes-ingress/pull/1449) Add support for ClusterIP in upstreams in VirtualServers/VirtualServerRoutes.
* [1413](https://github.com/nginxinc/kubernetes-ingress/pull/1413) Add serverSnippets to TransportServer.
* [1425](https://github.com/nginxinc/kubernetes-ingress/pull/1425) Add status field to TransportServer resource.
* [1384](https://github.com/nginxinc/kubernetes-ingress/pull/1384) Add active health checks to TransportServer.
* [1382](https://github.com/nginxinc/kubernetes-ingress/pull/1382) Add passive health checks to TransportServer.
* [1346](https://github.com/nginxinc/kubernetes-ingress/pull/1346) Add configurable timeouts to TransportServer.
* [1297](https://github.com/nginxinc/kubernetes-ingress/pull/1297) Support custom return in the default server. Thanks to [030](https://github.com/030).

FEATURES FOR NGINX APP PROTECT:
* [1378](https://github.com/nginxinc/kubernetes-ingress/pull/1378) Add WAF Policy.

IMPROVEMENTS:
* [1420](https://github.com/nginxinc/kubernetes-ingress/pull/1420) Support IngressClassName in TransportServer.
* [1415](https://github.com/nginxinc/kubernetes-ingress/pull/1415) Handle host and listener collisions for TransportServer resource.
* [1322](https://github.com/nginxinc/kubernetes-ingress/pull/1322) Improve VirtualServer/VirtualServerRoute warnings for Policies.
* [1288](https://github.com/nginxinc/kubernetes-ingress/pull/1288) Add stricter validation for some ingress annotations.
* [1241](https://github.com/nginxinc/kubernetes-ingress/pull/1241) Refactor Dockerfile and Makefile.
* Documentation improvements: [1320](https://github.com/nginxinc/kubernetes-ingress/pull/1320), [1326](https://github.com/nginxinc/kubernetes-ingress/pull/1326), and [1377](https://github.com/nginxinc/kubernetes-ingress/pull/1377).

FIXES:
* [1457](https://github.com/nginxinc/kubernetes-ingress/pull/1457) Wait for caches to sync when the Ingress Controller starts.
* [1444](https://github.com/nginxinc/kubernetes-ingress/pull/1444) Fix setting host header in action proxy in VirtualServer/VirtualServerRoute.
* [1396](https://github.com/nginxinc/kubernetes-ingress/pull/1396) Fix reload timeout calculation for verifying NGINX reloads.

HELM CHART:
* The version of the helm chart is now 0.9.0.

CHANGES:
* [1455](https://github.com/nginxinc/kubernetes-ingress/pull/1455) Update NGINX version to 1.19.8.
* [1428](https://github.com/nginxinc/kubernetes-ingress/pull/1428) Update Nginx App Protect version to 3.0. **Note**:  [The Advanced gRPC Protection for Unary Traffic](/nginx-app-protect/configuration/#advanced-grpc-protection-for-unary-traffic) is not currently supported.

KNOWN ISSUES:
* [1448](https://github.com/nginxinc/kubernetes-ingress/issues/1448) When an Ingress Controller pod starts, it can report warnings about missing secrets for Ingress and other resources that reference secrets. Those warnings are intermittent - once the Ingress Controller fully processes the resources of the cluster, it will clear the warnings. Only after that, the Ingress Controller will become ready to accept client traffic - its readiness probe will succeed.

UPGRADE:
* For NGINX, use the 1.11.0 image from our DockerHub: `nginx/nginx-ingress:1.11.0`, `nginx/nginx-ingress:1.11.0-alpine` or `nginx-ingress:1.11.0-ubi`
* For NGINX Plus, please build your own image using the 1.11.0 source code.
* For Helm, use version 0.9.0 of the chart.
* [1241](https://github.com/nginxinc/kubernetes-ingress/pull/1241) improved the Makefile. As a result, the commands for building the Ingress Controller image were changed. See the updated commands [here](https://docs.nginx.com/nginx-ingress-controller/installation/building-ingress-controller-image/#building-the-image-and-pushing-it-to-the-private-registry).
* [1241](https://github.com/nginxinc/kubernetes-ingress/pull/1241) also consolidated all Dockerfiles into a single Dockerfile. If you customized any of the Dockerfiles, make sure to port the changes to the new Dockerfile.
* [1288](https://github.com/nginxinc/kubernetes-ingress/pull/1288) further improved validation of Ingress annotations. See this [document](https://docs.nginx.com/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations/#validation) to learn more about which annotations are validated. Note that the Ingress Controller will reject resources with invalid annotations, which means clients will see `404` responses from NGINX.  Before upgrading, ensure the Ingress resources don't have annotations with invalid values. Otherwise, after the upgrade, the Ingress Controller will reject such resources.
* [1457](https://github.com/nginxinc/kubernetes-ingress/pull/1457) fixed the bug when an Ingress Controller pod could become ready before it generated the configuration for all relevant resources in the cluster. The fix also requires that the Ingress Controller can successfully list the relevant resources from the Kubernetes API. For example, if the `-enable-custom-resources` cli argument is `true` (which is the default), the VirtualServer, VirtualServerRoute, TransportServer, and Policy CRDs must be created in the cluster, so that the Ingress Controller can list them. This is similar to other custom resources -- see the list [here](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/#create-custom-resources). Thus, before upgrading, make sure that the CRDs are created in the cluster. Otherwise, the Ingress Controller pods will not become ready.

SUPPORTED PLATFORMS:

We will provide technical support for the NGINX Ingress Controller on any Kubernetes platform that is currently supported by its provider and which passes the Kubernetes conformance tests.  This release was fully tested on the following Kubernetes versions: 1.16-1.20.

## NGINX Ingress Controller 1.10.1

16 March 2021

CHANGES:
* Update NGINX version to 1.19.8.
* Add Kubernetes 1.20 support.
* [1373](https://github.com/nginxinc/kubernetes-ingress/pull/1373), [1439](https://github.com/nginxinc/kubernetes-ingress/pull/1439), [1440](https://github.com/nginxinc/kubernetes-ingress/pull/1440): Fix various issues in the Makefile. In 1.10.0, a bug was introduced that prevented building Ingress Controller images on versions of make < 4.1.

HELM CHART:
* The version of the Helm chart is now 0.8.1.

UPGRADE:
* For NGINX, use the 1.10.1 image from our DockerHub: `nginx/nginx-ingress:1.10.1`, `nginx/nginx-ingress:1.10.1-alpine` or `nginx/nginx-ingress:1.10.1-ubi`
* For NGINX Plus, please build your own image using the 1.10.1 source code.
* For Helm, use version 0.8.1 of the chart.

## NGINX Ingress Controller 1.10.0

26 January 2021

OVERVIEW:

Release 1.10.0 includes:
* Open ID Connect authentication policy.
* Improved handling of Secret resources with extended validation and error reporting.
* Improved visibility with Prometheus metrics for the configuration workqueue and the ability to annotate NGINX logs with the metadata of Kubernetes resources.
* NGINX App Protect User-Defined signatures support.
* Improved validation of Ingress annotations.

You will find the complete changelog for release 1.10.0, including bug fixes, improvements, and changes below.

FEATURES FOR POLICY RESOURCE:
* [1304](https://github.com/nginxinc/kubernetes-ingress/pull/1304) Add Open ID Connect policy.

FEATURES FOR NGINX APP PROTECT:
* [1281](https://github.com/nginxinc/kubernetes-ingress/pull/1281) Add support for App Protect User Defined Signatures.

FEATURES:
* [1266](https://github.com/nginxinc/kubernetes-ingress/pull/1266) Add workqueue metrics to Prometheus metrics.
* [1233](https://github.com/nginxinc/kubernetes-ingress/pull/1233) Annotate tcp metrics with k8s object labels.
* [1231](https://github.com/nginxinc/kubernetes-ingress/pull/1231) Support k8s objects variables in log format.

IMPROVEMENTS:
* [1270](https://github.com/nginxinc/kubernetes-ingress/pull/1270) and [1277](https://github.com/nginxinc/kubernetes-ingress/pull/1277) Improve validation of Ingress annotations.
* [1265](https://github.com/nginxinc/kubernetes-ingress/pull/1265) Report warnings for misconfigured TLS and JWK secrets.
* [1262](https://github.com/nginxinc/kubernetes-ingress/pull/1262) Use setcap(8) only once. [1263](https://github.com/nginxinc/kubernetes-ingress/pull/1263) Use chown(8) only once. [1264](https://github.com/nginxinc/kubernetes-ingress/pull/1264) Use mkdir(1) only once. Thanks to [Sergey A. Osokin](https://github.com/osokin).
* [1256](https://github.com/nginxinc/kubernetes-ingress/pull/1256) and [1260](https://github.com/nginxinc/kubernetes-ingress/pull/1260) Improve handling of secret resources.
* [1240](https://github.com/nginxinc/kubernetes-ingress/pull/1240) Validate TLS and CA secrets.
* [1235](https://github.com/nginxinc/kubernetes-ingress/pull/1235) Use buildkit secret flag for NGINX plus images.
* Documentation improvements: [1282](https://github.com/nginxinc/kubernetes-ingress/pull/1282), [1293](https://github.com/nginxinc/kubernetes-ingress/pull/1293), [1303](https://github.com/nginxinc/kubernetes-ingress/pull/1303), [1315](https://github.com/nginxinc/kubernetes-ingress/pull/1315).

HELM CHART:
* The version of the helm chart is now 0.8.0.
* [1290](https://github.com/nginxinc/kubernetes-ingress/pull/1290) Add new preview policies parameter to chart. `controller.enablePreviewPolicies` was added.
* [1232](https://github.com/nginxinc/kubernetes-ingress/pull/1232) Replace deprecated imagePullSecrets helm setting. `controller.serviceAccount.imagePullSecrets` was removed. `controller.serviceAccount.imagePullSecretName` was added.
* [1228](https://github.com/nginxinc/kubernetes-ingress/pull/1228) Fix installation of ingressclass on Kubernetes versions `v1.18.x-*`

CHANGES:
* [1299](https://github.com/nginxinc/kubernetes-ingress/pull/1299) Update NGINX App Protect version to 2.3 and debian distribution to `debian:buster-slim`.
* [1291](https://github.com/nginxinc/kubernetes-ingress/pull/1291) Update NGINX OSS to `1.19.6`. Update NGINX Plus to `R23`.
* [1290](https://github.com/nginxinc/kubernetes-ingress/pull/1290) Graduate policy resource and accessControl policy to generally available.
* [1225](https://github.com/nginxinc/kubernetes-ingress/pull/1225) Require secrets to have types.
* [1237](https://github.com/nginxinc/kubernetes-ingress/pull/1237) Deprecate support for helm2 clients.

UPGRADE:
* For NGINX, use the 1.10.0 image from our DockerHub: `nginx/nginx-ingress:1.10.0`, `nginx/nginx-ingress:1.10.0-alpine` or `nginx-ingress:1.10.0-ubi`
* For NGINX Plus, please build your own image using the 1.10.0 source code.
* For Helm, use version 0.8.0 of the chart.
* As a result of [1270](https://github.com/nginxinc/kubernetes-ingress/pull/1270) and [1277](https://github.com/nginxinc/kubernetes-ingress/pull/1277), the Ingress Controller improved validation of Ingress annotations: more annotations are validated and validation errors are reported via events for Ingress resources. Additionally, the default behavior for invalid annotation values was changed: instead of using the default values, the Ingress Controller will reject a resource with an invalid annotation value, which will make clients see `404` responses from NGINX. See this [document](https://docs.nginx.com/nginx-ingress-controller/configuration/ingress-resources/advanced-configuration-with-annotations/#validation) to learn more. Before upgrading, ensure the Ingress resources don't have annotations with invalid values. Otherwise, after the upgrade, the Ingress Controller will reject such resources.
* In [1232](https://github.com/nginxinc/kubernetes-ingress/pull/1232) `controller.serviceAccount.imagePullSecrets` was removed. Use the new `controller.serviceAccount.imagePullSecretName` instead.
* The Policy resource was promoted to `v1`. If you used the `alpha1` version, the policies are needed to be recreated with the `v1` version. Before upgrading the Ingress Controller, run the following command to remove the `alpha1` policies CRD (that will also remove all existing `alpha1` policies):
    ```
     kubectl delete crd policies.k8s.nginx.org
    ```
  As part of the upgrade, make sure to create the `v1` policies CRD. See the corresponding instructions for the [manifests](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/#create-custom-resources) and [Helm](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/#upgrading-the-crds) installations.

  Also note that all policies except for `accessControl` are still in preview. To enable them, run the Ingress Controller with `- -enable-preview-policies` command-line argument (`controller.enablePreviewPolicies` Helm parameter).
* It is necessary to update secret resources. See the section UPDATING SECRETS below.

UPDATING SECRETS:

In [1225](https://github.com/nginxinc/kubernetes-ingress/pull/1225), as part of improving how the Ingress Controller handles secret resources, we added a requirement for secrets to be of one of the following types:
- `kubernetes.io/tls` for TLS secrets.
- `nginx.org/jwk` for JWK secrets.
- `nginx.org/ca` for CA secrets.

The Ingress Controller now ignores secrets that are not of a supported type. As a consequence, special upgrade steps are required.

Before upgrading, ensure that the secrets referenced in Ingress, VirtualServer or Policies resources are of a supported type, which is configured via the `type` field. Because that field is immutable, it is necessary to either:
* Recreate the secrets. Note that in this case, the client traffic for the affected resources will be rejected for the period during which a secret doesn't exist in the cluster.
* Create copies of the secrets and update the affected resources to reference the copies. The copies need to be of a supported type. In contrast with the previous options, this will not make NGINX reject the client traffic.

It is also necessary to update the default server secret and the wildcard secret (if it was configured) in case their type is not `kubernetes.io/tls`. The steps depend on how you installed the Ingress Controller: via manifests or Helm. Performing the steps will not lead to a disruption of the client traffic, as the Ingress Controller retains the default and wildcard secrets if they are removed.

For *manifests installation*:
1. Recreate the default server secret and the wildcard secret with the type `kubernetes.io/tls`.
1. Upgrade the Ingress Controller.

For *Helm installation*, there two cases:
1. If Helm created the secrets (you configured `controller.defaultTLS.cert` and `controller.defaultTLS.key` for the default secret and `controller.wildcardTLS.cert` and `controller.wildcardTLS.key` for the wildcard secret), then no special upgrade steps are required: during the upgrade, the Helm will remove the existing default and wildcard secrets and create new ones with different names with the type `kubernetes.io/tls`.
1.  If you created the secrets separately from Helm (you configured `controller.defaultTLS.secret` for the default secret and `controller.wildcardTLS.secret` for the wildcard secret):
    1. Recreate the secrets with the type `kubernetes.io/tls`.
    1. Upgrade to the new Helm release.

NOTES:
* Helm 2 clients are no longer supported due to reaching End of Life: https://helm.sh/blog/helm-2-becomes-unsupported/

## NGINX Ingress Controller 1.9.1

23 November 2020

CHANGES:
* Fix deployment of ingressclass resource via helm on some versions of Kubernetes.
* Update the base ubi images to 8.3.
* Renew CA cert for egress-mtls example.
* Add imagePullSecretName support to helm chart.

HELM CHART:
* The version of the Helm chart is now 0.7.1.

UPGRADE:
* For NGINX, use the 1.9.1 image from our DockerHub: `nginx/nginx-ingress:1.9.1`, `nginx/nginx-ingress:1.9.1-alpine` or `nginx/nginx-ingress:1.9.1-ubi`
* For NGINX Plus, please build your own image using the 1.9.1 source code.
* For Helm, use version 0.7.1 of the chart.

## NGINX Ingress Controller 1.9.0

20 October 2020

OVERVIEW:

Release 1.9.0 includes:
* Support for new Prometheus metrics and enhancements of the existing ones, including configuration reload reason, NGINX worker processes count, upstream latency, and more.
* Support for rate limiting, JWT authentication, ingress(client) and egress(upstream) mutual TLS via the Policy resource.
* Support for the latest Ingress resource features and the IngressClass resource.
* Support for NGINX Service Mesh.

You will find the complete changelog for release 1.9.0, including bug fixes, improvements, and changes below.

FEATURES FOR POLICY RESOURCE:
* [1180](https://github.com/nginxinc/kubernetes-ingress/pull/1180) Add support for EgressMTLS.
* [1166](https://github.com/nginxinc/kubernetes-ingress/pull/1166) Add IngressMTLS policy support.
* [1154](https://github.com/nginxinc/kubernetes-ingress/pull/1154) Add JWT policy support.
* [1120](https://github.com/nginxinc/kubernetes-ingress/pull/1120) Add RateLimit policy support.
* [1058](https://github.com/nginxinc/kubernetes-ingress/pull/1058) Support policies in VS routes and VSR subroutes.

FEATURES FOR NGINX APP PROTECT:
* [1147](https://github.com/nginxinc/kubernetes-ingress/pull/1147) Add option to specify other log destinations in AppProtect.
* [1131](https://github.com/nginxinc/kubernetes-ingress/pull/1131) Update packages and CRDs to AppProtect 2.0. This update includes features such as: [JSON Schema Validation](https://docs.nginx.com/nginx-app-protect/configuration#applying-a-json-schema), [User-Defined URLs](https://docs.nginx.com/nginx-app-protect/configuration/#user-defined-urls) and [User-Defined Parameters](https://docs.nginx.com/nginx-app-protect/configuration/#user-defined-parameters). See the [release notes](https://docs.nginx.com/nginx-app-protect/releases/#release-2-0) for a complete feature list.
* [1100](https://github.com/nginxinc/kubernetes-ingress/pull/1100) Add external references to AppProtect.
* [1085](https://github.com/nginxinc/kubernetes-ingress/pull/1085) Add installation of threat campaigns package.

FEATURES:
* [1133](https://github.com/nginxinc/kubernetes-ingress/pull/1133) Add support for IngressClass resources.
* [1130](https://github.com/nginxinc/kubernetes-ingress/pull/1130) Add prometheus latency collector.
* [1076](https://github.com/nginxinc/kubernetes-ingress/pull/1076) Add prometheus worker process metrics.
* [1075](https://github.com/nginxinc/kubernetes-ingress/pull/1075) Add support for NGINX Service Mesh internal routes.

IMPROVEMENTS:
* [1178](https://github.com/nginxinc/kubernetes-ingress/pull/1178) Resolve host collisions in VirtualServer and Ingresses.
* [1158](https://github.com/nginxinc/kubernetes-ingress/pull/1158) Support variables in action proxy headers.
* [1137](https://github.com/nginxinc/kubernetes-ingress/pull/1137) Add pod_owner label to metrics when -spire-agent-address is set.
* [1107](https://github.com/nginxinc/kubernetes-ingress/pull/1107) Extend Upstream Servers with pod_name label.
* [1099](https://github.com/nginxinc/kubernetes-ingress/pull/1099) Add reason label to total_reload metrics.
* [1088](https://github.com/nginxinc/kubernetes-ingress/pull/1088) Extend Upstream Servers and Server Zones metrics, thanks to [Raúl](https://github.com/Rulox).
* [1080](https://github.com/nginxinc/kubernetes-ingress/pull/1080) Support pathType field in the Ingress resource.
* [1078](https://github.com/nginxinc/kubernetes-ingress/pull/1078) Remove trailing blank lines in vs/vsr snippets.
* Documentation improvements: [1083](https://github.com/nginxinc/kubernetes-ingress/pull/1083), [1092](https://github.com/nginxinc/kubernetes-ingress/pull/1092), [1089](https://github.com/nginxinc/kubernetes-ingress/pull/1089), [1174](https://github.com/nginxinc/kubernetes-ingress/pull/1174), [1175](https://github.com/nginxinc/kubernetes-ingress/pull/1175), [1171](https://github.com/nginxinc/kubernetes-ingress/pull/1171).

BUGFIXES:
* [1179](https://github.com/nginxinc/kubernetes-ingress/pull/1179) Fix TransportServers in debian AppProtect image.
* [1129](https://github.com/nginxinc/kubernetes-ingress/pull/1129) Support real-ip in default server.
* [1110](https://github.com/nginxinc/kubernetes-ingress/pull/1110) Add missing threat campaigns key to AppProtect CRD.

HELM CHART:
* The version of the helm chart is now 0.7.0
* [1105](https://github.com/nginxinc/kubernetes-ingress/pull/1105) Fix GlobalConfiguration support in helm chart.
* Add new parameters to the Chart: `controller.setAsDefaultIngress`, `controller.enableLatencyMetrics`. Added in [1133](https://github.com/nginxinc/kubernetes-ingress/pull/1133) and [1148](https://github.com/nginxinc/kubernetes-ingress/pull/1148).

CHANGES:
* [1182](https://github.com/nginxinc/kubernetes-ingress/pull/1182) Update NGINX version to 1.19.3.

UPGRADE:
* For NGINX, use the 1.9.0 image from our DockerHub: `nginx/nginx-ingress:1.9.0`, `nginx/nginx-ingress:1.9.0-alpine` or `nginx-ingress:1.9.0-ubi`
* For NGINX Plus, please build your own image using the 1.9.0 source code.
* For Helm, use version 0.7.0 of the chart.

For Kubernetes >= 1.18, when upgrading using the [manifests](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-manifests/), make sure to update the [ClusterRole](https://github.com/nginxinc/kubernetes-ingress/blob/v1.9.0/deployments/rbac/rbac.yaml) and create the [IngressClass resource](https://github.com/nginxinc/kubernetes-ingress/blob/v1.9.0/deployments/common/ingress-class.yaml), which is required for Kubernetes >= 1.18. Otherwise, the Ingress Controller will fail to start. If you run multiple NGINX Ingress Controllers in the cluster, each Ingress Controller must have its own IngressClass resource. As the `-use-ingress-class-only` argument is now ignored (see NOTES), make sure your Ingress resources have the `ingressClassName` field or the `kubernetes.io/ingress.class` annotation set to the name of the IngressClass resource. Otherwise, the Ingress Controller will ignore them.

HELM UPGRADE:
* If you're using custom resources like VirtualServer and TransportServer (`controller.enableCustomResources` is set to `true`), after you run the `helm upgrade` command, the CRDs will not be upgraded. After running the `helm upgrade` command, run `kubectl apply -f deployments/helm-chart/crds` to upgrade the CRDs.
* For Kubernetes >= 1.18, a dedicated IngressClass resource, which is configured by `controller.ingressClass`, is required per helm release. Ensure `controller.ingressClass` is not set to the name of the IngressClass of other releases or Ingress Controllers. As the `controller.useIngressClassOnly` parameter is now ignored (see NOTES), make sure your Ingress resources have the `ingressClassName` field or the `kubernetes.io/ingress.class` annotation set to the value of `controller.ingressClass`. Otherwise, the Ingress Controller will ignore them.

NOTES:
* When using Kubernetes >= 1.18, the `-use-ingress-class-only` command-line argument is now ignored, and the Ingress Controller will only process resources that belong to its class. See [IngressClass doc](https://docs.nginx.com/nginx-ingress-controller/installation/running-multiple-ingress-controllers/#ingress-class) for more details.
* For Kubernetes >= 1.18, a dedicated IngressClass resource, which is configured by `controller.ingressClass`, is required per helm release. When upgrading or installing releases, ensure `controller.ingressClass` is not set to the name of the IngressClass of other releases or Ingress Controllers.

## NGINX Ingress Controller 1.8.1

14 August 2020

CHANGES:
* Update NGINX version to 1.19.2.

HELM CHART:
* The version of the Helm chart is now 0.6.1.

UPGRADE:
* For NGINX, use the 1.8.1 image from our DockerHub: `nginx/nginx-ingress:1.8.1`, `nginx/nginx-ingress:1.8.1-alpine` or `nginx/nginx-ingress:1.8.1-ubi`
* For NGINX Plus, please build your own image using the 1.8.1 source code.
* For Helm, use version 0.6.1 of the chart.


## NGINX Ingress Controller 1.8.0

22 July 2020

OVERVIEW:

Release 1.8.0 includes:
* Support for NGINX App Protect Web Application Firewall.
* Support for configuration snippets and custom template for VirtualServer and VirtualServerRoute resources.
* Support for request/response header manipulation and request URI rewriting for VirtualServer/VirtualServerRoute.
* Introducing a new configuration resource - Policy - with the first policy for IP-based access control.

You will find the complete changelog for release 1.8.0, including bug fixes, improvements, and changes below.

FEATURES FOR VIRTUALSERVER AND VIRTUALSERVERROUTE RESOURCES:
* [1036](https://github.com/nginxinc/kubernetes-ingress/pull/1036): Add VirtualServer custom template support.
* [1028](https://github.com/nginxinc/kubernetes-ingress/pull/1028): Add access control policy.
* [1019](https://github.com/nginxinc/kubernetes-ingress/pull/1019): Add VirtualServer/VirtualServerRoute snippets support.
* [1006](https://github.com/nginxinc/kubernetes-ingress/pull/1006): Add request/response modifiers to VS and VSR.
* [994](https://github.com/nginxinc/kubernetes-ingress/pull/994): Support Class Field in VS/VSR.
* [973](https://github.com/nginxinc/kubernetes-ingress/pull/973): Add status to VirtualServer and VirtualServerRoute.

FEATURES:
* [1035](https://github.com/nginxinc/kubernetes-ingress/pull/1035): Support for App Protect module.
* [1029](https://github.com/nginxinc/kubernetes-ingress/pull/1029): Add readiness endpoint.

IMPROVEMENTS:
* [995](https://github.com/nginxinc/kubernetes-ingress/pull/995): Emit event for orphaned VirtualServerRoutes.
* Documentation improvements: [946](https://github.com/nginxinc/kubernetes-ingress/pull/946) thanks to [谭九鼎](https://github.com/imba-tjd), [948](https://github.com/nginxinc/kubernetes-ingress/pull/948), [972](https://github.com/nginxinc/kubernetes-ingress/pull/972), [965](https://github.com/nginxinc/kubernetes-ingress/pull/965).

BUGFIXES:
* [1030](https://github.com/nginxinc/kubernetes-ingress/pull/1030): Fix port range validation in cli arguments.
* [953](https://github.com/nginxinc/kubernetes-ingress/pull/953): Fix error logging of master/minion ingresses.

HELM CHART:
* The version of the helm chart is now 0.6.0.
* Add new parameters to the Chart: `controller.appprotect.enable`, `controller.globalConfiguration.create`, `controller.globalConfiguration.spec`, `controller.readyStatus.enable`, `controller.readyStatus.port`, `controller.config.annotations`, `controller.reportIngressStatus.annotations`. Added in  [1035](https://github.com/nginxinc/kubernetes-ingress/pull/1035), [1034](https://github.com/nginxinc/kubernetes-ingress/pull/1034), [1029](https://github.com/nginxinc/kubernetes-ingress/pull/1029), [1003](https://github.com/nginxinc/kubernetes-ingress/pull/1003) thanks to [RubyLangdon](https://github.com/RubyLangdon).
* [1047](https://github.com/nginxinc/kubernetes-ingress/pull/1047) and [1009](https://github.com/nginxinc/kubernetes-ingress/pull/1009): Change how Helm manages the custom resource definitions (CRDs) to support installing multiple Ingress Controller releases. **Note**: If you're using the custom resources (`controller.enableCustomResources` is set to `true`), this is a breaking change. See the HELM UPGRADE section below for the upgrade instructions.

CHANGES:
* Update NGINX version to 1.19.1.
* Update NGINX Plus to R22.
* [1029](https://github.com/nginxinc/kubernetes-ingress/pull/1029): Add readiness endpoint. The Ingress Controller now exposes a readiness endpoint on port `8081` and the path `/nginx-ready`. The endpoint returns a `200` response after the Ingress Controller finishes the initial configuration of NGINX at the start. The pod template was updated to use that endpoint in a readiness probe.
* [980](https://github.com/nginxinc/kubernetes-ingress/pull/980): Enable leader election by default.

UPGRADE:
* For NGINX, use the 1.8.0 image from our DockerHub: `nginx/nginx-ingress:1.8.0`, `nginx/nginx-ingress:1.8.0-alpine` or `nginx-ingress:1.8.0-ubi`
* For NGINX Plus, please build your own image using the 1.8.0 source code.
* For Helm, use version 0.6.0 of the chart.

HELM UPGRADE:

If you're using custom resources like VirtualServer and TransportServer (`controller.enableCustomResources` is set to `true`), after you run the `helm upgrade` command, the CRDs and the corresponding custom resources will be removed from the cluster. Before upgrading, make sure to back up the custom resources. After running the `helm upgrade` command, run `kubectl apply -f deployments/helm-chart/crds` to re-install the CRDs and then restore the custom resources.

NOTES:
* As part of installing a release, Helm will install the CRDs unless that step is disabled (see the [corresponding doc](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/)). The installed CRDs include the CRDs for all Ingress Controller features, including the ones disabled by default (like App Protect with `aplogconfs.appprotect.f5.com` and `appolicies.appprotect.f5.com` CRDs).

## NGINX Ingress Controller 1.7.2

23 June 2020

CHANGES:
* Update NGINX Plus version to R22.

HELM CHART:
* The version of the Helm chart is now 0.5.2.

UPGRADE:
* For NGINX, use the 1.7.2 image from our DockerHub: `nginx/nginx-ingress:1.7.2`, `nginx/nginx-ingress:1.7.2-alpine` or `nginx/nginx-ingress:1.7.2-ubi`
* For NGINX Plus, please build your own image using the 1.7.2 source code.
* For Helm, use version 0.5.2 of the chart.

## NGINX Ingress Controller 1.7.1

4 June 2020

CHANGES:
* Update NGINX version to 1.19.0.

HELM CHART:
* The version of the Helm chart is now 0.5.1.

UPGRADE:
* For NGINX, use the 1.7.1 image from our DockerHub: `nginx/nginx-ingress:1.7.1`, `nginx/nginx-ingress:1.7.1-alpine` or `nginx/nginx-ingress:1.7.1-ubi`
* For NGINX Plus, please build your own image using the 1.7.1 source code.
* For Helm, use version 0.5.1 of the chart.

## NGINX Ingress Controller 1.7.0

30 April 2020

OVERVIEW:

Release 1.7.0 includes:
* Support for TCP, UDP, and TLS Passthrough load balancing with the new configuration resources: TransportServer and GlobalConfiguration. The resources allow users to deliver complex, non-HTTP-based applications from Kubernetes using the NGINX Ingress Controller.
* Support for error pages in VirtualServer and VirtualServerRoute resources. A user can now specify custom error responses for errors returned by backend applications or generated by NGINX, such as a 502 response.
* Improved validation of VirtualServer and VirtualServerRoute resources. kubectl and the Kubernetes API server can now detect violations of the structure of VirtualServer/VirtualServerRoute resources and return an error.
* Support for an operator which manages the lifecycle of the Ingress Controller on Kubernetes or OpenShift. See the [NGINX Ingress Operator GitHub repo](https://github.com/nginxinc/nginx-ingress-operator).

See the [1.7.0 release announcement blog post](https://www.nginx.com/blog/announcing-nginx-ingress-controller-for-kubernetes-release-1-7-0/), which includes an overview of each feature.

You will find the complete changelog for release 1.7.0, including bug fixes, improvements, and changes below.

FEATURES FOR VIRTUALSERVER AND VIRTUALSERVERROUTE RESOURCES:
* [868](https://github.com/nginxinc/kubernetes-ingress/pull/868): Add OpenAPI CRD schema validation.
* [847](https://github.com/nginxinc/kubernetes-ingress/pull/847): Add support for error pages for VS/VSR.

FEATURES:
* [902](https://github.com/nginxinc/kubernetes-ingress/pull/902): Add TransportServer and GlobalConfiguration Resources.
* [894](https://github.com/nginxinc/kubernetes-ingress/pull/894): Add Dockerfile for NGINX Open Source for Openshift.
* [857](https://github.com/nginxinc/kubernetes-ingress/pull/857): Add Openshift Dockerfile for NGINX Plus.
* [852](https://github.com/nginxinc/kubernetes-ingress/pull/852): Add default-server-access-log-off to configmap.
* [845](https://github.com/nginxinc/kubernetes-ingress/pull/845): Add log-format-escaping and stream-log-format-escaping configmap keys. Thanks to [Alexey Maslov](https://github.com/alxmsl).
* [827](https://github.com/nginxinc/kubernetes-ingress/pull/827): Add ingress class label to all Prometheus metrics.


IMPROVEMENTS:
* [850](https://github.com/nginxinc/kubernetes-ingress/pull/850): Extend redirect URI validation with protocol check in VS/VSR.
* [832](https://github.com/nginxinc/kubernetes-ingress/pull/832): Update the examples to run the `nginxdemos/nginx-hello:plain-text` image, that doesn't require root user.
* [825](https://github.com/nginxinc/kubernetes-ingress/pull/825): Add multi-stage docker builds.

BUGFIXES:
* [828](https://github.com/nginxinc/kubernetes-ingress/pull/828): Fix error messages for actions of the type return.

HELM CHART:
* The version of the helm chart is now 0.5.0.
* Add new parameters to the Chart: `controller.enableTLSPassthrough`, `controller.volumes`, `controller.volumeMounts`, `controller.priorityClassName`. Added in [921](https://github.com/nginxinc/kubernetes-ingress/pull/921), [878](https://github.com/nginxinc/kubernetes-ingress/pull/878), [807](https://github.com/nginxinc/kubernetes-ingress/pull/807) thanks to [Greg Snow](https://github.com/gsnegovskiy).

CHANGES:
* Update NGINX version to 1.17.10.
* Update NGINX Plus to R21.
* [854](https://github.com/nginxinc/kubernetes-ingress/pull/854): Update the Debian base images for NGINX Plus to `debian:buster-slim`.
* [852](https://github.com/nginxinc/kubernetes-ingress/pull/852): Add default-server-access-log-off to configmap. The access logs for the default server are now enabled by default.
* [847](https://github.com/nginxinc/kubernetes-ingress/pull/847): Add support for error pages for VS/VSR. The PR affects how the Ingress Controller generates configuration for VirtualServer and VirtualServerRoutes. See [this comment](https://github.com/nginxinc/kubernetes-ingress/pull/847) for more details.
* [827](https://github.com/nginxinc/kubernetes-ingress/pull/827): Add ingress class label to all Prometheus metrics. Every Prometheus metric exposed by the Ingress Controller now includes the label `class` with the value of the Ingress Controller class (by default `nginx`),
* [825](https://github.com/nginxinc/kubernetes-ingress/pull/825): Add multi-stage docker builds. When building the Ingress Controller image in Docker, we now use a multi-stage docker build.

UPGRADE:
* For NGINX, use the 1.7.0 image from our DockerHub: `nginx/nginx-ingress:1.7.0`, `nginx/nginx-ingress:1.7.0-alpine` or `nginx-ingress:1.7.0-ubi`
* For NGINX Plus, please build your own image using the 1.7.0 source code.
* For Helm, use version 0.5.0 of the chart.

When upgrading using the [manifests](/nginx-ingress-controller/installation/installation-with-manifests/), make sure to deploy the new TransportServer CRD (`common/ts-definition.yaml`), as it is required by the Ingress Controller. Otherwise, you will get error messages in the Ingress Controller logs.

## NGINX Ingress Controller 1.6.3

6 March 2020

CHANGES:
* Update NGINX version to 1.17.9.

HELM CHART:
* The version of the Helm chart is now 0.4.3.

UPGRADE:
* For NGINX, use the 1.6.3 image from our DockerHub: `nginx/nginx-ingress:1.6.3` or `nginx/nginx-ingress:1.6.3-alpine`
* For NGINX Plus, please build your own image using the 1.6.3 source code.
* For Helm, use version 0.4.3 of the chart.

## NGINX Ingress Controller 1.6.2

6 February 2020

CHANGES:
* Update NGINX version to 1.17.8.

HELM CHART:
* The version of the Helm chart is now 0.4.2.

UPGRADE:
* For NGINX, use the 1.6.2 image from our DockerHub: `nginx/nginx-ingress:1.6.2` or `nginx/nginx-ingress:1.6.2-alpine`
* For NGINX Plus, please build your own image using the 1.6.2 source code.
* For Helm, use version 0.4.2 of the chart.

## NGINX Ingress Controller 1.6.1

14 January 2020

CHANGES:
* Update NGINX version to 1.17.7.

HELM CHART:
* The version of the Helm chart is now 0.4.1.

UPGRADE:
* For NGINX, use the 1.6.1 image from our DockerHub: `nginx/nginx-ingress:1.6.1` or `nginx/nginx-ingress:1.6.1-alpine`
* For NGINX Plus, please build your own image using the 1.6.1 source code.
* For Helm, use version 0.4.1 of the chart.

## NGINX Ingress Controller 1.6.0

19 December 2019

OVERVIEW:

Release 1.6.0 includes:
* Improvements to VirtualServer and VirtualServerRoute resources, adding support for richer load balancing behavior, more sophisticated request routing, redirects, direct responses, and blue-green and circuit breaker patterns. The VirtualServer and VirtualServerRoute resources are enabled by default and are ready for production use.
* Support for OpenTracing, helping you to monitor and debug complex transactions.
* An improved security posture, with support to run the Ingress Controller as a non-root user.

The release announcement blog post includes the overview for each feature. See https://www.nginx.com/blog/announcing-nginx-ingress-controller-for-kubernetes-release-1-6-0/

You will find the complete changelog for release 1.6.0, including bug fixes, improvements, and changes below.

FEATURES FOR VIRTUALSERVER AND VIRTUALSERVERROUTE RESOURCES:
* [780](https://github.com/nginxinc/kubernetes-ingress/pull/780): Add support for canned responses to VS/VSR.
* [778](https://github.com/nginxinc/kubernetes-ingress/pull/778): Add redirect support in VS/VSR.
* [766](https://github.com/nginxinc/kubernetes-ingress/pull/766): Add exact matches and regex support to location paths in VS/VSR.
* [748](https://github.com/nginxinc/kubernetes-ingress/pull/748): Add TLS redirect support in Virtualserver.
* [745](https://github.com/nginxinc/kubernetes-ingress/pull/745): Improve routing rules in VS/VSR
* [728](https://github.com/nginxinc/kubernetes-ingress/pull/728): Add session persistence in VS/VSR.
* [724](https://github.com/nginxinc/kubernetes-ingress/pull/724): Add VS/VSR Prometheus metrics.
* [712](https://github.com/nginxinc/kubernetes-ingress/pull/712): Add service subselector support in vs/vsr.
* [707](https://github.com/nginxinc/kubernetes-ingress/pull/707): Emit warning events in VS/VSR.
* [701](https://github.com/nginxinc/kubernetes-ingress/pull/701): Add support queue in upstreams for plus in VS/VSR.
* [693](https://github.com/nginxinc/kubernetes-ingress/pull/693): Add ServerStatusZones support in vs/vsr.
* [670](https://github.com/nginxinc/kubernetes-ingress/pull/670): Add buffering support for vs/vsr.
* [660](https://github.com/nginxinc/kubernetes-ingress/pull/660): Add ClientBodyMaxSize support in vs/vsr.
* [659](https://github.com/nginxinc/kubernetes-ingress/pull/659): Support configuring upstream zone sizes in VS/VSR.
* [655](https://github.com/nginxinc/kubernetes-ingress/pull/655): Add slow-start support in vs/vsr.
* [653](https://github.com/nginxinc/kubernetes-ingress/pull/653): Add websockets support for vs/vsr upstreams.
* [641](https://github.com/nginxinc/kubernetes-ingress/pull/641): Add support for ExternalName Services for vs/vsr.
* [635](https://github.com/nginxinc/kubernetes-ingress/pull/635): Add HealthChecks support for vs/vsr.
* [634](https://github.com/nginxinc/kubernetes-ingress/pull/634): Add Active Connections support to vs/vsr.
* [628](https://github.com/nginxinc/kubernetes-ingress/pull/628): Add retries support for vs/vsr.
* [621](https://github.com/nginxinc/kubernetes-ingress/pull/621): Add TLS support for vs/vsr upstreams.
* [617](https://github.com/nginxinc/kubernetes-ingress/pull/617): Add keepalive support to vs/vsr.
* [612](https://github.com/nginxinc/kubernetes-ingress/pull/612): Add timeouts support to vs/vsr.
* [607](https://github.com/nginxinc/kubernetes-ingress/pull/607): Add fail-timeout and max-fails support to vs/vsr.
* [596](https://github.com/nginxinc/kubernetes-ingress/pull/596): Add lb-method support in vs and vsr.

FEATURES:
* [750](https://github.com/nginxinc/kubernetes-ingress/pull/750): Add support for health status uri customisation.
* [691](https://github.com/nginxinc/kubernetes-ingress/pull/691): Helper Functions for custom annotations.
* [631](https://github.com/nginxinc/kubernetes-ingress/pull/631): Add max_conns support for NGINX plus.
* [629](https://github.com/nginxinc/kubernetes-ingress/pull/629): Added upstream zone directive annotation. Thanks to [Victor Regalado](https://github.com/vrrs).
* [616](https://github.com/nginxinc/kubernetes-ingress/pull/616): Add proxy-send-timeout to configmap key and annotation.
* [615](https://github.com/nginxinc/kubernetes-ingress/pull/615): Add support for Opentracing.
* [614](https://github.com/nginxinc/kubernetes-ingress/pull/614): Add max-conns annotation. Thanks to [Victor Regalado](https://github.com/vrrs).


IMPROVEMENTS:
* [678](https://github.com/nginxinc/kubernetes-ingress/pull/678): Increase defaults for server-names-hash-max-size and servers-names-hash-bucket-size ConfigMap keys.
* [694](https://github.com/nginxinc/kubernetes-ingress/pull/694): Reject VS/VSR resources with enabled plus features for OSS.
* Documentation improvements: [713](https://github.com/nginxinc/kubernetes-ingress/pull/713) thanks to [Matthew Wahner](https://github.com/mattwahner).

BUGFIXES:
* [788](https://github.com/nginxinc/kubernetes-ingress/pull/788): Fix VSR updates when namespace is set implicitly.
* [736](https://github.com/nginxinc/kubernetes-ingress/pull/736): Init Ingress labeled metrics on start.
* [686](https://github.com/nginxinc/kubernetes-ingress/pull/686): Check if config map created for leader-election.
* [664](https://github.com/nginxinc/kubernetes-ingress/pull/664): Fix reporting events for Ingress minions.
* [632](https://github.com/nginxinc/kubernetes-ingress/pull/632): Fix hsts support when not using SSL. Thanks to [Martín Fernández](https://github.com/bilby91).

HELM CHART:
* The version of the helm chart is now 0.4.0.
* Add new parameters to the Chart: `controller.healthCheckURI`, `controller.resources`, `controller.logLevel`, `controller.customPorts`, `controller.service.customPorts`. Added in [750](https://github.com/nginxinc/kubernetes-ingress/pull/750), [636](https://github.com/nginxinc/kubernetes-ingress/pull/636) thanks to [Guilherme Oki](https://github.com/guilhermeoki), [600](https://github.com/nginxinc/kubernetes-ingress/pull/600), [581](https://github.com/nginxinc/kubernetes-ingress/pull/581) thanks to [Alex Meijer](https://github.com/ameijer-corsha).
* [722](https://github.com/nginxinc/kubernetes-ingress/pull/722): Fix trailing leader election cm when using helm. This change might lead to a failed upgrade. See the helm upgrade instruction below.
* [573](https://github.com/nginxinc/kubernetes-ingress/pull/573): Use Controller name value for app selectors.

CHANGES:
* Update NGINX versions to 1.17.6.
* Update NGINX Plus version to R20.
* [799](https://github.com/nginxinc/kubernetes-ingress/pull/779): Enable CRDs by default. VirtualServer and VirtualServerRoute resources are now enabled by default.
* [772](https://github.com/nginxinc/kubernetes-ingress/pull/772): Update VS/VSR version from v1alpha1 to v1. Make sure to update the `apiVersion` of your VirtualServer and VirtualServerRoute resources.
* [748](https://github.com/nginxinc/kubernetes-ingress/pull/748): Add TLS redirect support in VirtualServer. The `redirect-to-https` and `ssl-redirect` ConfigMap keys no longer have any effect on generated configs for VirtualServer resources.
* [745](https://github.com/nginxinc/kubernetes-ingress/pull/745): Improve routing rules. Update the spec of VirtualServer and VirtualServerRoute accordingly. See YAML examples of the changes [here](https://github.com/nginxinc/kubernetes-ingress/pull/745).
* [710](https://github.com/nginxinc/kubernetes-ingress/pull/710): Run IC as non-root. Make sure to use the updated manifests to install/upgrade the Ingress Controller.
* [603](https://github.com/nginxinc/kubernetes-ingress/pull/603): Update apiVersion in Deployments and DaemonSets to apps/v1.

UPGRADE:
* For NGINX, use the 1.6.0 image from our DockerHub: `nginx/nginx-ingress:1.6.0` or `nginx/nginx-ingress:1.6.0-alpine`
* For NGINX Plus, please build your own image using the 1.6.0 source code.
* For Helm, use version 0.4.0 of the chart.

HELM UPGRADE:

If leader election (the `controller.reportIngressStatus.enableLeaderElection` parameter) is enabled, when upgrading to the new version of the Helm chart:
1. Make sure to specify a new ConfigMap lock name (`controller.reportIngressStatus.leaderElectionLockName`) different from the one that was created by the current version. To find out the current name, check ConfigMap resources in the namespace where the Ingress Controller is running.
1. After the upgrade, delete the old ConfigMap.

Otherwise, the helm upgrade will not succeed.

## Previous Releases

To see the previous releases, see the [Releases page](https://github.com/nginxinc/kubernetes-ingress/releases) on the Ingress Controller GitHub repo.
