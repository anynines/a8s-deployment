{{- define "a8s-postgresql-operator.fullname" -}}
a8s-postgresql-operator
{{- end -}}

{{- define "a8s-postgresql-operator.namespace" -}}
{{ .Values.namespace | default .Release.Namespace }}
{{- end -}}

{{- define "a8s-postgresql-operator.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/name: {{ include "a8s-postgresql-operator.fullname" . }}
app.kubernetes.io/part-of: a8s-postgres
{{- end -}}
