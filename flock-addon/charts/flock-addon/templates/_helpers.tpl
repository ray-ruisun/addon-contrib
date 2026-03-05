{{- define "flock-addon.flockAllianceImage" -}}
{{ printf "%s:%s" .Values.image.repository .Values.image.tag }}
{{- end -}}

{{- define "flock-addon.flockitImage" -}}
{{ printf "%s:%s" .Values.flockitImage.repository .Values.flockitImage.tag }}
{{- end -}}
