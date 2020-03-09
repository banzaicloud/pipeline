CREATE TABLE `vsphere_pke_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `spec` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vsphere_pke_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
