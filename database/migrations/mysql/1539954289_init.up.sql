CREATE TABLE `alibaba_acsk_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `provider_cluster_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `region_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `zone_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `master_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `master_system_disk_category` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `master_system_disk_size` int(11) DEFAULT NULL,
  `snat_entry` tinyint(1) DEFAULT NULL,
  `ssh_flags` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `alibaba_acsk_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `system_disk_category` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `system_disk_size` int(11) DEFAULT NULL,
  `image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `count` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cluster_id_name` (`cluster_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `alibaba_buckets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `org_id` int(10) unsigned NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `region` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_ref` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_bucket_name` (`name`),
  KEY `idx_alibaba_buckets_org_id` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_buckets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `organization_id` int(10) unsigned NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `region` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_ref` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_msg` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_bucket_name` (`name`),
  KEY `idx_amazon_buckets_organization_id` (`organization_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_ec2_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `master_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `master_image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_ec2_profile_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'm4.xlarge',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `spot_price` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '0.2',
  `autoscaling` tinyint(1) DEFAULT '0',
  `min_count` int(11) DEFAULT '1',
  `max_count` int(11) DEFAULT '2',
  `count` int(11) DEFAULT '1',
  `image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'ami-4d485ca7',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_node_name` (`name`,`node_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_ec2_profiles` (
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'eu-west-1',
  `master_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'm4.xlarge',
  `master_image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'ami-4d485ca7',
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_eks_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_eks_profile_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'm4.xlarge',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `spot_price` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '0.2',
  `autoscaling` tinyint(1) DEFAULT '0',
  `min_count` int(11) DEFAULT '1',
  `max_count` int(11) DEFAULT '2',
  `count` int(11) DEFAULT '1',
  `image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'ami-0a54c984b9f908c81',
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_node_name` (`name`,`node_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_eks_profiles` (
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `region` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'us-west-2',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '1.10',
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_spot_price` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `autoscaling` tinyint(1) DEFAULT NULL,
  `node_min_count` int(11) DEFAULT NULL,
  `node_max_count` int(11) DEFAULT NULL,
  `count` int(11) DEFAULT NULL,
  `node_image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cluster_id_name` (`cluster_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_route53_domains` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `organization_id` int(10) unsigned NOT NULL,
  `domain` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `hosted_zone_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `policy_arn` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `iam_user` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `aws_access_key_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `error_message` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uix_amazon_route53_domains_organization_id` (`organization_id`),
  UNIQUE KEY `uix_amazon_route53_domains_domain` (`domain`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `ark_backup_buckets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cloud` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `bucket_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `storage_account` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `resource_group` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_message` text COLLATE utf8mb4_unicode_ci,
  `organization_id` int(10) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ark_backup_buckets_organization_id` (`organization_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `ark_backups` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `uid` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cloud` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `distribution` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_count` int(10) unsigned DEFAULT NULL,
  `content_checked` tinyint(1) DEFAULT NULL,
  `started_at` timestamp NULL DEFAULT NULL,
  `completed_at` timestamp NULL DEFAULT NULL,
  `expire_at` timestamp NULL DEFAULT NULL,
  `state` json DEFAULT NULL,
  `nodes` json DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_message` text COLLATE utf8mb4_unicode_ci,
  `organization_id` int(10) unsigned NOT NULL,
  `cluster_id` int(10) unsigned NOT NULL,
  `deployment_id` int(10) unsigned NOT NULL,
  `bucket_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ark_backups_organization_id` (`organization_id`),
  KEY `idx_ark_backups_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `ark_deployments` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `namespace` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `restore_mode` tinyint(1) DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_message` text COLLATE utf8mb4_unicode_ci,
  `bucket_id` int(10) unsigned NOT NULL,
  `organization_id` int(10) unsigned NOT NULL,
  `cluster_id` int(10) unsigned NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ark_deployments_bucket_id` (`bucket_id`),
  KEY `idx_ark_deployments_organization_id` (`organization_id`),
  KEY `idx_ark_deployments_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `ark_restores` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `uid` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `state` json DEFAULT NULL,
  `results` json DEFAULT NULL,
  `warnings` int(10) unsigned DEFAULT NULL,
  `errors` int(10) unsigned DEFAULT NULL,
  `bucket_id` int(10) unsigned NOT NULL,
  `cluster_id` int(10) unsigned NOT NULL,
  `organization_id` int(10) unsigned NOT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_message` text COLLATE utf8mb4_unicode_ci,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_ark_restores_bucket_id` (`bucket_id`),
  KEY `idx_ark_restores_cluster_id` (`cluster_id`),
  KEY `idx_ark_restores_organization_id` (`organization_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `audit_events` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `time` timestamp NULL DEFAULT NULL,
  `client_ip` varchar(45) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_agent` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `path` varchar(8000) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `method` varchar(7) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_id` int(10) unsigned DEFAULT NULL,
  `status_code` int(11) DEFAULT NULL,
  `body` json DEFAULT NULL,
  `headers` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_audit_events_time` (`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `auth_identities` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `provider` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `uid` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `encrypted_password` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `user_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `confirmed_at` timestamp NULL DEFAULT NULL,
  `sign_logs` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `azure_aks_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `resource_group` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `kubernetes_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `azure_aks_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `autoscaling` tinyint(1) DEFAULT NULL,
  `node_min_count` int(11) DEFAULT NULL,
  `node_max_count` int(11) DEFAULT NULL,
  `count` int(11) DEFAULT NULL,
  `node_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cluster_id_name` (`cluster_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `azure_aks_profile_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `autoscaling` tinyint(1) DEFAULT '0',
  `min_count` int(11) DEFAULT '1',
  `max_count` int(11) DEFAULT '2',
  `count` int(11) DEFAULT '1',
  `node_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'Standard_D4_v2',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_node_name` (`name`,`node_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `azure_aks_profiles` (
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'eastus',
  `kubernetes_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '1.9.2',
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `azure_buckets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `organization_id` int(10) unsigned NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `resource_group` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `storage_account` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_ref` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_msg` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_bucket_name` (`name`,`resource_group`,`storage_account`),
  KEY `idx_azure_buckets_organization_id` (`organization_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `casbin_rule` (
  `p_type` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v0` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v1` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v2` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v3` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v4` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `v5` varchar(100) COLLATE utf8mb4_unicode_ci DEFAULT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `uid` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cloud` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `distribution` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `organization_id` int(10) unsigned DEFAULT NULL,
  `secret_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `config_secret_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ssh_secret_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `rbac_enabled` tinyint(1) DEFAULT NULL,
  `monitoring` tinyint(1) DEFAULT NULL,
  `logging` tinyint(1) DEFAULT NULL,
  `status_message` text COLLATE utf8mb4_unicode_ci,
  `created_by` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_uid` (`uid`),
  UNIQUE KEY `idx_unique_id` (`deleted_at`,`name`,`organization_id`),
  KEY `idx_clusters_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `dummy_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `kubernetes_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_count` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `google_buckets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `organization_id` int(10) unsigned NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_ref` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_msg` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_bucket_name` (`name`),
  KEY `idx_google_buckets_organization_id` (`organization_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `google_gke_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `master_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `region` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `project_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `google_gke_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `autoscaling` tinyint(1) DEFAULT '0',
  `node_min_count` int(11) DEFAULT NULL,
  `node_max_count` int(11) DEFAULT NULL,
  `node_count` int(11) DEFAULT NULL,
  `node_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cluster_id_name` (`cluster_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `google_gke_profile_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `autoscaling` tinyint(1) DEFAULT '0',
  `min_count` int(11) DEFAULT '1',
  `max_count` int(11) DEFAULT '2',
  `count` int(11) DEFAULT '1',
  `node_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'n1-standard-1',
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_node_name` (`name`,`node_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `google_gke_profiles` (
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'us-central1-a',
  `node_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '1.10',
  `master_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT '1.10',
  PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `kubernetes_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `metadata_raw` varbinary(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_buckets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `org_id` int(10) unsigned NOT NULL,
  `compartment_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_ref` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_msg` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_bucket_name_location_compartment` (`compartment_id`,`name`,`location`),
  KEY `idx_oracle_buckets_org_id` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `vcn_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `lb_subnet_id1` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `lb_subnet_id2` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `ocid` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cluster_model_id` int(10) unsigned DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_node_pool_labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `value` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_pool_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_node_pool_id_name` (`name`,`node_pool_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_node_pool_subnets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `subnet_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_pool_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_node_pool_id_subnet_id` (`subnet_id`,`node_pool_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'Oracle-Linux-7.4',
  `shape` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'VM.Standard1.1',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'v1.10.3',
  `quantity_per_subnet` int(10) unsigned DEFAULT '1',
  `ocid` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_cluster_id_name` (`name`,`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_profile_node_pool_labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `value` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `profile_node_pool_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_profile_node_pool_id` (`name`,`profile_node_pool_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_profile_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `count` int(10) unsigned DEFAULT '1',
  `image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'Oracle-Linux-7.4',
  `shape` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'VM.Standard1.1',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'v1.10.3',
  `profile_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_profile_id` (`name`,`profile_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_profiles` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'eu-frankfurt-1',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'v1.10.3',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `organizations` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `github_id` bigint(20) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  UNIQUE KEY `github_id` (`github_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `spotguide_repos` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `organization_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `icon` mediumtext COLLATE utf8mb4_unicode_ci,
  `readme` mediumtext COLLATE utf8mb4_unicode_ci,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `spotguide_yaml_raw` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name_and_version` (`organization_id`,`name`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_organizations` (
  `user_id` int(10) unsigned NOT NULL,
  `organization_id` int(10) unsigned NOT NULL,
  `role` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'admin',
  PRIMARY KEY (`user_id`,`organization_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `users` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `email` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `login` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  `image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `login` (`login`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
