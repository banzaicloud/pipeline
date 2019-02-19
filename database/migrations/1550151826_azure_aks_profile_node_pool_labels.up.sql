CREATE TABLE `azure_aks_profile_node_pool_labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `value` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `node_pool_profile_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_profile_node_pool_id` (`name`,`node_pool_profile_id`),
  KEY `idx_azure_aks_profile_node_pool_profile_id` (`node_pool_profile_id`),
  CONSTRAINT `fk_azure_aks_profile_node_pool_profile_id` FOREIGN KEY (`node_pool_profile_id`) REFERENCES `azure_aks_profile_node_pools` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `azure_aks_profile_node_pools` ADD CONSTRAINT `fk_azure_aks_profile_node_pools_name` FOREIGN KEY (`name`) REFERENCES `azure_aks_profiles`(`name`);
