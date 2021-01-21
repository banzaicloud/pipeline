CREATE TABLE "oracle_buckets" (
  "id" serial,
  "org_id" integer NOT NULL,
  "compartment_id" text,
  "name" text,
  "location" text,
  "secret_ref" text,
  "status" text,
  "status_msg" text,
  "namespace" text,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_oracle_buckets_org_id ON "oracle_buckets"(org_id);

CREATE UNIQUE INDEX idx_bucket_name_location_compartment ON "oracle_buckets"(
  compartment_id, "name", "location"
);

CREATE TABLE "oracle_oke_clusters" (
  "id" serial,
  "name" text,
  "version" text,
  "vcn_id" text,
  "lb_subnet_id1" text,
  "lb_subnet_id2" text,
  "ocid" text,
  "cluster_model_id" integer,
  "created_by" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_clusters_name ON "oracle_oke_clusters"("name");

CREATE TABLE "oracle_oke_node_pools" (
  "id" serial,
  "name" text,
  "image" text DEFAULT 'Oracle-Linux-7.4',
  "shape" text DEFAULT 'VM.Standard1.1',
  "version" text DEFAULT 'v1.10.3',
  "quantity_per_subnet" integer DEFAULT 1,
  "ocid" text,
  "cluster_id" integer,
  "created_by" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_node_pools_cluster_id_name ON "oracle_oke_node_pools"("name", cluster_id);

CREATE TABLE "oracle_oke_node_pool_subnets" (
  "id" serial,
  "subnet_id" text,
  "node_pool_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_node_pool_subnets_id_subnet_id ON "oracle_oke_node_pool_subnets"(subnet_id, node_pool_id);
