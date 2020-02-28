CREATE TABLE `helm_repositories` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY ,
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `organization_id` int(10) unsigned DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `url` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `password_secret_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `tls_secret_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  CONSTRAINT `idx_org_name` UNIQUE (`organization_id`, `name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;