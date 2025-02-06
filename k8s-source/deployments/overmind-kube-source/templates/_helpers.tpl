{{/*
Expand the name of the chart.
*/}}
{{- define "overmind-kube-source.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "overmind-kube-source.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "overmind-kube-source.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "overmind-kube-source.labels" -}}
helm.sh/chart: {{ include "overmind-kube-source.chart" . }}
{{ include "overmind-kube-source.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "overmind-kube-source.selectorLabels" -}}
app.kubernetes.io/name: {{ include "overmind-kube-source.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "overmind-kube-source.serviceAccountName" -}}
{{- default (include "overmind-kube-source.fullname" .) .Values.serviceAccount.name }}
{{- end }}

{{/*
Create the name of the cluster role to use
*/}}
{{- define "overmind-kube-source.clusterRoleName" -}}
{{- default (include "overmind-kube-source.fullname" .) }}
{{- end }}

{{/*
Create the name of the cluster role binidng to use
*/}}
{{- define "overmind-kube-source.clusterRoleBindingName" -}}
{{- default (include "overmind-kube-source.fullname" .) }}
{{- end }}

{{/*
Validate API Key configuration
*/}}
{{- define "overmind-kube-source.validateAPIKey" -}}
{{- if and .Values.source.apiKey.existingSecretName (not .Values.source.apiKey.value) }}
  {{- $secret := lookup "v1" "Secret" .Release.Namespace .Values.source.apiKey.existingSecretName }}
  {{- if not $secret }}
    {{- fail (printf "Secret %q not found in namespace %q" .Values.source.apiKey.existingSecretName .Release.Namespace) }}
  {{- end }}
  {{- if not (get $secret.data "API_KEY") }}
    {{- fail (printf "Secret %q does not contain required key 'API_KEY'" .Values.source.apiKey.existingSecretName) }}
  {{- end }}
{{- else if not .Values.source.apiKey.value }}
  {{- fail "Either source.apiKey.value or source.apiKey.existingSecretName must be set" }}
{{- end }}
{{- end }}
