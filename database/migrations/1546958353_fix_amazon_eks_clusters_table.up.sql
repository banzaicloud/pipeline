ALTER TABLE `amazon_eks_clusters` CHANGE COLUMN `cluster_id` `cluster_id` int(10) unsigned DEFAULT NULL;

CREATE UNIQUE INDEX ux_cluster_id ON `amazon_eks_clusters` (`cluster_id`);
