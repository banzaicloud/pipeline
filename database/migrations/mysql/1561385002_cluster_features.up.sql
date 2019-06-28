create table cluster_features
(
    id         int unsigned auto_increment
        primary key,
    created_at timestamp    null,
    updated_at timestamp    null,
    deleted_at timestamp    null,
    name       varchar(255) null,
    status     varchar(255) null,
    cluster_id int unsigned null,
    spec       text         null,
    created_by int unsigned null
);

create index idx_cluster_feature_deleted_at
    on cluster_features (deleted_at);

