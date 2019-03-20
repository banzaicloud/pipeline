ALTER TABLE `clusters` DROP COLUMN `ttl_minutes`;
ALTER TABLE `google_gke_profiles` DROP COLUMN `ttl_minutes`;
ALTER TABLE `amazon_eks_profiles` DROP COLUMN `ttl_minutes`;
ALTER TABLE `azure_aks_profiles` DROP COLUMN `ttl_minutes`;
ALTER TABLE `oracle_oke_profiles` DROP COLUMN `ttl_minutes`;
