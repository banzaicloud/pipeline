ALTER TABLE `google_gke_clusters` ADD COLUMN `vpc`      varchar(64) DEFAULT NULL;
ALTER TABLE `google_gke_clusters` ADD COLUMN `subnet`   varchar(64) DEFAULT NULL;
