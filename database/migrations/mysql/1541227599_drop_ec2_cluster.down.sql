CREATE TABLE `amazon_ec2_clusters` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `master_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `master_image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_ec2_profile_node_pools` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'm4.xlarge',
    `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `node_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `spot_price` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `autoscaling` tinyint(1) DEFAULT '0',
    `min_count` int(11) DEFAULT '1',
    `max_count` int(11) DEFAULT '2',
    `count` int(11) DEFAULT '1',
    `image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'ami-4d485ca7',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_name_node_name` (`name`,`node_name`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `amazon_ec2_profiles` (
    `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
    `created_at` timestamp NULL DEFAULT NULL,
    `updated_at` timestamp NULL DEFAULT NULL,
    `location` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'eu-west-1',
    `master_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'm4.xlarge',
    `master_image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'ami-4d485ca7',
    PRIMARY KEY (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
