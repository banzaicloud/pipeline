ALTER TABLE `google_gke_clusters` DROP FOREIGN KEY `fk_google_gke_clusters_cluster_id`;
ALTER TABLE `google_gke_node_pools` DROP FOREIGN KEY `fk_google_gke_node_pools_cluster_id`;
ALTER TABLE `google_gke_node_pool_labels` DROP FOREIGN KEY `fk_google_gke_node_pool_labels_node_pool_id`;
