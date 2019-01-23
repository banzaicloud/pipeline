ALTER TABLE `google_gke_clusters` ADD CONSTRAINT `fk_google_gke_clusters_cluster_id` FOREIGN KEY (`cluster_id`) REFERENCES `clusters`(`id`);
ALTER TABLE `google_gke_node_pools` ADD CONSTRAINT `fk_google_gke_node_pools_cluster_id` FOREIGN KEY (`cluster_id`) REFERENCES `google_gke_clusters`(`cluster_id`);
ALTER TABLE `google_gke_node_pool_labels` ADD CONSTRAINT `fk_google_gke_node_pool_labels_node_pool_id` FOREIGN KEY (`node_pool_id`) REFERENCES `google_gke_node_pools`(`id`);
