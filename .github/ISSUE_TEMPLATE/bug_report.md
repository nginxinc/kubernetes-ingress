---
name: Bug report
about: Create a report to help us improve
title: ''
labels: ''
assignees: ''

---

**Describe the bug**
A clear and concise description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Deploy x to '...' using some.yaml
2. View logs on '....'
3. See error

**Expected behavior**
A clear and concise description of what you expected to happen.

**Your environment**
* Version of the Ingress Controller - release version or a specific commit
* Version of Kubernetes
* Kubernetes platform (e.g. Mini-kube or GCP)
* Using NGINX or NGINX Plus
<!-- output from `docker inspect --format '{{ json .Config.Labels }}' <docker image> | jq` if available -->

**Additional context**
Add any other context about the problem here. Any log files you want to share.
