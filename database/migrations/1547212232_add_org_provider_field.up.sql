ALTER TABLE `organizations` ADD COLUMN `provider` varchar(255) NOT NULL;

UPDATE `organizations` SET `provider` = 'github';
