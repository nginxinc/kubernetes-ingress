---
docs: DOCS-1459
doctypes:
- troubleshooting
draft: false
tags:
- docs
title: Support
toc: true
weight: 100
---
NGINX Ingress Controller adheres to the support policy detailed in the following knowledge base article: [K000140156](https://my.f5.com/manage/s/article/K000140156).

In order to open a support ticket, F5 would like additional information to better understand the problem.

The [nginx-supportpkg-for-k8s](https://github.com/nginxinc/nginx-supportpkg-for-k8s) plugin is such a tool used to obtain additional information needed by F5 Technical Support when troubleshooting your issue.

The plugin leverages [krew](https://krew.sigs.k8s.io), the plugin manager for [kubectl](https://kubernetes.io/docs/reference/kubectl/) command-line tool of kubernetes.

The plugin may collect some or all of the following global and namespace-specific information:
* k8s version, nodes information and CRDs
* pods logs
* list of pods, events, configmaps, services, deployments, statefulsets, replicasets and leases
* k8s metrics
* helm deployments
* nginx -T output from NGINX related pods

This plugin **DOES NOT** collect secrets or coredumps.

Please visit the [project’s GitHub page](https://github.com/nginxinc/nginx-supportpkg-for-k8s) for further details.

When used, the plugin will generate a tarball of the collected information which can be shared via the support channels.


**Support Channels:**

- If you experience issues with NGINX Ingress Controller, please [open an issue](https://github.com/nginxinc/kubernetes-ingress/issues/new?assignees=&labels=bug%2Cneeds+triage&projects=&template=BUG-REPORT.yml&title=%5BBug%5D%3A+) in GitHub.

- If you have any enhancement requests, please [open a feature request](https://github.com/nginxinc/kubernetes-ingress/issues/new?assignees=&labels=proposal&projects=&template=feature_request.md&title=) in GitHub.

- If you have any ideas or suggestions to discuss, please [open an idea discussion](https://github.com/nginxinc/kubernetes-ingress/discussions/categories/ideas) in Github.

- You can contact us directly, by sending an email to [kubernetes@nginx.com](mailto:kubernetes@nginx.com) or on the [NGINX Community Slack channel of NGINX Ingress Controller](https://nginxcommunity.slack.com/channels/nginx-ingress-controller).