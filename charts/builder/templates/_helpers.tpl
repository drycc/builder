{{- define "builder.envs" -}}
env:
# NOTE(bacongobbler): use drycc/registry_proxy to work around Docker --insecure-registry requirements
- name: "DRYCC_REGISTRY_PROXY_HOST"
  value: "127.0.0.1"
- name: "DRYCC_REGISTRY_PROXY_PORT"
  value: "{{ .Values.global.registryProxyPort }}"
- name: "HEALTH_SERVER_PORT"
  value: "8092"
- name: "EXTERNAL_PORT"
  value: "2223"
- name: BUILDER_STORAGE
  value: "{{ .Values.global.storage }}"
- name: "DRYCC_REGISTRY_LOCATION"
  value: "{{ .Values.global.registryLocation }}"
- name: "TTL_SECONDS_AFTER_FINISHED"
  value: "{{ .Values.global.ttlSecondsAfterFinished }}"
# Set GIT_LOCK_TIMEOUT to number of minutes you want to wait to git push again to the same repository
- name: "GIT_LOCK_TIMEOUT"
  value: "30"
- name: IMAGEBUILDER_IMAGE_PULL_POLICY
  valueFrom:
    configMapKeyRef:
      name: imagebuilder-config
      key: imagePullPolicy
- name: "DRYCC_DEBUG"
  value: "false"
- name: "POD_NAMESPACE"
  valueFrom:
    fieldRef:
    fieldPath: metadata.namespace
- name: DRYCC_BUILDER_KEY
  valueFrom:
    secretKeyRef:
      name: builder-key-auth
      key: builder-key
{{- if (.Values.builder_pod_node_selector) }}
- name: BUILDER_POD_NODE_SELECTOR
  value: {{.Values.builder_pod_node_selector}}
{{- if eq .Values.global.minioLocation "on-cluster" }}
- name: "DRYCC_MINIO_ENDPOINT"
  value: http://${DRYCC_MINIO_SERVICE_HOST}:${DRYCC_MINIO_SERVICE_PORT}
{{- else }}
- name: "DRYCC_MINIO_ENDPOINT"
  value: "{{ .Values.minio.endpoint }}"
{{- end }}
{{- end }}

{{/* Generate builder deployment limits */}}
{{- define "builder.limits" -}}
{{- if or (.Values.limitsCpu) (.Values.limitsMemory) }}
resources:
  limits:
    {{- if (.Values.limitsCpu) }}
    cpu: {{.Values.limitsCpu}}
    {{- end }}
    {{- if (.Values.limitsMemory) }}
    memory: {{.Values.limitsMemory}}
    {{- end }}
{{- end }}
{{- end }}
