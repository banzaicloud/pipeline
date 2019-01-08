ALTER TABLE `amazon_eks_clusters` CHANGE COLUMN `cluster_id` `cluster_id` int(10) unsigned DEFAULT '0';

ALTER TABLE `amazon_eks_clusters` DROP INDEX ux_cluster_id;
