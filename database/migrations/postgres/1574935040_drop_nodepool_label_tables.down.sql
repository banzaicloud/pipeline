CREATE TABLE "amazon_node_pool_labels" (
                                           "id" serial,
                                           "name" text,
                                           "value" text,
                                           "node_pool_id" integer,
                                           "created_at" timestamp with time zone,
                                           "updated_at" timestamp with time zone,
                                           PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_amazon_node_pool_labels_id_name ON "amazon_node_pool_labels"("name", node_pool_id);

CREATE TABLE "google_gke_node_pool_labels" (
                                               "id" serial,
                                               "name" text,
                                               "value" text,
                                               "node_pool_id" integer,
                                               "created_at" timestamp with time zone,
                                               "updated_at" timestamp with time zone,
                                               PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_gke_node_pool_labels_id_name ON "google_gke_node_pool_labels"("name", node_pool_id);
