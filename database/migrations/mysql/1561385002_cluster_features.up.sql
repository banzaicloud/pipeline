CREATE TABLE `clusterfeature`
(
    `id`         int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name`       varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `cluster_id` int(10) unsigned                        DEFAULT NULL,
    `spec`       text,
    `status`     text,
    `created_at` timestamp        NULL                   DEFAULT NULL,
    `updated_at` timestamp        NULL                   DEFAULT NULL,
    `deleted_at` timestamp        NULL                   DEFAULT NULL,
    `created_by` int(10) unsigned                        DEFAULT NULL,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;


