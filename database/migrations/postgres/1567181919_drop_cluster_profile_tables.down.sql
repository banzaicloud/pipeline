CREATE TABLE "amazon_eks_profiles"
(
    "name"        text,
    "created_at"  timestamp with time zone,
    "updated_at"  timestamp with time zone,
    "region"      text             DEFAULT 'us-west-2',
    "version"     text             DEFAULT '1.10',
    "ttl_minutes" integer NOT NULL DEFAULT 0,
    PRIMARY KEY ("name")
);

CREATE TABLE "amazon_eks_profile_node_pools"
(
    "id"            serial,
    "instance_type" text    DEFAULT 'm4.xlarge',
    "name"          text,
    "node_name"     text,
    "spot_price"    text,
    "autoscaling"   boolean DEFAULT false,
    "min_count"     integer DEFAULT 1,
    "max_count"     integer DEFAULT 2,
    "count"         integer DEFAULT 1,
    "image"         text    DEFAULT 'ami-0a54c984b9f908c81',
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_amazon_name_node_name ON "amazon_eks_profile_node_pools" ("name", node_name);

CREATE TABLE "amazon_eks_profile_node_pool_labels"
(
    "id"                   serial,
    "name"                 text,
    "value"                text,
    "node_pool_profile_id" integer,
    "created_at"           timestamp with time zone,
    "updated_at"           timestamp with time zone,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_eks_profile_node_pool_labels_id_name ON "amazon_eks_profile_node_pool_labels" ("name", node_pool_profile_id);

CREATE TABLE "azure_aks_profiles"
(
    "name"               text,
    "created_at"         timestamp with time zone,
    "updated_at"         timestamp with time zone,
    "location"           text             DEFAULT 'eastus',
    "kubernetes_version" text             DEFAULT '1.9.2',
    "ttl_minutes"        integer NOT NULL DEFAULT 0,
    PRIMARY KEY ("name")
);

CREATE TABLE "azure_aks_profile_node_pools"
(
    "id"                 serial,
    "autoscaling"        boolean DEFAULT false,
    "min_count"          integer DEFAULT 1,
    "max_count"          integer DEFAULT 2,
    "count"              integer DEFAULT 1,
    "node_instance_type" text    DEFAULT 'Standard_D4_v2',
    "name"               text,
    "node_name"          text,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_aks_profile_node_pools_name_node_name ON "azure_aks_profile_node_pools" ("name", node_name);

CREATE TABLE "azure_aks_profile_node_pool_labels"
(
    "id"                   serial,
    "name"                 text,
    "value"                text,
    "node_pool_profile_id" integer,
    "created_at"           timestamp with time zone,
    "updated_at"           timestamp with time zone,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_aks_profile_node_pool_labels_name_id ON "azure_aks_profile_node_pool_labels" ("name", node_pool_profile_id);

CREATE TABLE "google_gke_profiles"
(
    "name"           text,
    "created_at"     timestamp with time zone,
    "updated_at"     timestamp with time zone,
    "location"       text             DEFAULT 'us-central1-a',
    "node_version"   text             DEFAULT '1.10',
    "master_version" text             DEFAULT '1.10',
    "ttl_minutes"    integer NOT NULL DEFAULT 0,
    PRIMARY KEY ("name")
);

CREATE TABLE "google_gke_profile_node_pools"
(
    "id"                 serial,
    "autoscaling"        boolean DEFAULT false,
    "min_count"          integer DEFAULT 1,
    "max_count"          integer DEFAULT 2,
    "count"              integer DEFAULT 1,
    "node_instance_type" text    DEFAULT 'n1-standard-1',
    "name"               text,
    "node_name"          text,
    "preemptible"        boolean DEFAULT false,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_gke_profile_node_pools_name_node_name ON "google_gke_profile_node_pools" ("name", node_name);

CREATE TABLE "google_gke_profile_node_pool_labels"
(
    "id"                   serial,
    "name"                 text,
    "value"                text,
    "node_pool_profile_id" integer,
    "created_at"           timestamp with time zone,
    "updated_at"           timestamp with time zone,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_gke_name_profile_node_pool_id ON "google_gke_profile_node_pool_labels" ("name", node_pool_profile_id);

ALTER TABLE
    "amazon_eks_profile_node_pools"
    ADD
        CONSTRAINT amazon_eks_profile_node_pools_name_amazon_eks_profiles_name_foreign FOREIGN KEY ("name") REFERENCES amazon_eks_profiles (name) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
    "amazon_eks_profile_node_pool_labels"
    ADD
        CONSTRAINT amazon_eks_profile_node_pool_labels_node_pool_profile_id_amazon_eks_profile_node_pools_id_foreign FOREIGN KEY (node_pool_profile_id) REFERENCES amazon_eks_profile_node_pools (id) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
    "azure_aks_profile_node_pools"
    ADD
        CONSTRAINT azure_aks_profile_node_pools_name_azure_aks_profiles_name_foreign FOREIGN KEY ("name") REFERENCES azure_aks_profiles (name) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
    "azure_aks_profile_node_pool_labels"
    ADD
        CONSTRAINT azure_aks_profile_node_pool_labels_node_pool_profile_id_azure_aks_profile_node_pools_id_foreign FOREIGN KEY (node_pool_profile_id) REFERENCES azure_aks_profile_node_pools (id) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
    "google_gke_profile_node_pools"
    ADD
        CONSTRAINT google_gke_profile_node_pools_name_google_gke_profiles_name_foreign FOREIGN KEY ("name") REFERENCES google_gke_profiles (name) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
    "google_gke_profile_node_pool_labels"
    ADD
        CONSTRAINT google_gke_profile_node_pool_labels_node_pool_profile_id_google_gke_profile_node_pools_id_foreign FOREIGN KEY (node_pool_profile_id) REFERENCES google_gke_profile_node_pools (id) ON DELETE RESTRICT ON UPDATE RESTRICT;

CREATE TABLE "oracle_oke_profiles"
(
    "id"          serial,
    "name"        text,
    "location"    text             DEFAULT 'eu-frankfurt-1',
    "version"     text             DEFAULT 'v1.10.3',
    "ttl_minutes" integer NOT NULL DEFAULT 0,
    "created_at"  timestamp with time zone,
    "updated_at"  timestamp with time zone,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_profiles_name ON "oracle_oke_profiles" ("name");

CREATE TABLE "oracle_oke_profile_node_pools"
(
    "id"         serial,
    "name"       text,
    "count"      integer DEFAULT '1',
    "image"      text    DEFAULT 'Oracle-Linux-7.4',
    "shape"      text    DEFAULT 'VM.Standard1.1',
    "version"    text    DEFAULT 'v1.10.3',
    "profile_id" integer,
    "created_at" timestamp with time zone,
    "updated_at" timestamp with time zone,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_profile_node_pools_name_profile_id ON "oracle_oke_profile_node_pools" ("name", profile_id);

CREATE TABLE "oracle_oke_profile_node_pool_labels"
(
    "id"                   serial,
    "name"                 text,
    "value"                text,
    "profile_node_pool_id" integer,
    "created_at"           timestamp with time zone,
    "updated_at"           timestamp with time zone,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_profile_node_pool_labels_name_profile_id ON "oracle_oke_profile_node_pool_labels" ("name", profile_node_pool_id);
