DROP TABLE IF EXISTS `azure_aks_profile_node_pool_labels`;

ALTER TABLE `azure_aks_profile_node_pools` DROP FOREIGN KEY `fk_azure_aks_profile_node_pools_name`;
