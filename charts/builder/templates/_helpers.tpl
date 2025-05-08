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
{{- if (.Values.storageEndpoint) }}
- name: "DRYCC_STORAGE_LOOKUP"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: storage-lookup
- name: "DRYCC_STORAGE_BUCKET"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: storage-bucket
- name: "DRYCC_STORAGE_ENDPOINT"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: storage-endpoint
- name: "DRYCC_STORAGE_ACCESSKEY"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: storage-accesskey
- name: "DRYCC_STORAGE_SECRETKEY"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: storage-secretkey
{{- else if .Values.storage.enabled  }}
- name: "DRYCC_STORAGE_LOOKUP"
  value: "path"
- name: "DRYCC_STORAGE_BUCKET"
  value: "builder"
- name: "DRYCC_STORAGE_ENDPOINT"
  value: {{ printf "http://drycc-storage.%s.svc.%s:9000" .Release.Namespace .Values.global.clusterDomain }}
- name: "DRYCC_STORAGE_ACCESSKEY"
  valueFrom:
    secretKeyRef:
      name: storage-creds
      key: accesskey
- name: "DRYCC_STORAGE_SECRETKEY"
  valueFrom:
    secretKeyRef:
      name: storage-creds
      key: secretkey
{{- end }}
- name: "DRYCC_REGISTRY_LOCATION"
  value: {{ ternary "on-cluster" "off-cluster" .Values.registry.enabled }}
{{- if (.Values.registryHost) }}
- name: "DRYCC_REGISTRY_HOST"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: registry-host
- name: "DRYCC_REGISTRY_USERNAME"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: registry-username
- name: "DRYCC_REGISTRY_PASSWORD"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: registry-password
- name: "DRYCC_REGISTRY_ORGANIZATION"
  valueFrom:
    secretKeyRef:
      name: builder-secret
      key: registry-organization
{{- else if .Values.registry.enabled  }}
- name: "DRYCC_REGISTRY_HOST"
  value: {{ printf "drycc-registry.%s.svc.%s:5000" .Release.Namespace .Values.global.clusterDomain }}
- name: "DRYCC_REGISTRY_PROXY_HOST"
  value: {{ print "127.0.0.1"  ":" .Values.registry.proxyPort }}
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
{{- end }}

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
