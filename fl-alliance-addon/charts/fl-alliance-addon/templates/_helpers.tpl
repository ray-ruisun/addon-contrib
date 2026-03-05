{{- define "fl-alliance-addon.allianceImage" -}}
{{ printf "%s:%s" .Values.image.repository .Values.image.tag }}
{{- end -}}

{{- define "fl-alliance-addon.flockitImage" -}}
{{ printf "%s:%s" .Values.flockitImage.repository .Values.flockitImage.tag }}
{{- end -}}
