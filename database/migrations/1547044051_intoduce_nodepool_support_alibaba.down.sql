ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `image` varchar(255) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `system_disk_category` varchar(255) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `system_disk_size` int(11) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` DROP COLUMN `min_count`;
ALTER TABLE `alibaba_acsk_node_pools` DROP COLUMN `max_count`;
ALTER TABLE `alibaba_acsk_node_pools` DROP COLUMN `asg_id`;
ALTER TABLE `alibaba_acsk_node_pools` DROP COLUMN `scaling_conf_id`;
