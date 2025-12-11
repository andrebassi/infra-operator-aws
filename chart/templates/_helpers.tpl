{{/*
Expand the name of the chart.
*/}}
{{- define "infra-operator.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "infra-operator.fullname" -}}
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
{{- define "infra-operator.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "infra-operator.labels" -}}
helm.sh/chart: {{ include "infra-operator.chart" . }}
{{ include "infra-operator.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "infra-operator.selectorLabels" -}}
app.kubernetes.io/name: {{ include "infra-operator.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "infra-operator.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "infra-operator.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Create the name of the metrics service
*/}}
{{- define "infra-operator.metricsServiceName" -}}
{{- printf "%s-metrics" (include "infra-operator.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the name of the webhook service
*/}}
{{- define "infra-operator.webhookServiceName" -}}
{{- printf "%s-webhook" (include "infra-operator.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the name of the webhook certificate
*/}}
{{- define "infra-operator.webhookCertName" -}}
{{- printf "%s-webhook-cert" (include "infra-operator.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create the name of the webhook issuer
*/}}
{{- define "infra-operator.webhookIssuerName" -}}
{{- printf "%s-webhook-issuer" (include "infra-operator.fullname" .) | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Return the appropriate apiVersion for PodDisruptionBudget
*/}}
{{- define "infra-operator.pdb.apiVersion" -}}
{{- if semverCompare ">=1.21-0" .Capabilities.KubeVersion.GitVersion -}}
policy/v1
{{- else -}}
policy/v1beta1
{{- end -}}
{{- end -}}

{{/*
Return the appropriate apiVersion for NetworkPolicy
*/}}
{{- define "infra-operator.networkPolicy.apiVersion" -}}
{{- if semverCompare ">=1.7-0" .Capabilities.KubeVersion.GitVersion -}}
networking.k8s.io/v1
{{- else -}}
extensions/v1beta1
{{- end -}}
{{- end -}}

{{/*
Return the target Kubernetes version
*/}}
{{- define "infra-operator.kubeVersion" -}}
{{- default .Capabilities.KubeVersion.Version .Values.kubeVersionOverride }}
{{- end }}

{{/*
Return the namespace to use
*/}}
{{- define "infra-operator.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride }}
{{- end }}

{{/*
Compile all warnings into a single message
*/}}
{{- define "infra-operator.validateValues" -}}
{{- $messages := list -}}
{{- if and .Values.webhooks.enabled (not .Values.webhooks.certManager.enabled) -}}
{{- $messages = append $messages "WARNING: Webhooks are enabled but cert-manager integration is disabled. You need to provide certificates manually." -}}
{{- end -}}
{{- if and (gt (int .Values.replicaCount) 1) (not .Values.leaderElection.enabled) -}}
{{- $messages = append $messages "WARNING: Multiple replicas configured but leader election is disabled. This may cause conflicts." -}}
{{- end -}}
{{- if and .Values.aws.irsa.enabled .Values.aws.staticCredentials.enabled -}}
{{- $messages = append $messages "WARNING: Both IRSA and static credentials are enabled. IRSA will take precedence." -}}
{{- end -}}
{{- join "\n" $messages -}}
{{- end -}}
