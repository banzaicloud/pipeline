ALTER TABLE `amazon_eks_clusters` DROP FOREIGN KEY  `amazon_eks_clusters_cluster_id_clusters_id_foreign`;

ALTER TABLE `amazon_eks_clusters` DROP COLUMN `vpc_id`;
ALTER TABLE `amazon_eks_clusters` DROP COLUMN `vpc_cidr`;
ALTER TABLE `amazon_eks_clusters` DROP COLUMN `route_table_id`;
ALTER TABLE `amazon_eks_clusters` DROP COLUMN `cluster_id`;


DROP TABLE IF EXISTS `amazon_eks_subnets`;


