CREATE TABLE `amazon_eks_profiles`
(
    `name`        varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
    `created_at`  timestamp                               NULL     DEFAULT NULL,
    `updated_at`  timestamp                               NULL     DEFAULT NULL,
    `region`      varchar(255) COLLATE utf8mb4_unicode_ci          DEFAULT 'us-west-2',
    `version`     varchar(255) COLLATE utf8mb4_unicode_ci          DEFAULT '1.10',
    `ttl_minutes` int(10) unsigned                        NOT NULL DEFAULT '0',
    PRIMARY KEY (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `amazon_eks_profile_node_pools`
(
    `id`            int(10) unsigned NOT NULL AUTO_INCREMENT,
    `instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'm4.xlarge',
    `name`          varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `node_name`     varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `spot_price`    varchar(255) COLLATE utf8mb4_unicode_ci,
    `autoscaling`   tinyint(1)                              DEFAULT '0',
    `min_count`     int(11)                                 DEFAULT '1',
    `max_count`     int(11)                                 DEFAULT '2',
    `count`         int(11)                                 DEFAULT '1',
    `image`         varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'ami-0a54c984b9f908c81',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_amazon_name_node_name` (`name`, `node_name`),
    CONSTRAINT `fk_amazon_eks_profile_node_pools_name` FOREIGN KEY (`name`) REFERENCES `amazon_eks_profiles` (`name`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 3
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `amazon_eks_profile_node_pool_labels`
(
    `id`                   int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name`                 varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `value`                varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `node_pool_profile_id` int(10) unsigned                        DEFAULT NULL,
    `created_at`           timestamp        NULL                   DEFAULT NULL,
    `updated_at`           timestamp        NULL                   DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_eks_profile_node_pool_labels_id_name` (`name`, `node_pool_profile_id`),
    KEY `idx_amazon_eks_profile_node_pool_profile_id` (`node_pool_profile_id`),
    CONSTRAINT `fk_amazon_eks_profile_node_pool_profile_id` FOREIGN KEY (`node_pool_profile_id`) REFERENCES `amazon_eks_profile_node_pools` (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `azure_aks_profiles`
(
    `name`               varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
    `created_at`         timestamp                               NULL     DEFAULT NULL,
    `updated_at`         timestamp                               NULL     DEFAULT NULL,
    `location`           varchar(255) COLLATE utf8mb4_unicode_ci          DEFAULT 'eastus',
    `kubernetes_version` varchar(255) COLLATE utf8mb4_unicode_ci          DEFAULT '1.9.2',
    `ttl_minutes`        int(10) unsigned                        NOT NULL DEFAULT '0',
    PRIMARY KEY (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `azure_aks_profile_node_pools`
(
    `id`                 int(10) unsigned NOT NULL AUTO_INCREMENT,
    `autoscaling`        tinyint(1)                              DEFAULT '0',
    `min_count`          int(11)                                 DEFAULT '1',
    `max_count`          int(11)                                 DEFAULT '2',
    `count`              int(11)                                 DEFAULT '1',
    `node_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'Standard_D4_v2',
    `name`               varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `node_name`          varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_aks_profile_node_pools_name_node_name` (`name`, `node_name`),
    CONSTRAINT `fk_azure_aks_profile_node_pools_name` FOREIGN KEY (`name`) REFERENCES `azure_aks_profiles` (`name`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 3
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `azure_aks_profile_node_pool_labels`
(
    `id`                   int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name`                 varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `value`                varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `node_pool_profile_id` int(10) unsigned                        DEFAULT NULL,
    `created_at`           timestamp        NULL                   DEFAULT NULL,
    `updated_at`           timestamp        NULL                   DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_aks_profile_node_pool_labels_name_id` (`name`, `node_pool_profile_id`),
    KEY `idx_azure_aks_profile_node_pool_profile_id` (`node_pool_profile_id`),
    CONSTRAINT `fk_azure_aks_profile_node_pool_profile_id` FOREIGN KEY (`node_pool_profile_id`) REFERENCES `azure_aks_profile_node_pools` (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `google_gke_profiles`
(
    `name`           varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
    `created_at`     timestamp                               NULL     DEFAULT NULL,
    `updated_at`     timestamp                               NULL     DEFAULT NULL,
    `location`       varchar(255) COLLATE utf8mb4_unicode_ci          DEFAULT 'us-central1-a',
    `node_version`   varchar(255) COLLATE utf8mb4_unicode_ci          DEFAULT '1.10',
    `master_version` varchar(255) COLLATE utf8mb4_unicode_ci          DEFAULT '1.10',
    `ttl_minutes`    int(10) unsigned                        NOT NULL DEFAULT '0',
    PRIMARY KEY (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `google_gke_profile_node_pools`
(
    `id`                 int(10) unsigned NOT NULL AUTO_INCREMENT,
    `autoscaling`        tinyint(1)                              DEFAULT '0',
    `min_count`          int(11)                                 DEFAULT '1',
    `max_count`          int(11)                                 DEFAULT '2',
    `count`              int(11)                                 DEFAULT '1',
    `node_instance_type` varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'n1-standard-1',
    `name`               varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `node_name`          varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `preemptible`        tinyint(1)                              DEFAULT '0',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_gke_profile_node_pools_name_node_name` (`name`, `node_name`),
    CONSTRAINT `fk_google_gke_profile_node_pools_name` FOREIGN KEY (`name`) REFERENCES `google_gke_profiles` (`name`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 3
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `google_gke_profile_node_pool_labels`
(
    `id`                   int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name`                 varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `value`                varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `node_pool_profile_id` int(10) unsigned                        DEFAULT NULL,
    `created_at`           timestamp        NULL                   DEFAULT NULL,
    `updated_at`           timestamp        NULL                   DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_gke_name_profile_node_pool_id` (`name`, `node_pool_profile_id`),
    KEY `idx_google_gke_profile_node_pool_profile_id` (`node_pool_profile_id`),
    CONSTRAINT `fk_google_gke_profile_node_pool_profile_id` FOREIGN KEY (`node_pool_profile_id`) REFERENCES `google_gke_profile_node_pools` (`id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_profiles`
(
    `id`          int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name`        varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `location`    varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'eu-frankfurt-1',
    `version`     varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'v1.10.3',
    `created_at`  timestamp        NULL                   DEFAULT NULL,
    `updated_at`  timestamp        NULL                   DEFAULT NULL,
    `ttl_minutes` int(10) unsigned NOT NULL               DEFAULT '0',
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_oke_profiles_name` (`name`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 3
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_profile_node_pools`
(
    `id`         int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name`       varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `count`      int(10) unsigned                        DEFAULT '1',
    `image`      varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'Oracle-Linux-7.4',
    `shape`      varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'VM.Standard1.1',
    `version`    varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT 'v1.10.3',
    `profile_id` int(10) unsigned                        DEFAULT NULL,
    `created_at` timestamp        NULL                   DEFAULT NULL,
    `updated_at` timestamp        NULL                   DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_oke_profile_node_pools_name_profile_id` (`name`, `profile_id`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 3
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;

CREATE TABLE `oracle_oke_profile_node_pool_labels`
(
    `id`                   int(10) unsigned NOT NULL AUTO_INCREMENT,
    `name`                 varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `value`                varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
    `profile_node_pool_id` int(10) unsigned                        DEFAULT NULL,
    `created_at`           timestamp        NULL                   DEFAULT NULL,
    `updated_at`           timestamp        NULL                   DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_oke_profile_node_pool_labels_name_profile_id` (`name`, `profile_node_pool_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci;
