 ALTER TABLE `audit_events` ADD COLUMN `response_time` int(11) DEFAULT NULL;
 ALTER TABLE `audit_events` ADD COLUMN `response_size` int(11) DEFAULT NULL;
 ALTER TABLE `audit_events` ADD COLUMN `errors` json DEFAULT NULL;
