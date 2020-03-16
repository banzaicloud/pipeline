CREATE TABLE "vsphere_pke_clusters" (
  "id" serial,
  "cluster_id" integer,
  "spec" json,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_vsphere_pke_cluster_id ON "vsphere_pke_clusters"(cluster_id);

CREATE TABLE "vsphere_pke_node_pools" (
  "id" serial,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deleted_at" timestamp with time zone,
  "autoscaling" boolean,"cluster_id" integer,
  "created_by" integer,"size" integer,
  "max_size" integer,
  "min_size" integer,
  "vcpu" integer,
  "ram_mb" integer,
  "name" text,
  "roles" json,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_vsphere_pke_node_pools_deleted_at ON "vsphere_pke_node_pools"(deleted_at);

CREATE UNIQUE INDEX idx_vsphere_pke_np_cluster_id_name ON "vsphere_pke_node_pools"(cluster_id, "name");
