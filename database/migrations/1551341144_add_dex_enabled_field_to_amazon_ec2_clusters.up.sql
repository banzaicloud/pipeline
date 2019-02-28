ALTER TABLE `amazon_ec2_clusters` ADD COLUMN `dex_enabled` tinyint(1) DEFAULT 0 NOT NULL;

UPDATE `amazon_ec2_clusters` SET `dex_enabled` = 0;

