{{- define "flock-addon.flockAllianceImage" -}}
{{ printf "%s:%s" .Values.image.repository .Values.image.tag }}
{{- end -}}

{{- define "flock-addon.templateName" -}}
{{ .Values.addon.name }}
{{- end -}}

{{- define "flock-addon.gpuTemplateName" -}}
{{ printf "%s-gpu" .Values.addon.name }}
{{- end -}}
