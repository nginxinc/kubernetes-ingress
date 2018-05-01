# Changelog

### 1.2.0

* [279](https://github.com/nginxinc/kubernetes-ingress/pull/279): Update dependencies.
* [278](https://github.com/nginxinc/kubernetes-ingress/pull/278): Fix mergeable Ingress types.
* [277](https://github.com/nginxinc/kubernetes-ingress/pull/277): Support grpc error responses.
* [276](https://github.com/nginxinc/kubernetes-ingress/pull/276): Add gRPC support.
* [274](https://github.com/nginxinc/kubernetes-ingress/pull/274): Change the default load balancing method to least_conn. 
* [272](https://github.com/nginxinc/kubernetes-ingress/pull/272): Move nginx-ingress image to the official nginx DockerHub.
* [268](https://github.com/nginxinc/kubernetes-ingress/pull/268): Correct Mergeable Types misspelling and optimize blacklists. Thanks to [Fernando Diaz](https://github.com/diazjf). 
* [266](https://github.com/nginxinc/kubernetes-ingress/pull/266): Add support for passive health checks.
* [261](https://github.com/nginxinc/kubernetes-ingress/pull/261): Update Customization Example.
* [258](https://github.com/nginxinc/kubernetes-ingress/pull/258): Handle annotations and conflicting paths for MergeableTypes. Thanks to [Fernando Diaz](https://github.com/diazjf).
* [256](https://github.com/nginxinc/kubernetes-ingress/pull/256): Add helm chart support. 
* [249](https://github.com/nginxinc/kubernetes-ingress/pull/249): Add support for prometheus for Plus.
* [241](https://github.com/nginxinc/kubernetes-ingress/pull/241): Update the doc about building the Docker image.
* [240](https://github.com/nginxinc/kubernetes-ingress/pull/240): Use new NGINX Plus API.
* [239](https://github.com/nginxinc/kubernetes-ingress/pull/239): Fix a typo in a variable name. Thanks to [Tony Li](https://github.com/mysterytony).
* [238](https://github.com/nginxinc/kubernetes-ingress/pull/238): Remove apt-get upgrade from Plus Dockerfile.
* [237](https://github.com/nginxinc/kubernetes-ingress/pull/237): Add unit test for ingress-class handling.
* [236](https://github.com/nginxinc/kubernetes-ingress/pull/236): Always respect `-ingress-class` option. Thanks to [Nick Novitski](https://github.com/nicknovitski).
* [235](https://github.com/nginxinc/kubernetes-ingress/pull/235): Change the base image to Debian Stretch for Plus controller.
* [234](https://github.com/nginxinc/kubernetes-ingress/pull/234): Update installation manifests and instructions.
* [233](https://github.com/nginxinc/kubernetes-ingress/pull/233): Add docker build options to Makefile.
* [231](https://github.com/nginxinc/kubernetes-ingress/pull/231): Prevent a possible failure of building Plus image.
* Documentation Fixes: [248](https://github.com/nginxinc/kubernetes-ingress/pull/248), thanks to [zariye](https://github.com/zariye). [252](https://github.com/nginxinc/kubernetes-ingress/pull/252). [270](https://github.com/nginxinc/kubernetes-ingress/pull/270).
* Update NGINX version to 1.13.12.
* Update NGINX Plus version to R15 P1.


### 1.1.1

* [228](https://github.com/nginxinc/kubernetes-ingress/pull/228): Add worker-rlimit-nofile configmap key. Thanks to [Aleksandr Lysenko](https://github.com/Sarga).
* [223](https://github.com/nginxinc/kubernetes-ingress/pull/223): Add worker-connections configmap key. Thanks to [Aleksandr Lysenko](https://github.com/Sarga).
* Update NGINX version to 1.13.8.

### 1.1.0

* [221](https://github.com/nginxinc/kubernetes-ingress/pull/221): Add git commit info to the IC log.
* [220](https://github.com/nginxinc/kubernetes-ingress/pull/220): Update dependencies.
* [213](https://github.com/nginxinc/kubernetes-ingress/pull/213): Add main snippets to allow Main context customization. Thanks to [Dewen Kong](https://github.com/kongdewen).
* [211](https://github.com/nginxinc/kubernetes-ingress/pull/211): Minimize the number of configuration reloads when the Ingress controller starts; fix a problem with endpoints updates for Plus.
* [208](https://github.com/nginxinc/kubernetes-ingress/pull/208): Add worker-shutdown-timeout configmap key. Thanks to [Aleksandr Lysenko](https://github.com/Sarga).
* [199](https://github.com/nginxinc/kubernetes-ingress/pull/199): Add support for Kubernetes ssl-redirect annotation. Thanks to [Luke Seelenbinder](https://github.com/lseelenbinder).
* [194](https://github.com/nginxinc/kubernetes-ingress/pull/194) Add keepalive configmap key and annotation.
* [193](https://github.com/nginxinc/kubernetes-ingress/pull/193): Add worker-cpu-affinity configmap key.
* [192](https://github.com/nginxinc/kubernetes-ingress/pull/192): Add worker-processes configmap key.
* [186](https://github.com/nginxinc/kubernetes-ingress/pull/186): Fix hardcoded controller class. Thanks to [Serhii M](https://github.com/SiriusRed).
* [184](https://github.com/nginxinc/kubernetes-ingress/pull/184): Return a meaningful error when there is no cert and key for the default server.
* Update NGINX version to 1.13.7.
* Makefile updates: golang container was updated to 1.9.

### 1.0.0

* [175](https://github.com/nginxinc/kubernetes-ingress/pull/175): Add support for JWT for NGINX Plus.
* [171](https://github.com/nginxinc/kubernetes-ingress/pull/171): Allow NGINX to listen on non-standard ports. Thanks to [Stanislav Seletskiy](https://github.com/seletskiy).
* [170](https://github.com/nginxinc/kubernetes-ingress/pull/170): Add the default server. **Note**: The Ingress controller will fail to start if there are no cert and key for the default server. You can pass a TLS Secret for the default server as an argument to the Ingress controller or add a cert and a key to the Docker image. 
* [169](https://github.com/nginxinc/kubernetes-ingress/pull/169): Ignore Ingress resources with empty hostnames.
* [168](https://github.com/nginxinc/kubernetes-ingress/pull/168): Add the `nginx.org/lb-method` annotation. Thanks to [Sajal Kayan](https://github.com/sajal).
* [166](https://github.com/nginxinc/kubernetes-ingress/pull/166): Watch Secret resources for updates. **Note**: If a Secret referenced by one or more Ingress resources becomes invalid or gets removed, the configuration for those Ingress resources will be disabled until there is a valid Secret.
* [160](https://github.com/nginxinc/kubernetes-ingress/pull/160): Add support for events. See the details [here](https://github.com/nginxinc/kubernetes-ingress/pull/160).
* [157](https://github.com/nginxinc/kubernetes-ingress/pull/157): Add graceful termination - when the Ingress controller receives `SIGTERM`, it shutdowns itself as well as NGINX, using `nginx -s quit`.

### 0.9.0

* [156](https://github.com/nginxinc/kubernetes-ingress/pull/156): Write a pem file with an SSL certificate and key atomically.
* [155](https://github.com/nginxinc/kubernetes-ingress/pull/155): Remove http2 annotation (http/2 can be enabled globally in the ConfigMap).
* [154](https://github.com/nginxinc/kubernetes-ingress/pull/154): Merge NGINX and NGINX Plus Ingress controller implementations.
* [151](https://github.com/nginxinc/kubernetes-ingress/pull/151): Use k8s.io/client-go.
* [146](https://github.com/nginxinc/kubernetes-ingress/pull/146): Fix health status.
* [141](https://github.com/nginxinc/kubernetes-ingress/pull/141): Set `worker_processes` to `auto` in NGINX configuration. Thanks to [Andreas Krüger](https://github.com/woopstar).
* [140](https://github.com/nginxinc/kubernetes-ingress/pull/140): Fix an error message. Thanks to [Andreas Krüger](https://github.com/woopstar).
* Update NGINX to version 1.13.3.

### 0.8.1

* Update NGINX version to 1.13.0.

### 0.8.0

* [117](https://github.com/nginxinc/kubernetes-ingress/pull/117): Add a customization option: location-snippets, server-snippets and http-snippets. Thanks to [rchicoli](https://github.com/rchicoli).
* [116](https://github.com/nginxinc/kubernetes-ingress/pull/116): Add support for the 301 redirect to https based on the `http_x_forwarded_proto` header. Thanks to [Chris](https://github.com/cwhenderson20).
* Update NGINX version to 1.11.13.
* Makefile updates: gcloud docker push command; golang container was updated to 1.8.
* Documentation fixes: [113](https://github.com/nginxinc/kubernetes-ingress/pull/113). Thanks to [Linus Lewandowski](https://github.com/LEW21).

### 0.7.0

* [108](https://github.com/nginxinc/kubernetes-ingress/pull/108): Support for the `server_tokens` directive via the annotation and in the configmap. Thanks to [David Radcliffe](https://github.com/dwradcliffe).
* [103](https://github.com/nginxinc/kubernetes-ingress/pull/103): Improve error reporting when NGINX fails to start.
* [100](https://github.com/nginxinc/kubernetes-ingress/pull/100): Add the health check location. Thanks to [Julian](https://github.com/jmastr).
* [95](https://github.com/nginxinc/kubernetes-ingress/pull/95): Fix the runtime.TypeAssertionError issue, which sometimes occurred when deleting resources. Thanks to [Tang Le](https://github.com/tangle329).
* [93](https://github.com/nginxinc/kubernetes-ingress/pull/93): Fix overwriting of Secrets with the same name from different namespaces.
* [92](https://github.com/nginxinc/kubernetes-ingress/pull/92/files): Add overwriting of the HSTS header. Previously, when HSTS was enabled, if a backend issued the HSTS header, the controller would add the second HSTS header. Now the controller overwrites the HSTS header, if a backend also issues it.
* [91](https://github.com/nginxinc/kubernetes-ingress/pull/91):
Fix the issue with single service Ingress resources without any Ingress rules: the controller didn't pick up any updates of the endpoints of the service of such an Ingress resource. Thanks to [Tang Le](https://github.com/tangle329).
* [88](https://github.com/nginxinc/kubernetes-ingress/pull/88): Support for the `proxy_hide_header` and the `proxy_pass_header` directives via annotations and in the configmap. Thanks to [Nico Schieder](https://github.com/thetechnick).
* [85](https://github.com/nginxinc/kubernetes-ingress/pull/85): Add the configmap settings to support perfect forward secrecy. Thanks to [Nico Schieder](https://github.com/thetechnick).
* [84](https://github.com/nginxinc/kubernetes-ingress/pull/84): Secret retry: If a certificate Secret referenced in an Ingress object is not found,
the Ingress controller will reject the Ingress object. but retries every 5s. Thanks to [Nico Schieder](https://github.com/thetechnick).
* [81](https://github.com/nginxinc/kubernetes-ingress/pull/81): Add configmap options to turn on the PROXY protocol. Thanks to [Nico Schieder](https://github.com/thetechnick).
* Update NGINX version to 1.11.8.
* Documentation fixes: [104](https://github.com/nginxinc/kubernetes-ingress/pull/104/files) and [97](https://github.com/nginxinc/kubernetes-ingress/pull/97/files). Thanks to [Ruilin Huang](https://github.com/hrl) and [Justin Garrison](https://github.com/rothgar).

### 0.6.0

* [75](https://github.com/nginxinc/kubernetes-ingress/pull/75): Add the HSTS settings in the configmap and annotations. Thanks to [Nico Schieder](https://github.com/thetechnick).
* [74](https://github.com/nginxinc/kubernetes-ingress/pull/74): Fix the issue of the `kubernetes.io/ingress.class` annotation handling. Thanks to [Tang Le](https://github.com/tangle329).
* [70](https://github.com/nginxinc/kubernetes-ingress/pull/70): Add support for the alpine-based image for the NGINX controller.
* [68](https://github.com/nginxinc/kubernetes-ingress/pull/68): Support for proxy-buffering settings in the configmap and annotations. Thanks to [Mark Daniel Reidel](https://github.com/df-mreidel).
* [66](https://github.com/nginxinc/kubernetes-ingress/pull/66): Support for custom log-format in the configmap. Thanks to [Mark Daniel Reidel](https://github.com/df-mreidel).
* [65](https://github.com/nginxinc/kubernetes-ingress/pull/65): Add HTTP/2 as an option in the configmap and annotations. Thanks to [Nico Schieder](https://github.com/thetechnick).
* The NGINX Plus controller image is now based on Ubuntu Xenial.

### 0.5.0

* Update NGINX version to 1.11.5.
* [64](https://github.com/nginxinc/kubernetes-ingress/pull/64): Add the `nginx.org/rewrites` annotation, which allows to rewrite the URI of a request before sending it to the application. Thanks to [Julian](https://github.com/jmastr).
* [62](https://github.com/nginxinc/kubernetes-ingress/pull/62): Add the `nginx.org/ssl-services` annotation, which allows load balancing of HTTPS applications. Thanks to [Julian](https://github.com/jmastr).

### 0.4.0

* [54](https://github.com/nginxinc/kubernetes-ingress/pull/54): Previously, when specifying the port of a service in an Ingress rule, you had to use the value of the target port of that port of the service, which was incorrect. Now you must use the port value or the name of the port of the service instead of the target port value. **Note**: Please make necessary changes to your Ingress resources, if ports of your services have different values of the port and the target port fields.
* [55](https://github.com/nginxinc/kubernetes-ingress/pull/55): Add support for the `kubernetes.io/ingress.class` annotation in Ingress resources.
* [58](https://github.com/nginxinc/kubernetes-ingress/pull/58): Add the version information to the controller. For each version of the NGINX controller, you can find a corresponding image on [DockerHub](https://hub.docker.com/r/nginxdemos/nginx-ingress/tags/) with a tag equal to the version. The latest version is available through the `latest` tag.

The previous version was 0.3


### Notes

* Except when mentioned otherwise, the controller refers both to the NGINX and the NGINX Plus Ingress controllers.
