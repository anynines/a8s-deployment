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
