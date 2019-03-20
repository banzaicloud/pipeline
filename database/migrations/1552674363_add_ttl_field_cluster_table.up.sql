ALTER TABLE `clusters` ADD COLUMN `ttl_minutes` int(10) unsigned DEFAULT '0' NOT NULL;
ALTER TABLE `google_gke_profiles` ADD COLUMN `ttl_minutes` int(10) unsigned DEFAULT '0' NOT NULL;
ALTER TABLE `amazon_eks_profiles` ADD COLUMN `ttl_minutes` int(10) unsigned DEFAULT '0' NOT NULL;
ALTER TABLE `azure_aks_profiles` ADD COLUMN `ttl_minutes` int(10) unsigned DEFAULT '0' NOT NULL;
ALTER TABLE `oracle_oke_profiles` ADD COLUMN `ttl_minutes` int(10) unsigned DEFAULT '0' NOT NULL;

