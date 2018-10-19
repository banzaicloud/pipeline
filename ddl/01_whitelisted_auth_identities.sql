CREATE TABLE `whitelisted_auth_identities` (
    `id` int unsigned AUTO_INCREMENT,
    `created_at` timestamp NULL,
    `updated_at` timestamp NULL,
    `provider` varchar(255),
    `type` ENUM('User', 'Organization'),
    `login` varchar(255),
    `uid` varchar(255),
    PRIMARY KEY (`id`))

CREATE UNIQUE INDEX provider_login ON `whitelisted_auth_identities`(`provider`, `login`)
