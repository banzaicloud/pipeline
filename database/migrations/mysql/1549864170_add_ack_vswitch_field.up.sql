ALTER TABLE `alibaba_acsk_clusters` ADD COLUMN `v_switch_id` varchar(255) DEFAULT NULL;

UPDATE `alibaba_acsk_clusters` SET `v_switch_id` = NULL;
