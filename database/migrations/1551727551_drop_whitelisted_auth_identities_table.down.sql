CREATE TABLE `whitelisted_auth_identities` (
    `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
    `created_at` timestamp NULL DEFAULT NULL,
    `updated_at` timestamp NULL DEFAULT NULL,
    `provider` varchar(255),
    `type` ENUM('User', 'Organization'),
    `login` varchar(255),
    `uid` varchar(255),
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE UNIQUE INDEX provider_login ON `whitelisted_auth_identities` (`provider`, `login`);
