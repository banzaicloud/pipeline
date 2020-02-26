create table helm_repositories
(
    id                 serial not null
        constraint helm_repositories_pkey
            primary key,
    created_at         timestamp with time zone,
    updated_at         timestamp with time zone,
    deleted_at         timestamp with time zone,
    organization_id    integer,
    name               text,
    url                text,
    password_secret_id text,
    tls_secret_id      text
);

alter table helm_repositories
    owner to sparky;

create index idx_helm_repositories_deleted_at
    on helm_repositories (deleted_at);

create unique index idx_org_name
    on helm_repositories (organization_id, name);
