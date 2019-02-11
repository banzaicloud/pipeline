CREATE TABLE `azure_aks_node_pool_labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `value` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_pool_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_node_pool_id_name` (`name`,`node_pool_id`),
  KEY `idx_azure_node_pool_labels_node_pool_id` (`node_pool_id`),
  CONSTRAINT `fk_azure_node_pool_labels_node_pool_id` FOREIGN KEY (`node_pool_id`) REFERENCES `azure_aks_node_pools` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
