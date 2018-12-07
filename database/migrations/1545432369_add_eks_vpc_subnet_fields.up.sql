ALTER TABLE `amazon_eks_clusters` ADD COLUMN `vpc_id`             varchar(32) DEFAULT NULL;
ALTER TABLE `amazon_eks_clusters` ADD COLUMN `vpc_cidr`           varchar(18) DEFAULT NULL;
ALTER TABLE `amazon_eks_clusters` ADD COLUMN `route_table_id`     varchar(32) DEFAULT NULL;
ALTER TABLE `amazon_eks_clusters` ADD COLUMN `cluster_id`         int(10) unsigned DEFAULT 0;

UPDATE `amazon_eks_clusters` SET cluster_id=id;

SELECT @max := COALESCE(MAX(id),0)+1 FROM `amazon_eks_clusters`;

SET @q = CONCAT('ALTER TABLE amazon_eks_clusters AUTO_INCREMENT=', @max);
PREPARE stmt FROM @q;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

ALTER TABLE `amazon_eks_clusters`
ADD CONSTRAINT `amazon_eks_clusters_cluster_id_clusters_id_foreign`
FOREIGN KEY (`cluster_id`) REFERENCES `clusters` (`id`);


CREATE TABLE `amazon_eks_subnets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `subnet_id` varchar(32) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `cidr` varchar(18) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_cluster_id` (`cluster_id`),
  CONSTRAINT `amazon_eks_subnets_cluster_id_amazon_eks_clusters_id_foreign` FOREIGN KEY (`cluster_id`) REFERENCES `amazon_eks_clusters` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

