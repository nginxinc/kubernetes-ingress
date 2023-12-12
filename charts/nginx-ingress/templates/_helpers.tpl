{{/* vim: set filetype=mustache: */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "nginx-ingress.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "nginx-ingress.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Create a default fully qualified controller name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "nginx-ingress.controller.fullname" -}}
{{- printf "%s-%s" (include "nginx-ingress.fullname" .) .Values.controller.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified controller service name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
*/}}
{{- define "nginx-ingress.controller.service.name" -}}
{{- default (include "nginx-ingress.controller.fullname" .) .Values.serviceNameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "nginx-ingress.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "nginx-ingress.labels" -}}
helm.sh/chart: {{ include "nginx-ingress.chart" . }}
{{ include "nginx-ingress.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "nginx-ingress.selectorLabels" -}}
{{- if .Values.controller.selectorLabels -}}
{{ toYaml .Values.controller.selectorLabels }}
{{- else -}}
app.kubernetes.io/name: {{ include "nginx-ingress.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}
{{- end -}}

{{/*
Expand the name of the configmap.
*/}}
{{- define "nginx-ingress.configName" -}}
{{- if .Values.controller.customConfigMap -}}
{{ .Values.controller.customConfigMap }}
{{- else -}}
{{- default (include "nginx-ingress.fullname" .) .Values.controller.config.name -}}
{{- end -}}
{{- end -}}

{{/*
Expand leader election lock name.
*/}}
{{- define "nginx-ingress.leaderElectionName" -}}
{{- if .Values.controller.reportIngressStatus.leaderElectionLockName -}}
{{ .Values.controller.reportIngressStatus.leaderElectionLockName }}
{{- else -}}
{{- printf "%s-%s" (include "nginx-ingress.fullname" .) "leader-election" -}}
{{- end -}}
{{- end -}}

{{/*
Expand service account name.
*/}}
{{- define "nginx-ingress.serviceAccountName" -}}
{{- default (include "nginx-ingress.fullname" .) .Values.controller.serviceAccount.name -}}
{{- end -}}

{{/*
Expand default TLS name.
*/}}
{{- define "nginx-ingress.defaultTLSName" -}}
{{- printf "%s-%s" (include "nginx-ingress.fullname" .) "default-server-tls" -}}
{{- end -}}

{{/*
Expand wildcard TLS name.
*/}}
{{- define "nginx-ingress.wildcardTLSName" -}}
{{- printf "%s-%s" (include "nginx-ingress.fullname" .) "wildcard-tls" -}}
{{- end -}}

{{- define "nginx-ingress.tag" -}}
{{- default .Chart.AppVersion .Values.controller.image.tag -}}
{{- end -}}

{{/*
Expand image name.
*/}}
{{- define "nginx-ingress.image" -}}
{{- if .Values.controller.image.digest -}}
{{- printf "%s@%s" .Values.controller.image.repository .Values.controller.image.digest -}}
{{- else -}}
{{- printf "%s:%s" .Values.controller.image.repository (include "nginx-ingress.tag" .) -}}
{{- end -}}
{{- end -}}

{{- define "nginx-ingress.prometheus.serviceName" -}}
{{- printf "%s-%s" (include "nginx-ingress.fullname" .) "prometheus-service"  -}}
{{- end -}}

{{/*
Build the args for the service binary.
*/}}
{{- define "nginx-ingress.args" -}}
- -nginx-plus={{ .Values.controller.nginxplus }}
- -nginx-reload-timeout={{ .Values.controller.nginxReloadTimeout }}
- -enable-app-protect={{ .Values.controller.appprotect.enable }}
{{- if and .Values.controller.appprotect.enable .Values.controller.appprotect.logLevel }}
- -app-protect-log-level={{ .Values.controller.appprotect.logLevel }}
{{ end }}
- -enable-app-protect-dos={{ .Values.controller.appprotectdos.enable }}
{{- if .Values.controller.appprotectdos.enable }}
- -app-protect-dos-debug={{ .Values.controller.appprotectdos.debug }}
- -app-protect-dos-max-daemons={{ .Values.controller.appprotectdos.maxDaemons }}
- -app-protect-dos-max-workers={{ .Values.controller.appprotectdos.maxWorkers }}
- -app-protect-dos-memory={{ .Values.controller.appprotectdos.memory }}
{{ end }}
- -nginx-configmaps=$(POD_NAMESPACE)/{{ include "nginx-ingress.configName" . }}
{{- if .Values.controller.defaultTLS.secret }}
- -default-server-tls-secret={{ .Values.controller.defaultTLS.secret }}
{{ else if and (.Values.controller.defaultTLS.cert) (.Values.controller.defaultTLS.key) }}
- -default-server-tls-secret=$(POD_NAMESPACE)/{{ include "nginx-ingress.defaultTLSName" . }}
{{- end }}
- -ingress-class={{ .Values.controller.ingressClass.name }}
{{- if .Values.controller.watchNamespace }}
- -watch-namespace={{ .Values.controller.watchNamespace }}
{{- end }}
{{- if .Values.controller.watchNamespaceLabel }}
- -watch-namespace-label={{ .Values.controller.watchNamespaceLabel }}
{{- end }}
{{- if .Values.controller.watchSecretNamespace }}
- -watch-secret-namespace={{ .Values.controller.watchSecretNamespace }}
{{- end }}
- -health-status={{ .Values.controller.healthStatus }}
- -health-status-uri={{ .Values.controller.healthStatusURI }}
- -nginx-debug={{ .Values.controller.nginxDebug }}
- -v={{ .Values.controller.logLevel }}
- -nginx-status={{ .Values.controller.nginxStatus.enable }}
{{- if .Values.controller.nginxStatus.enable }}
- -nginx-status-port={{ .Values.controller.nginxStatus.port }}
- -nginx-status-allow-cidrs={{ .Values.controller.nginxStatus.allowCidrs }}
{{- end }}
{{- if .Values.controller.reportIngressStatus.enable }}
- -report-ingress-status
{{- if .Values.controller.reportIngressStatus.ingressLink }}
- -ingresslink={{ .Values.controller.reportIngressStatus.ingressLink }}
{{- else if .Values.controller.reportIngressStatus.externalService }}
- -external-service={{ .Values.controller.reportIngressStatus.externalService }}
{{- else if and (.Values.controller.service.create) (eq .Values.controller.service.type "LoadBalancer") }}
- -external-service={{ include "nginx-ingress.controller.service.name" . }}
{{- end }}
{{- end }}
- -enable-leader-election={{ .Values.controller.reportIngressStatus.enableLeaderElection }}
{{- if .Values.controller.reportIngressStatus.enableLeaderElection }}
- -leader-election-lock-name={{ include "nginx-ingress.leaderElectionName" . }}
{{- end }}
{{- if .Values.controller.wildcardTLS.secret }}
- -wildcard-tls-secret={{ .Values.controller.wildcardTLS.secret }}
{{- else if and .Values.controller.wildcardTLS.cert .Values.controller.wildcardTLS.key }}
- -wildcard-tls-secret=$(POD_NAMESPACE)/{{ include "nginx-ingress.wildcardTLSName" . }}
{{- end }}
- -enable-prometheus-metrics={{ .Values.prometheus.create }}
- -prometheus-metrics-listen-port={{ .Values.prometheus.port }}
- -prometheus-tls-secret={{ .Values.prometheus.secret }}
- -enable-service-insight={{ .Values.serviceInsight.create }}
- -service-insight-listen-port={{ .Values.serviceInsight.port }}
- -service-insight-tls-secret={{ .Values.serviceInsight.secret }}
- -enable-custom-resources={{ .Values.controller.enableCustomResources }}
- -enable-snippets={{ .Values.controller.enableSnippets }}
- -include-year={{ .Values.controller.includeYear }}
- -disable-ipv6={{ .Values.controller.disableIPV6 }}
{{- if .Values.controller.enableCustomResources }}
- -enable-tls-passthrough={{ .Values.controller.enableTLSPassthrough }}
{{- if .Values.controller.enableTLSPassthrough }}
- -tls-passthrough-port={{ .Values.controller.tlsPassthroughPort }}
{{- end }}
- -enable-cert-manager={{ .Values.controller.enableCertManager }}
- -enable-oidc={{ .Values.controller.enableOIDC }}
- -enable-external-dns={{ .Values.controller.enableExternalDNS }}
- -default-http-listener-port={{ .Values.controller.defaultHTTPListenerPort}}
- -default-https-listener-port={{ .Values.controller.defaultHTTPSListenerPort}}
{{- if .Values.controller.globalConfiguration.create }}
- -global-configuration=$(POD_NAMESPACE)/{{ include "nginx-ingress.controller.fullname" . }}
{{- end }}
{{- end }}
- -ready-status={{ .Values.controller.readyStatus.enable }}
- -ready-status-port={{ .Values.controller.readyStatus.port }}
- -enable-latency-metrics={{ .Values.controller.enableLatencyMetrics }}
- -ssl-dynamic-reload={{ .Values.controller.enableSSLDynamicReload }}
{{- end -}}
