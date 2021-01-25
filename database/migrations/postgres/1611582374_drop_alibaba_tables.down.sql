CREATE TABLE "alibaba_buckets" (
  "id" serial,
  "org_id" integer NOT NULL,
  "name" text,
  "region" text,
  "secret_ref" text,
  "status" text,
  "status_msg" text,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_alibaba_buckets_org_id ON "alibaba_buckets"(org_id);

CREATE UNIQUE INDEX idx_alibaba_bucket_name ON "alibaba_buckets"("name");

CREATE TABLE "alibaba_acsk_node_pools" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "name" text,
  "instance_type" text,
  "system_disk_category" text,
  "system_disk_size" integer,
  "image" text,
  "count" integer,
  "min_count" integer,
  "max_count" integer,
  "asg_id" text,
  "scaling_config_id" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_ack_node_pools_cluster_id_name ON "alibaba_acsk_node_pools"(cluster_id, "name");

CREATE TABLE "alibaba_acsk_clusters" (
  "id" serial,
  "provider_cluster_id" text,
  "region_id" text,
  "zone_id" text,
  "master_instance_type" text,
  "master_system_disk_category" text,
  "master_system_disk_size" integer,
  "snat_entry" boolean,
  "ssh_flags" boolean,
  "kubernetes_version" text,
  "v_switch_id" text,
  PRIMARY KEY ("id")
);