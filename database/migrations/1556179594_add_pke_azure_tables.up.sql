CREATE TABLE `azure_pke_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `resource_group_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `virtual_network_location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `virtual_network_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `active_workflow_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `kubernetes_version` varchar(255),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `azure_pke_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `autoscaling` tinyint(1) DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `desired_count` int(10) unsigned DEFAULT NULL,
  `instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `max` int(10) unsigned DEFAULT NULL,
  `min` int(10) unsigned DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `roles` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `subnet_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `zones` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_azure_pke_node_pools_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
