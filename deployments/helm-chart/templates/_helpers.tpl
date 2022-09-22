{{/* vim: set filetype=mustache: */}}

{{/*
Expand the name of the chart.
*/}}
{{- define "nginx-ingress.name" -}}
{{- printf "%s-%s" .Release.Name .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create labels
*/}}
{{- define "nginx-ingress.labels" -}}
app.kubernetes.io/name: {{ include "nginx-ingress.name" . }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
Expand the name of the configmap.
*/}}
{{- define "nginx-ingress.configName" -}}
{{- if .Values.controller.customConfigMap -}}
{{ .Values.controller.customConfigMap }}
{{- else -}}
{{- default (include "nginx-ingress.name" .) .Values.controller.config.name -}}
{{- end -}}
{{- end -}}

{{/*
Expand leader election lock name.
*/}}
{{- define "nginx-ingress.leaderElectionName" -}}
{{- if .Values.controller.reportIngressStatus.leaderElectionLockName -}}
{{ .Values.controller.reportIngressStatus.leaderElectionLockName }}
{{- else -}}
{{- printf "%s-%s" (include "nginx-ingress.name" .) "leader-election" -}}
{{- end -}}
{{- end -}}

{{/*
Expand service account name.
*/}}
{{- define "nginx-ingress.serviceAccountName" -}}
{{- default (include "nginx-ingress.name" .) .Values.controller.serviceAccount.name -}}
{{- end -}}

{{/*
Expand service name.
*/}}
{{- define "nginx-ingress.serviceName" -}}
{{- default (include "nginx-ingress.name" .) .Values.controller.service.name }}
{{- end -}}

{{/*
Expand serviceMonitor name.
*/}}
{{- define "nginx-ingress.serviceMonitorName" -}}
{{- default (include "nginx-ingress.name" .) .Values.controller.serviceMonitor.name }}
{{- end -}}

{{/*
Expand default TLS name.
*/}}
{{- define "nginx-ingress.defaultTLSName" -}}
{{- printf "%s-%s" (include "nginx-ingress.name" .) "default-server-tls" -}}
{{- end -}}

{{/*
Expand wildcard TLS name.
*/}}
{{- define "nginx-ingress.wildcardTLSName" -}}
{{- printf "%s-%s" (include "nginx-ingress.name" .) "wildcard-tls" -}}
{{- end -}}

{{/*
Expand app name.
*/}}
{{- define "nginx-ingress.appName" -}}
{{- default (include "nginx-ingress.name" .) .Values.controller.name -}}
{{- end -}}

{{/*
Expand image name.
*/}}
{{- define "nginx-ingress.image" -}}
{{- if .Values.controller.image.digest -}}
{{- printf "%s@%s" .Values.controller.image.repository .Values.controller.image.digest -}}
{{- else -}}
{{- printf "%s:%s" .Values.controller.image.repository .Values.controller.image.tag -}}
{{- end -}}
{{- end -}}
