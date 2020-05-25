CREATE TABLE `processes` (
  `id` varchar(255) NOT NULL,
  `parent_id` varchar(255) DEFAULT NULL,
  `org_id` int(10) unsigned NOT NULL,
  `type` varchar(255) NOT NULL,
  `log` text,
  `resource_id` varchar(255) NOT NULL,
  `status` varchar(255) NOT NULL,
  `started_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `finished_at` timestamp NULL,
  PRIMARY KEY (`id`),
  KEY `idx_processes_parent_id` (`parent_id`),
  KEY `idx_start_time_end_time` (`started_at`,`finished_at`)
);

CREATE TABLE `process_events` (
  `id` int(10) unsigned NOT NULL AUTO_INCREMENT,
  `process_id` varchar(255) NOT NULL,
  `type` varchar(255) NOT NULL,
  `log` text,
  `status` varchar(255) NOT NULL,
  `timestamp` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `process_events_process_id_processes_id_foreign` (`process_id`),
  CONSTRAINT `process_events_process_id_processes_id_foreign` FOREIGN KEY (`process_id`) REFERENCES `processes` (`id`)
);
