CREATE TABLE "helm_repositories"
(
    "id"                 serial,
    "created_at"         timestamp with time zone,
    "updated_at"         timestamp with time zone,
    "organization_id"    integer,
    "name"               text,
    "url"                text,
    "password_secret_id" text,
    "tls_secret_id"      text,
    PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_org_name ON "helm_repositories" (organization_id, name);


