ALTER TABLE `vsphere_pke_node_pools` ADD COLUMN `admin_username` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL;
ALTER TABLE `vsphere_pke_node_pools` ADD COLUMN `template_name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL;
