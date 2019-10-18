ALTER TABLE `amazon_eks_clusters`
ADD COLUMN `default_user` tinyint(1) DEFAULT NULL,
ADD COLUMN `cluster_role_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
ADD COLUMN `node_instance_role_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL;
