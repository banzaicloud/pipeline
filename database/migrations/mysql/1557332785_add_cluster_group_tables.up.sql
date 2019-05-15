CREATE TABLE `clustergroups` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `uid` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `organization_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_uid` (`uid`),
  UNIQUE KEY `idx_unique_id` (`deleted_at`,`name`,`organization_id`),
  KEY `idx_clustergroups_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `clustergroup_deployments` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cluster_group_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deployment_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `deployment_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `deployment_package` varbinary(255) DEFAULT NULL,
  `deployment_release_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `description` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `chart_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `namespace` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `organization_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `values` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_cid_rname` (`cluster_group_id`,`deployment_release_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `clustergroup_deployment_target_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cluster_group_deployment_id` int(10) unsigned DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `cluster_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `values` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_unique_dep_cl` (`cluster_group_deployment_id`,`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `clustergroup_features` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cluster_group_id` int(10) unsigned DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `properties` json DEFAULT NULL,
  `reconcile_state` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `last_reconcile_error` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `clustergroup_members` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cluster_group_id` int(10) unsigned DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
