ALTER TABLE `alibaba_acsk_node_pools` DROP COLUMN `count`;
ALTER TABLE `alibaba_acsk_node_pools` DROP COLUMN `image`;
ALTER TABLE `alibaba_acsk_node_pools` DROP COLUMN `system_disk_category`;
ALTER TABLE `alibaba_acsk_node_pools` DROP COLUMN `system_disk_size`;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `min_count` int(11) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `max_count` int(11) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `asg_id` varchar(255) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `scaling_conf_id` varchar(255) DEFAULT NULL;
