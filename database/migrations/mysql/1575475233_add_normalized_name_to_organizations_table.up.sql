ALTER TABLE `organizations` ADD COLUMN `normalized_name` varchar(255) AFTER `provider`, ADD UNIQUE KEY (`normalized_name`);

UPDATE organizations SET normalized_name = REPLACE(REPLACE(name, '.', '-'), '@', '-') WHERE normalized_name ='' OR normalized_name IS NULL;
