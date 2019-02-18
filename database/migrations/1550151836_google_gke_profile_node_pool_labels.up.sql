CREATE TABLE `google_gke_profile_node_pool_labels` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `value` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `profile_node_pool_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_name_profile_node_pool_id` (`name`,`profile_node_pool_id`),
  KEY `idx_google_gke_profile_node_pool_profile_id` (`profile_node_pool_id`),
  CONSTRAINT `fk_google_gke_profile_node_pool_profile_id` FOREIGN KEY (`profile_node_pool_id`) REFERENCES `google_gke_profile_node_pools` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE `google_gke_profile_node_pools` ADD CONSTRAINT `fk_google_gke_profile_node_pools_name` FOREIGN KEY (`name`) REFERENCES `google_gke_profiles`(`name`);
