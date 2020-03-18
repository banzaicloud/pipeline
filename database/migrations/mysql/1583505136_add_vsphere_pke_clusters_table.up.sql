CREATE TABLE `vsphere_pke_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `spec` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vsphere_pke_cluster_id` (`cluster_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;


CREATE TABLE `vsphere_pke_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `autoscaling` tinyint(1) DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `size` int(11) DEFAULT NULL,
  `max_size` int(10) unsigned DEFAULT NULL,
  `min_size` int(10) unsigned DEFAULT NULL,
  `vcpu` int(11) DEFAULT NULL,
  `ram` int(11) DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `roles` json DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_vsphere_pke_np_cluster_id_name` (`cluster_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
