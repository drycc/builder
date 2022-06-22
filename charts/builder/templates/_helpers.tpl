{{- define "builder.envs" }}
env:
- name: "HEALTH_SERVER_PORT"
  value: "8092"
- name: "EXTERNAL_PORT"
  value: "2223"
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
- name: "DRYCC_MINIO_LOOKUP"
  valueFrom:
    secretKeyRef:
      name: minio-creds
      key: lookup
- name: "DRYCC_MINIO_BUCKET"
  valueFrom:
    secretKeyRef:
      name: minio-creds
      key: builder-bucket
- name: "DRYCC_MINIO_ENDPOINT"
  valueFrom:
    secretKeyRef:
      name: minio-creds
      key: endpoint
- name: "DRYCC_MINIO_ACCESSKEY"
  valueFrom:
    secretKeyRef:
      name: minio-creds
      key: accesskey
- name: "DRYCC_MINIO_SECRETKEY"
  valueFrom:
    secretKeyRef:
      name: minio-creds
      key: secretkey
- name: "DRYCC_REGISTRY_LOCATION"
  value: "{{ .Values.global.registryLocation }}"
- name: "DRYCC_REGISTRY_HOST"
  valueFrom:
    secretKeyRef:
      name: registry-secret
      key: host
{{- if eq .Values.global.registryLocation "on-cluster" }}
# NOTE(bacongobbler): use drycc/registry_proxy to work around Docker --insecure-registry requirements
- name: "DRYCC_REGISTRY_PROXY_HOST"
  value: {{ print "127.0.0.1"  ":" .Values.global.registryProxyPort }}
{{- else }}
- name: "DRYCC_REGISTRY_ORGANIZATION"
  valueFrom:
    secretKeyRef:
      name: registry-secret
      key: organization
{{- end }}
- name: "DRYCC_REGISTRY_USERNAME"
  valueFrom:
    secretKeyRef:
      name: registry-secret
      key: username
- name: "DRYCC_REGISTRY_PASSWORD"
  valueFrom:
    secretKeyRef:
      name: registry-secret
      key: password
{{- if (.Values.builderPodNodeSelector) }}
- name: BUILDER_POD_NODE_SELECTOR
  value: {{.Values.builderPodNodeSelector}}
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
