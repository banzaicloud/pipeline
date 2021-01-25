CREATE TABLE `alibaba_buckets` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `org_id` int(10) unsigned NOT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `region` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `secret_ref` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `status_msg` text COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_alibaba_bucket_name` (`name`),
  KEY `idx_alibaba_buckets_org_id` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `alibaba_acsk_node_pools` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `created_at` timestamp NULL DEFAULT NULL,
  `created_by` int(10) unsigned DEFAULT NULL,
  `cluster_id` int(10) unsigned DEFAULT NULL,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `system_disk_category` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `system_disk_size` int(11) DEFAULT NULL,
  `image` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `count` int(11) DEFAULT NULL,
  `min_count` int(11) DEFAULT NULL,
  `max_count` int(11) DEFAULT NULL,
  `asg_id` varchar(255) DEFAULT NULL,
  `scaling_config_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_ack_node_pools_cluster_id_name` (`cluster_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `alibaba_acsk_clusters` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `provider_cluster_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `region_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `zone_id` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `master_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `master_system_disk_category` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  `master_system_disk_size` int(11) DEFAULT NULL,
  `snat_entry` tinyint(1) DEFAULT NULL,
  `ssh_flags` tinyint(1) DEFAULT NULL,
  `kubernetes_version` varchar(255),
  `v_switch_id` varchar(255) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

UPDATE `alibaba_acsk_clusters` SET `v_switch_id` = NULL;

