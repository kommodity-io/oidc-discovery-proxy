{{- define "oidc-discovery-proxy.image" -}}
{{- $tag := default .defaultTag .image.tag -}}
{{- if kindIs "invalid" $tag -}}
  {{- fail "An image tag is required. Please set .Values.image.tag or ensure .Chart.AppVersion is set." -}}
{{- else if not (typeIs "string" $tag) -}}
  {{- fail "Image tags must be strings." -}}
{{- end -}}
{{- if eq $tag "" -}}
  {{- fail "An image tag is required. Please set .Values.image.tag or ensure .Chart.AppVersion is set." -}}
{{- end -}}
{{- print (required "An image repository is required" .image.repository) ":" $tag -}}
{{- end -}}
