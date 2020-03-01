ALTER TABLE `clusters` ADD COLUMN `ttl_minutes` int(10) unsigned DEFAULT '0' NOT NULL AFTER created_by;
