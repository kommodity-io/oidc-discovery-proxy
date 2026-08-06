{{- define "oidc-discovery-proxy.version" -}}
{{- .Chart.AppVersion | default .Chart.Version -}}
{{- end -}}

{{- define "oidc-discovery-proxy.image" -}}
{{- $tag := default .Chart.AppVersion .Values.image.tag -}}
{{- print .Values.image.repository ":" $tag -}}
{{- end -}}
