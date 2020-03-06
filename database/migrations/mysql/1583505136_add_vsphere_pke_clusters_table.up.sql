CREATE TABLE `vsphere_pke_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `deleted_at` timestamp NULL DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `provider_data` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vsphere_pke_cluster_id` (`cluster_id`),
  KEY `idx_vsphere_pke_clusters_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
