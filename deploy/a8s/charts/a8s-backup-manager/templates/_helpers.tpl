{{- define "a8s-backup-manager.fullname" -}}
a8s-backup-manager
{{- end -}}

{{- define "a8s-backup-manager.namespace" -}}
{{ .Values.namespace | default .Release.Namespace }}
{{- end -}}

{{- define "a8s-backup-manager.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{ default (include "a8s-backup-manager.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{ default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end -}}

{{- define "a8s-backup-manager.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/name: {{ include "a8s-backup-manager.fullname" . }}
app.kubernetes.io/part-of: a8s-backup
{{- end -}}

{{/*
Prefix for user-facing value paths in error/NOTES messages.
Empty when this chart is installed standalone (paths are bare, e.g.
"backupStorageConfig.secret.accessKeyId"). The parent umbrella chart sets
`umbrellaValuePathPrefix` to its dependency alias (e.g. "a8sBackupOperator.")
so messages name the exact path the user edits in the umbrella values file.
*/}}
{{- define "a8s-backup-manager.valuePath" -}}
{{- .Values.umbrellaValuePathPrefix | default "" -}}
{{- end -}}
