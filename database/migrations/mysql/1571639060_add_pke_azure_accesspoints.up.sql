ALTER TABLE `azure_pke_clusters` ADD COLUMN `access_points` JSON DEFAULT NULL;
ALTER TABLE `azure_pke_clusters` ADD COLUMN `api_server_access_points` JSON DEFAULT NULL;
