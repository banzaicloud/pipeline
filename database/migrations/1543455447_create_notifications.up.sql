CREATE TABLE `notifications` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `message` text COLLATE utf8mb4_unicode_ci NOT NULL,
    `initial_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `end_time` timestamp NOT NULL DEFAULT '1970-01-01 00:00:01',
    `priority` tinyint(4) NOT NULL,
    PRIMARY KEY (`id`),
    KEY `idx_initial_time_end_time` (`initial_time`,`end_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
