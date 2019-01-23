CREATE TABLE `scale_options` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `enabled` tinyint(1) DEFAULT NULL,
  `desired_cpu` double DEFAULT NULL,
  `desired_mem` double DEFAULT NULL,
  `desired_gpu` int(11) DEFAULT NULL,
  `on_demand_pct` int(11) DEFAULT NULL,
  `excludes` text COLLATE utf8mb4_unicode_ci,
  `keep_desired_capacity` tinyint(1) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ux_cluster_id` (`cluster_id`),
  CONSTRAINT `scale_options_cluster_id_clusters_id_foreign` FOREIGN KEY (`cluster_id`) REFERENCES `clusters` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
