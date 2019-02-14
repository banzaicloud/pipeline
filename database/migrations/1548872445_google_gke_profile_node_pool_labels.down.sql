DROP TABLE IF EXISTS `google_gke_profile_node_pool_labels`;

ALTER TABLE `google_gke_profile_node_pools` DROP FOREIGN KEY `fk_google_gke_profile_node_pools_name`;
