ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `min_count` int(11) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `max_count` int(11) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `asg_id` varchar(255) DEFAULT NULL;
ALTER TABLE `alibaba_acsk_node_pools` ADD COLUMN `scaling_config_id` varchar(255) DEFAULT NULL;
