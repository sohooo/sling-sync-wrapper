{{/*
  This file contains helper templates for the Helm chart.
*/}}

{{- define "sling-sync-wrapper.fullname" -}}
{{ .Release.Name }}-{{ .Chart.Name }}
{{- end -}}

{{- define "sling-sync-wrapper.name" -}}
{{ .Chart.Name }}
{{- end -}}
