{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "pipeline.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create a default fully qualified app name.
We truncate at 63 chars because some Kubernetes name fields are limited to this (by the DNS naming spec).
If release name contains chart name it will be used as a full name.
*/}}
{{- define "pipeline.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "pipeline.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Call nested templates.
Source: https://stackoverflow.com/a/52024583/3027614
*/}}
{{- define "call-nested" }}
{{- $dot := index . 0 }}
{{- $subchart := index . 1 }}
{{- $template := index . 2 }}
{{- $subchartValues := index $dot.Values $subchart -}}
{{- $globalValues := dict "global" (index $dot.Values "global") -}}
{{- $values := merge $globalValues $subchartValues -}}
{{- include $template (dict "Chart" (dict "Name" $subchart) "Values" $values "Release" $dot.Release "Capabilities" $dot.Capabilities) }}
{{- end -}}

{{- define "pipeline.database.name" -}}
{{- if .Values.database.name -}}
{{- .Values.database.name -}}
{{- else if .Values.cloudsql.enabled -}}
{{- required "Please specify database name" .Values.database.name -}}
{{- else if and .Values.mysql.enabled ( eq (include "pipeline.database.driver" .) "mysql") -}}
{{- .Values.mysql.mysqlDatabase -}}
{{- else if and .Values.postgres.enabled (eq (include "pipeline.database.driver" .) "postgres") -}}
{{- .Values.postgres.postgresqlDatabase -}}
{{- else -}}
{{- required "Please specify database name" .Values.database.name -}}
{{- end -}}
{{- end -}}

{{- define "pipeline.database.username" -}}
{{- if .Values.database.username -}}
{{- .Values.database.username -}}
{{- else if .Values.cloudsql.enabled -}}
{{- required "Please specify database name" .Values.database.name -}}
{{- else if and .Values.mysql.enabled ( eq (include "pipeline.database.driver" .) "mysql") }}
{{- .Values.mysql.mysqlUser -}}
{{- else if and .Values.postgres.enabled (eq (include "pipeline.database.driver" .) "postgres") -}}
{{- .Values.postgres.postgresqlUsername -}}
{{- else -}}
{{- required "Please specify database user" .Values.database.username -}}
{{- end -}}
{{- end -}}

{{- define "pipeline.database.password" -}}
{{- if .Values.database.password -}}
{{- .Values.database.password -}}
{{- else if .Values.cloudsql.enabled -}}
{{- fail "Please specify database password or existing secret" -}}
{{- else if and .Values.mysql.enabled ( eq .Values.database.driver "mysql") -}}
{{- .Values.mysql.mysqlPassword -}}
{{- else if and .Values.postgres.enabled (eq (include "pipeline.database.driver" .) "postgres") -}}
{{- .Values.postgres.postgresqlPassword -}}
{{- else -}}
{{- required "Please specify database password" .Values.database.password -}}
{{- end -}}
{{- end -}}

{{- define "pipeline.database.secretName" -}}
{{- if .Values.database.existingSecret -}}
{{- .Values.database.existingSecret -}}
{{- else if .Values.database.password -}}
{{- printf "%s-database" (include "pipeline.fullname" .) -}}
{{- else if .Values.cloudsql.enabled -}}
{{- fail "Please specify database password or existing secret" -}}
{{- else if and .Values.mysql.enabled ( eq (include "pipeline.database.driver" .) "mysql") -}}
{{- include "call-nested" (list . "mysql" "mysql.secretName") -}}
{{- else if and .Values.postgres.enabled (eq (include "pipeline.database.driver" .) "postgres") -}}
{{- include "call-nested" (list . "postgres" "postgresql.secretName") -}}
{{- else -}}
{{- fail "Please specify database password or existing secret" -}}
{{- end -}}
{{- end -}}

{{- define "pipeline.database.secretKey" -}}
{{- if or .Values.database.existingSecret .Values.database.password -}}
{{- print "password" -}}
{{- else if .Values.cloudsql.enabled -}}
{{- fail "Please specify database password or existing secret" -}}
{{- else if and .Values.mysql.enabled ( eq (include "pipeline.database.driver" .) "mysql") }}
{{- print "mysql-password" -}}
{{- else if and .Values.postgres.enabled (eq (include "pipeline.database.driver" .) "postgres") -}}
{{- print "postgresql-password" -}}
{{- else -}}
{{- fail "Please specify database password or existing secret" -}}
{{- end -}}
{{- end -}}

{{- define "pipeline.database.host" -}}
{{- if .Values.database.host -}}
{{- .Values.database.host -}}
{{- else if .Values.cloudsql.enabled -}}
{{- printf "%s.%s.svc.cluster.local" (include "call-nested" (list . "cloudsql" "gcloud-sqlproxy.fullname")) .Release.Namespace -}}
{{- else if and .Values.mysql.enabled ( eq (include "pipeline.database.driver" .) "mysql") -}}
{{- printf "%s.%s.svc.cluster.local" (include "call-nested" (list . "mysql" "mysql.fullname")) .Release.Namespace -}}
{{- else if and .Values.postgres.enabled (eq (include "pipeline.database.driver" .) "postgres") -}}
{{- printf "%s.%s.svc.cluster.local" (include "call-nested" (list . "postgres" "postgresql.fullname")) .Release.Namespace -}}
{{- else -}}
{{- required "Please specify database host" .Values.database.host -}}
{{- end -}}
{{- end -}}

{{- define "pipeline.database.port" -}}
{{- if .Values.database.port -}}
{{- .Values.database.port -}}
{{- else if .Values.cloudsql.enabled -}}
{{- (index .Values.cloudsql.cloudsql.instances 0).port -}}
{{- else if and .Values.mysql.enabled ( eq (include "pipeline.database.driver" .) "mysql") }}
{{- .Values.mysql.service.port -}}
{{- else if and .Values.postgres.enabled (eq (include "pipeline.database.driver" .) "postgres") -}}
{{- include "call-nested" (list . "postgres" "postgresql.port") -}}
{{- else -}}
{{- required "Please specify database port" .Values.database.port -}}
{{- end -}}
{{- end -}}

{{- define "pipeline.database.driver" -}}
{{- if .Values.database.driver -}}
{{- .Values.database.driver -}}
{{- else if .Values.cloudsql.enabled -}}
{{- required "Please specify database driver" .Values.database.driver -}}
{{- else if and .Values.mysql.enabled .Values.postgres.enabled -}}
{{- fail "Please enable only one database engine or specify database driver" -}}
{{- else if .Values.mysql.enabled -}}
{{- print "mysql" -}}
{{- else if .Values.postgres.enabled -}}
{{- print "postgres" -}}
{{- else -}}
{{- required "Please specify database driver" .Values.database.driver -}}
{{- end -}}
{{- end -}}

{{- define "pipeline.database.tls" -}}
{{- if .Values.database.tls -}}
{{- .Values.database.tls -}}
{{- else if ( eq (include "pipeline.database.driver" .) "mysql") -}}
{{- print "false" -}}
{{- else if ( eq (include "pipeline.database.driver" .) "postgres") -}}
{{- print "disable" -}}
{{- else -}}
{{- fail "Please specify database tls" -}}
{{- end -}}
{{- end -}}
