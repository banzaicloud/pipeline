CREATE TABLE "public"."cluster_tags" (
    "id" serial,
    "cluster_id" integer,
    "key" text,
    "value" text,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_cluster_tags_unique_id ON "cluster_tags"(cluster_id, "key");

ALTER TABLE "cluster_tags" ADD CONSTRAINT cluster_tags_cluster_id_clusters_id_foreign FOREIGN KEY (cluster_id) REFERENCES "public"."clusters"("id") ON DELETE RESTRICT ON UPDATE RESTRICT;
