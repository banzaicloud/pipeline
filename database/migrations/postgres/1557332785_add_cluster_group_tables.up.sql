CREATE TABLE "clustergroups" (
  "id" serial,
  "uid" text,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deleted_at" timestamp with time zone,
  "created_by" integer,
  "name" text,
  "organization_id" integer,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_clustergroups_deleted_at ON "clustergroups"(deleted_at);

CREATE UNIQUE INDEX idx_uid ON "clustergroups"("uid");

CREATE UNIQUE INDEX idx_unique_id ON "clustergroups"(deleted_at, "name", organization_id);

CREATE TABLE "clustergroup_features" (
  "id" serial,
  "name" text,
  "cluster_group_id" integer,
  "enabled" boolean,
  "properties" json,
  "reconcile_state" text,
  "last_reconcile_error" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "clustergroup_members" (
  "id" serial,
  "cluster_group_id" integer,
  "cluster_id" integer,
  PRIMARY KEY ("id")
);

CREATE TABLE "clustergroup_deployments" (
  "id" serial,
  "cluster_group_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deployment_name" text,
  "deployment_version" text,
  "deployment_package" bytea,
  "deployment_release_name" text,
  "description" text,
  "chart_name" text,
  "namespace" text,
  "organization_name" text,
  "values" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_unique_cid_rname ON "clustergroup_deployments"(cluster_group_id, deployment_release_name);


CREATE TABLE "clustergroup_deployment_target_clusters" (
  "id" serial,
  "cluster_group_deployment_id" integer,
  "cluster_id" integer,
  "cluster_name" text,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "values" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_unique_dep_cl ON "clustergroup_deployment_target_clusters"(cluster_group_deployment_id, cluster_id);
