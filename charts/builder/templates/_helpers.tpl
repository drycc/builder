{{/* Generate builder affinity */}}
{{- define "builder.affinity" -}}
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - drycc-builder
        topologyKey: topology.kubernetes.io/zone
    - weight: 90
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app
            operator: In
            values:
            - drycc-builder
        topologyKey: kubernetes.io/hostname
{{- end }}