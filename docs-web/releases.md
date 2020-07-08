# Releases 

## NGINX Ingress Controller 1.8.0

### 1.8.0

14 July 2020

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
* Add new parameters to the Chart: `controller.appprotect.enable`, `controller.globalConfiguration.create`, `controller.globalConfiguration.spec`, `controller.readyStatus.enable`, `controller.readyStaus.port`, `controller.config.annotations`, `controller.reportIngressStatus.annotations`. Added in  [1035](https://github.com/nginxinc/kubernetes-ingress/pull/1035), [1034](https://github.com/nginxinc/kubernetes-ingress/pull/1034), [1029](https://github.com/nginxinc/kubernetes-ingress/pull/1029), [1003](https://github.com/nginxinc/kubernetes-ingress/pull/1003) thanks to [RubyLangdon](https://github.com/RubyLangdon). 
* [1047](https://github.com/nginxinc/kubernetes-ingress/pull/1047) and [1009](https://github.com/nginxinc/kubernetes-ingress/pull/1009): Change how Helm manages the custom resource defintions (CRDs) to support installing multiple Ingress Controller releases. **Note**: If you're using the custom resources (`controller.enableCustomResources` is set to `true`), this is a breaking change. See the HELM UPGRADE section below for the upgrade instructions.

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
* As part of installing a release, Helm will install the CRDs unless that step is disabled (see the [corresponding doc](https://docs.nginx.com/nginx-ingress-controller/installation/installation-with-helm/)). The installed CRDs include  the CRDs for all Ingress Controller features, including the ones disabled by default (like App Protect with `aplogconfs.appprotect.f5.com` and `appolicies.appprotect.f5.com` CRDs).

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
