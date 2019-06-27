create table if not exists clusterfeature
(
    id         int unsigned auto_increment
        primary key,
    created_at timestamp      null,
    updated_at timestamp      null,
    deleted_at timestamp      null,
    name       varchar(255)   null,
    status     varchar(255)   null,
    cluster_id int unsigned   null,
    spec       varbinary(255) null,
    created_by int unsigned   null
);

create index idx_clusterfeature_deleted_at
    on clusterfeature (deleted_at);

