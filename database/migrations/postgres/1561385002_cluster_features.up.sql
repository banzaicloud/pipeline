create table cluster_features
(
    id         serial not null
        constraint cluster_features_pkey
            primary key,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name       text,
    status     text,
    cluster_id integer,
    spec       text,
    created_by integer
);

alter table cluster_features
    owner to sparky;

create index idx_cluster_features_deleted_at
    on cluster_features (deleted_at);

