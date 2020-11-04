ALTER TABLE `auth_identities` ADD COLUMN `sign_logs` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL AFTER confirmed_at;
