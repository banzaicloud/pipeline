CREATE TABLE `spotguide_repos` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `organization_id` int(10) unsigned DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `icon` mediumtext COLLATE utf8mb4_unicode_ci,
  `readme` mediumtext COLLATE utf8mb4_unicode_ci,
  `version` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `spotguide_yaml_raw` text COLLATE utf8mb4_unicode_ci,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_spotguide_name_and_version` (`organization_id`,`name`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
