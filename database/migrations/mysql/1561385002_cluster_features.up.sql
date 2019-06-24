CREATE TABLE IF NOT EXISTS "clusterfeature"
(
    "id"         serial,
    "name"       text,
    "cluster_id" integer,
    "spec"       text,
    "status"     text,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    "deleted_at" timestamp with time zone,
    "created_by" integer,
    PRIMARY KEY ("id")
);

