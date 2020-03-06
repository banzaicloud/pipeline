CREATE TABLE "vsphere_pke_clusters" (
  "id" serial,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deleted_at" timestamp with time zone,
  "cluster_id" integer,
  "provider_data" json,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_vsphere_pke_clusters_deleted_at ON "vsphere_pke_clusters"(deleted_at);

CREATE UNIQUE INDEX idx_vsphere_pke_cluster_id ON "vsphere_pke_clusters"(cluster_id);
