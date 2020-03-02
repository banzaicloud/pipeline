CREATE TABLE `dummy_clusters` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `kubernetes_version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `node_count` int(11) DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
