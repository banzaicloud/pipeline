CREATE TABLE `amazon_ec2_clusters` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `master_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `master_image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
