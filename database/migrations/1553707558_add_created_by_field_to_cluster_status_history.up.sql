ALTER TABLE `cluster_status_history` ADD COLUMN `created_by` int(10) unsigned DEFAULT '0' NOT NULL AFTER `created_at`;
