{{- /*
Chart-wide helpers.

  flock-addon.flockAllianceImage  - "repository:tag" string used as the value
                                    of the FLOCK_ALLIANCE_IMAGE customized
                                    variable on both AddOnDeploymentConfig
                                    objects.
  flock-addon.templateName        - name of the CPU AddOnTemplate. Also used
                                    as the default ClusterManagementAddOn
                                    supportedConfigs template name.
  flock-addon.gpuTemplateName     - name of the GPU AddOnTemplate, i.e.
                                    "<addon>-gpu".
*/ -}}

{{- define "flock-addon.flockAllianceImage" -}}
{{- printf "%s:%s" .Values.image.repository .Values.image.tag -}}
{{- end -}}

{{- define "flock-addon.templateName" -}}
{{- .Values.addon.name -}}
{{- end -}}

{{- define "flock-addon.gpuTemplateName" -}}
{{- printf "%s-gpu" .Values.addon.name -}}
{{- end -}}
