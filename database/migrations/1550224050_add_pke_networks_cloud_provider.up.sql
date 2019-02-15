ALTER TABLE `topology_networks`
  ADD `cloud_provider` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  ADD `cloud_provider_config` text COLLATE utf8mb4_unicode_ci;
