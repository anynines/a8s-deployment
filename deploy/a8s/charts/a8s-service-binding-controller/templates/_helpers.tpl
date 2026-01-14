{{- define "a8s-service-binding-controller.fullname" -}}
a8s-service-binding-controller
{{- end -}}

{{- define "a8s-service-binding-controller.namespace" -}}
{{ .Values.namespace | default .Release.Namespace }}
{{- end -}}

{{- define "a8s-service-binding-controller.labels" -}}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/name: {{ include "a8s-service-binding-controller.fullname" . }}
app.kubernetes.io/part-of: a8s-service-binding
{{- end -}}
