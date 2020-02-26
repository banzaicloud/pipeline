create table helm_repositories
(
    id                 int unsigned auto_increment
        primary key,
    created_at         timestamp    null,
    updated_at         timestamp    null,
    deleted_at         timestamp    null,
    organization_id    int unsigned null,
    name               varchar(255) null,
    url                varchar(255) null,
    password_secret_id varchar(255) null,
    tls_secret_id      varchar(255) null,
    constraint idx_org_name
        unique (organization_id, name)
);

create index idx_helm_repositories_deleted_at
    on helm_repositories (deleted_at);
