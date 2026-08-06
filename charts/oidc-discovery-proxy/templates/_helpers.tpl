{{- define "oidc-discovery-proxy.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- print .Values.image.repository ":" $tag -}}
{{- end -}}
