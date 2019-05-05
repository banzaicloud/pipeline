DROP TABLE IF EXISTS `amazon_node_pool_labels`;

ALTER TABLE `amazon_eks_profile_node_pools` DROP FOREIGN KEY `fk_amazon_eks_profile_node_pools_name`;

DROP TABLE IF EXISTS `amazon_eks_profile_node_pool_labels`;
