CREATE TABLE `oracle_buckets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `org_id` int(10) unsigned NOT NULL,
  `compartment_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_ref` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_msg` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `namespace` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
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
  UNIQUE KEY `idx_oke_clusters_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_node_pool_subnets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `subnet_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_pool_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_oke_node_pool_subnets_id_subnet_id` (`subnet_id`,`node_pool_id`)
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
  UNIQUE KEY `idx_oke_node_pools_cluster_id_name` (`name`,`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
