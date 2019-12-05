ALTER TABLE `clusters` ADD COLUMN `monitoring` tinyint(1);
ALTER TABLE `clusters` ADD COLUMN `logging` tinyint(1);
ALTER TABLE `clusters` ADD COLUMN `security_scan` tinyint(1);
