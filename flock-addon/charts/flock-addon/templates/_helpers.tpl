{{- define "flock-addon.flockAllianceImage" -}}
{{ printf "%s:%s" .Values.image.repository .Values.image.tag }}
{{- end -}}
