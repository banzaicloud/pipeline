CREATE TABLE "vsphere_pke_clusters" (
  "id" serial,
  "cluster_id" integer,
  "spec" json,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_vsphere_pke_cluster_id ON "vsphere_pke_clusters"(cluster_id);
