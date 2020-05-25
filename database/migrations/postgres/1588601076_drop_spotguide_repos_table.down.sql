CREATE TABLE "spotguide_repos" (
  "id" serial,
  "organization_id" integer,
  "created_at" timestamp with time zone,
  "updated_at" timestamp with time zone,
  "name" text,
  "icon" bytea,
  "readme" text,
  "version" text,
  "spotguide_yaml_raw" text,
  PRIMARY KEY ("id")
);

CREATE UNIQUE INDEX idx_spotguide_name_and_version ON "spotguide_repos"(
  organization_id, "name", "version"
);