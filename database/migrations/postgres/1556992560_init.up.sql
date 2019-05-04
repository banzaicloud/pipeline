CREATE TABLE "clusters" (
  "id" serial,
  "uid" text,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deleted_at" timestamp with time zone,
  "started_at" timestamp with time zone,
  "name" text,
  "location" text,
  "cloud" text,
  "distribution" text,
  "organization_id" integer,
  "secret_id" text,
  "config_secret_id" text,
  "ssh_secret_id" text,
  "status" text,
  "rbac_enabled" boolean,
  "monitoring" boolean,
  "logging" boolean,
  "service_mesh" boolean,
  "security_scan" boolean,
  "status_message" text,
  "created_by" integer,
  "ttl_minutes" integer NOT NULL DEFAULT 0,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_clusters_deleted_at ON "clusters"(deleted_at);

CREATE UNIQUE INDEX idx_clusters_unique_id ON "clusters"(
  deleted_at, "name", organization_id
);

CREATE UNIQUE INDEX idx_clusters_uid ON "clusters"("uid");

CREATE TABLE "scale_options" (
  "id" serial,
  "cluster_id" integer,
  "enabled" boolean,
  "desired_cpu" numeric,
  "desired_mem" numeric,
  "desired_gpu" integer,
  "on_demand_pct" integer,
  "excludes" text,
  "keep_desired_capacity" boolean,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_scale_options_cluster_id ON "scale_options"(cluster_id);

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

CREATE TABLE "amazon_node_pools" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "name" text,
  "node_spot_price" text,
  "autoscaling" boolean,
  "node_min_count" integer,
  "node_max_count" integer,
  "count" integer,
  "node_image" text,
  "node_instance_type" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_amazon_node_pools_cluster_id_name ON "amazon_node_pools"(cluster_id, "name");

CREATE TABLE "amazon_eks_clusters" (
  "id" serial,
  "cluster_id" integer,
  "version" text,
  "vpc_id" varchar(32),
  "vpc_cidr" varchar(18),
  "route_table_id" varchar(32),
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_eks_clusters_cluster_id ON "amazon_eks_clusters"(cluster_id);

CREATE TABLE "amazon_eks_subnets" (
  "id" serial,
  "created_at" timestamp with time zone,
  "cluster_id" integer,
  "subnet_id" varchar(32),
  "cidr" varchar(18),
  PRIMARY KEY ("id")
);

CREATE INDEX idx_eks_subnets_cluster_id ON "amazon_eks_subnets"(cluster_id);

CREATE TABLE "azure_aks_clusters" (
  "id" serial,
  "resource_group" text,
  "kubernetes_version" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "azure_aks_node_pools" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "name" text,
  "autoscaling" boolean,
  "node_min_count" integer,
  "node_max_count" integer,
  "count" integer,
  "node_instance_type" text,
  "v_net_subnet_id" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_aks_node_pools_cluster_id_name ON "azure_aks_node_pools"(cluster_id, "name");

CREATE TABLE "dummy_clusters" (
  "id" serial,
  "kubernetes_version" text,
  "node_count" integer,
  PRIMARY KEY ("id")
);

CREATE TABLE "kubernetes_clusters" (
  "id" serial,
  "metadata_raw" bytea,
  PRIMARY KEY ("id")
);

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

ALTER TABLE
  "amazon_eks_clusters"
ADD
  CONSTRAINT amazon_eks_clusters_cluster_id_clusters_id_foreign FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
  "scale_options"
ADD
  CONSTRAINT scale_options_cluster_id_clusters_id_foreign FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
  "amazon_eks_subnets"
ADD
  CONSTRAINT amazon_eks_subnets_cluster_id_amazon_eks_clusters_id_foreign FOREIGN KEY (cluster_id) REFERENCES amazon_eks_clusters(id) ON DELETE RESTRICT ON UPDATE RESTRICT;

CREATE TABLE "auth_identities" (
  "id" serial,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "provider" text,
  "uid" text,
  "encrypted_password" text,
  "user_id" text,
  "confirmed_at" timestamp with time zone,
  "sign_logs" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "user_organizations" (
  "user_id" integer,
  "organization_id" integer,
  PRIMARY KEY ("user_id", "organization_id")
);

CREATE TABLE "users" (
  "id" serial,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "name" text,
  "email" text,
  "login" text NOT NULL UNIQUE,
  "image" text,
  PRIMARY KEY ("id")
);

ALTER TABLE
  "user_organizations"
ADD
  "role" text DEFAULT 'admin';

CREATE TABLE "organizations" (
  "id" serial,
  "github_id" bigint UNIQUE,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "name" text NOT NULL UNIQUE,
  "provider" text NOT NULL,
  PRIMARY KEY ("id")
);

CREATE TABLE "amazon_eks_profiles" (
  "name" text,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "region" text DEFAULT 'us-west-2',
  "version" text DEFAULT '1.10',
  "ttl_minutes" integer NOT NULL DEFAULT 0,
  PRIMARY KEY ("name")
);

CREATE TABLE "amazon_eks_profile_node_pools" (
  "id" serial,
  "instance_type" text DEFAULT 'm4.xlarge',
  "name" text,
  "node_name" text,
  "spot_price" text,
  "autoscaling" boolean DEFAULT false,
  "min_count" integer DEFAULT 1,
  "max_count" integer DEFAULT 2,
  "count" integer DEFAULT 1,
  "image" text DEFAULT 'ami-0a54c984b9f908c81',
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_amazon_name_node_name ON "amazon_eks_profile_node_pools"("name", node_name);

CREATE TABLE "amazon_eks_profile_node_pool_labels" (
  "id" serial,
  "name" text,
  "value" text,
  "node_pool_profile_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_eks_profile_node_pool_labels_id_name ON "amazon_eks_profile_node_pool_labels"("name", node_pool_profile_id);

CREATE TABLE "azure_aks_profiles" (
  "name" text,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "location" text DEFAULT 'eastus',
  "kubernetes_version" text DEFAULT '1.9.2',
  "ttl_minutes" integer NOT NULL DEFAULT 0,
  PRIMARY KEY ("name")
);

CREATE TABLE "azure_aks_profile_node_pools" (
  "id" serial,
  "autoscaling" boolean DEFAULT false,
  "min_count" integer DEFAULT 1,
  "max_count" integer DEFAULT 2,
  "count" integer DEFAULT 1,
  "node_instance_type" text DEFAULT 'Standard_D4_v2',
  "name" text,
  "node_name" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_aks_profile_node_pools_name_node_name ON "azure_aks_profile_node_pools"("name", node_name);

CREATE TABLE "azure_aks_profile_node_pool_labels" (
  "id" serial,
  "name" text,
  "value" text,
  "node_pool_profile_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_aks_profile_node_pool_labels_name_id ON "azure_aks_profile_node_pool_labels"("name", node_pool_profile_id);

CREATE TABLE "google_gke_profiles" (
  "name" text,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "location" text DEFAULT 'us-central1-a',
  "node_version" text DEFAULT '1.10',
  "master_version" text DEFAULT '1.10',
  "ttl_minutes" integer NOT NULL DEFAULT 0,
  PRIMARY KEY ("name")
);

CREATE TABLE "google_gke_profile_node_pools" (
  "id" serial,
  "autoscaling" boolean DEFAULT false,
  "min_count" integer DEFAULT 1,
  "max_count" integer DEFAULT 2,
  "count" integer DEFAULT 1,
  "node_instance_type" text DEFAULT 'n1-standard-1',
  "name" text,
  "node_name" text,
  "preemptible" boolean DEFAULT false,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_gke_profile_node_pools_name_node_name ON "google_gke_profile_node_pools"("name", node_name);

CREATE TABLE "google_gke_profile_node_pool_labels" (
  "id" serial,
  "name" text,
  "value" text,
  "node_pool_profile_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_gke_name_profile_node_pool_id ON "google_gke_profile_node_pool_labels"("name", node_pool_profile_id);

ALTER TABLE
  "amazon_eks_profile_node_pools"
ADD
  CONSTRAINT amazon_eks_profile_node_pools_name_amazon_eks_profiles_name_foreign FOREIGN KEY ("name") REFERENCES amazon_eks_profiles(name) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
  "amazon_eks_profile_node_pool_labels"
ADD
  CONSTRAINT amazon_eks_profile_node_pool_labels_node_pool_profile_id_amazon_eks_profile_node_pools_id_foreign FOREIGN KEY (node_pool_profile_id) REFERENCES amazon_eks_profile_node_pools(id) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
  "azure_aks_profile_node_pools"
ADD
  CONSTRAINT azure_aks_profile_node_pools_name_azure_aks_profiles_name_foreign FOREIGN KEY ("name") REFERENCES azure_aks_profiles(name) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
  "azure_aks_profile_node_pool_labels"
ADD
  CONSTRAINT azure_aks_profile_node_pool_labels_node_pool_profile_id_azure_aks_profile_node_pools_id_foreign FOREIGN KEY (node_pool_profile_id) REFERENCES azure_aks_profile_node_pools(id) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
  "google_gke_profile_node_pools"
ADD
  CONSTRAINT google_gke_profile_node_pools_name_google_gke_profiles_name_foreign FOREIGN KEY ("name") REFERENCES google_gke_profiles(name) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
  "google_gke_profile_node_pool_labels"
ADD
  CONSTRAINT google_gke_profile_node_pool_labels_node_pool_profile_id_google_gke_profile_node_pools_id_foreign FOREIGN KEY (node_pool_profile_id) REFERENCES google_gke_profile_node_pools(id) ON DELETE RESTRICT ON UPDATE RESTRICT;

CREATE TABLE "amazon_route53_domains" (
  "id" serial,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "organization_id" integer NOT NULL,
  "domain" text NOT NULL,
  "hosted_zone_id" text,
  "policy_arn" text,
  "iam_user" text,
  "aws_access_key_id" text,
  "status" text NOT NULL,
  "error_message" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_amazon_route53_domains_organization_id ON "amazon_route53_domains"(organization_id);

CREATE UNIQUE INDEX idx_amazon_route53_domains_domain ON "amazon_route53_domains"("domain");

CREATE TABLE "spotguide_repos" (
  "id" serial,
  "organization_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "name" text,
  "icon" bytea,
  "readme" text,
  "version" text,
  "spotguide_yaml_raw" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_spotguide_name_and_version ON "spotguide_repos"(
  organization_id, "name", "version"
);

CREATE TABLE "audit_events" (
  "id" serial,
  "time" timestamp with time zone,
  "correlation_id" varchar(36),
  "client_ip" varchar(45),
  "user_agent" text,
  "path" varchar(8000),
  "method" varchar(7),
  "user_id" integer,
  "status_code" integer,
  "body" json,
  "headers" json,
  "response_time" integer,
  "response_size" integer,
  "errors" json,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_audit_events_time ON "audit_events"("time");

CREATE TABLE "cluster_status_history" (
  "id" serial,
  "cluster_id" integer NOT NULL,
  "cluster_name" text NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "from_status" text NOT NULL,
  "from_status_message" text NOT NULL,
  "to_status" text NOT NULL,
  "to_status_message" text NOT NULL,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_cluster_status_history_cluster_id ON "cluster_status_history"(cluster_id);

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

CREATE TABLE "amazon_buckets" (
  "id" serial,
  "organization_id" integer NOT NULL,
  "name" text,
  "region" text,
  "secret_ref" text,
  "status" text,
  "status_msg" text,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_amazon_buckets_organization_id ON "amazon_buckets"(organization_id);

CREATE UNIQUE INDEX idx_amazon_bucket_name ON "amazon_buckets"("name");

CREATE TABLE "azure_buckets" (
  "id" serial,
  "organization_id" integer NOT NULL,
  "name" text,
  "resource_group" text,
  "storage_account" text,
  "location" text,
  "secret_ref" text,
  "status" text,
  "status_msg" text,
  "access_secret_ref" text,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_azure_buckets_organization_id ON "azure_buckets"(organization_id);

CREATE UNIQUE INDEX idx_azure_bucket_name ON "azure_buckets"(
  "name", resource_group, storage_account
);

CREATE TABLE "google_buckets" (
  "id" serial,
  "organization_id" integer NOT NULL,
  "name" text,
  "location" text,
  "secret_ref" text,
  "status" text,
  "status_msg" text,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_google_buckets_organization_id ON "google_buckets"(organization_id);

CREATE UNIQUE INDEX idx_google_bucket_name ON "google_buckets"("name");

CREATE TABLE "google_gke_clusters" (
  "id" serial,
  "cluster_id" integer,
  "master_version" text,
  "node_version" text,
  "region" text,
  "project_id" text,
  "vpc" varchar(64),
  "subnet" varchar(64),
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_gke_cluster_id ON "google_gke_clusters"(cluster_id);

CREATE TABLE "google_gke_node_pools" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "name" text,
  "autoscaling" boolean DEFAULT false,
  "preemptible" boolean DEFAULT false,
  "node_min_count" integer,
  "node_max_count" integer,
  "node_count" integer,
  "node_instance_type" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_gke_np_cluster_id_name ON "google_gke_node_pools"(cluster_id, "name");

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

ALTER TABLE
  "google_gke_clusters"
ADD
  CONSTRAINT google_gke_clusters_cluster_id_clusters_id_foreign FOREIGN KEY (cluster_id) REFERENCES clusters(id) ON DELETE RESTRICT ON UPDATE RESTRICT;

ALTER TABLE
  "google_gke_node_pools"
ADD
  CONSTRAINT google_gke_node_pools_cluster_id_google_gke_clusters_cluster_id_foreign FOREIGN KEY (cluster_id) REFERENCES google_gke_clusters(cluster_id) ON DELETE RESTRICT ON UPDATE RESTRICT;

CREATE TABLE "oracle_buckets" (
  "id" serial,
  "org_id" integer NOT NULL,
  "compartment_id" text,
  "name" text,
  "location" text,
  "secret_ref" text,
  "status" text,
  "status_msg" text,
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

CREATE TABLE "oracle_oke_profiles" (
  "id" serial,
  "name" text,
  "location" text DEFAULT 'eu-frankfurt-1',
  "version" text DEFAULT 'v1.10.3',
  "ttl_minutes" integer NOT NULL DEFAULT 0,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_profiles_name ON "oracle_oke_profiles"("name");

CREATE TABLE "oracle_oke_profile_node_pools" (
  "id" serial,
  "name" text,
  "count" integer DEFAULT '1',
  "image" text DEFAULT 'Oracle-Linux-7.4',
  "shape" text DEFAULT 'VM.Standard1.1',
  "version" text DEFAULT 'v1.10.3',
  "profile_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_profile_node_pools_name_profile_id ON "oracle_oke_profile_node_pools"("name", profile_id);

CREATE TABLE "oracle_oke_profile_node_pool_labels" (
  "id" serial,
  "name" text,
  "value" text,
  "profile_node_pool_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_profile_node_pool_labels_name_profile_id ON "oracle_oke_profile_node_pool_labels"("name", profile_node_pool_id);

CREATE TABLE "oracle_oke_node_pool_labels" (
  "id" serial,
  "name" text,
  "value" text,
  "node_pool_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_oke_node_pool_labels_node_pool_id_name ON "oracle_oke_node_pool_labels"("name", node_pool_id);

CREATE TABLE "amazon_ec2_clusters" (
  "id" serial,
  "cluster_id" integer,
  "master_instance_type" text,
  "master_image" text,
  "current_workflow_id" text,
  "dex_enabled" boolean NOT NULL DEFAULT false,
  PRIMARY KEY ("id")
);

CREATE TABLE "topology_cris" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "runtime" text,
  "runtime_config" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "topology_kubeadms" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "extra_args" varchar(255),
  PRIMARY KEY ("id")
);

CREATE TABLE "topology_kubernetes" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "version" text,
  "rbac_enabled" boolean,
  PRIMARY KEY ("id")
);

CREATE TABLE "topology_networks" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "service_cidr" text,
  "pod_cidr" text,
  "provider" text,
  "api_server_address" text,
  "cloud_provider" text,
  "cloud_provider_config" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "topology_nodepools" (
  "node_pool_id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "cluster_id" integer,
  "name" text,
  "roles" varchar(255),
  "provider" text,
  "provider_config" text,
  "autoscaling" boolean DEFAULT false,
  PRIMARY KEY ("node_pool_id")
);

CREATE UNIQUE INDEX idx_topology_nodepools_cluster_id_name ON "topology_nodepools"(cluster_id, "name");

CREATE TABLE "topology_nodepool_hosts" (
  "id" serial,
  "created_at" timestamp with time zone,
  "created_by" integer,
  "node_pool_id" integer,
  "name" text,
  "private_ip" text,
  "network_interface" text,
  "roles" varchar(255),
  "labels" varchar(255),
  "taints" varchar(255),
  PRIMARY KEY ("id")
);

CREATE TABLE "azure_pke_node_pools" (
  "id" serial,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deleted_at" timestamp with time zone,
  "autoscaling" boolean,
  "cluster_id" integer,
  "created_by" integer,
  "desired_count" integer,
  "instance_type" text,
  "max" integer,
  "min" integer,
  "name" text,
  "roles" text,
  "subnet_name" text,
  "zones" text,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_azure_pke_node_pools_deleted_at ON "azure_pke_node_pools"(deleted_at);

CREATE TABLE "azure_pke_clusters" (
  "id" serial,
  "cluster_id" integer,
  "resource_group_name" text,
  "virtual_network_location" text,
  "virtual_network_name" text,
  "active_workflow_id" text,
  "kubernetes_version" text,
  PRIMARY KEY ("id")
);

CREATE TABLE "ark_backups" (
  "id" serial,
  "uid" text,
  "name" text,
  "cloud" text,
  "distribution" text,
  "node_count" integer,
  "content_checked" boolean,
  "started_at" timestamp with time zone,
  "completed_at" timestamp with time zone,
  "expire_at" timestamp with time zone,
  "state" json,
  "nodes" json,
  "status" text,
  "status_message" text,
  "organization_id" integer NOT NULL,
  "cluster_id" integer NOT NULL,
  "deployment_id" integer NOT NULL,
  "bucket_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_ark_backups_organization_id ON "ark_backups"(organization_id);

CREATE INDEX idx_ark_backups_cluster_id ON "ark_backups"(cluster_id);

CREATE TABLE "ark_backup_buckets" (
  "id" serial,
  "cloud" text,
  "secret_id" text,
  "bucket_name" text,
  "location" text,
  "storage_account" text,
  "resource_group" text,
  "status" text,
  "status_message" text,
  "organization_id" integer NOT NULL,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deleted_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_ark_backup_buckets_organization_id ON "ark_backup_buckets"(organization_id);

CREATE TABLE "ark_restores" (
  "id" serial,
  "uid" text,
  "name" text,
  "state" json,
  "results" json,
  "warnings" integer,
  "errors" integer,
  "bucket_id" integer NOT NULL,
  "cluster_id" integer NOT NULL,
  "organization_id" integer NOT NULL,
  "status" text,
  "status_message" text,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deleted_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_ark_restores_bucket_id ON "ark_restores"(bucket_id);

CREATE INDEX idx_ark_restores_cluster_id ON "ark_restores"(cluster_id);

CREATE INDEX idx_ark_restores_organization_id ON "ark_restores"(organization_id);

CREATE TABLE "ark_deployments" (
  "id" serial,
  "name" text,
  "namespace" text,
  "restore_mode" boolean,
  "status" text,
  "status_message" text,
  "bucket_id" integer NOT NULL,
  "organization_id" integer NOT NULL,
  "cluster_id" integer NOT NULL,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "deleted_at" timestamp with time zone,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_ark_deployments_bucket_id ON "ark_deployments"(bucket_id);

CREATE INDEX idx_ark_deployments_organization_id ON "ark_deployments"(organization_id);

CREATE INDEX idx_ark_deployments_cluster_id ON "ark_deployments"(cluster_id);

CREATE TABLE "notifications" (
  "id" serial,
  "message" text NOT NULL,
  "initial_time" timestamp with time zone NOT NULL DEFAULT current_timestamp,
  "end_time" timestamp with time zone NOT NULL DEFAULT '1970-01-01 00:00:01',
  "priority" integer NOT NULL,
  PRIMARY KEY ("id")
);

CREATE INDEX idx_initial_time_end_time ON "notifications"(initial_time, end_time);
