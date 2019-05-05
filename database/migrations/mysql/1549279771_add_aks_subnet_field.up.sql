ALTER TABLE `azure_aks_node_pools` ADD COLUMN `v_net_subnet_id` varchar(255) DEFAULT NULL;

UPDATE `azure_aks_node_pools` SET `v_net_subnet_id` = '';
