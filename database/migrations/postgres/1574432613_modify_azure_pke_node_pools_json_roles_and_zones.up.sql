UPDATE "azure_pke_node_pools" SET "roles"='[]' WHERE "roles" = '';
UPDATE "azure_pke_node_pools" SET "roles"=concat('["', replace("roles", ',', '","'), '"]') WHERE NOT "roles" LIKE '%[%';
ALTER TABLE "azure_pke_node_pools" ALTER COLUMN "roles" TYPE json USING "roles"::json;

UPDATE "azure_pke_node_pools" SET "zones"='[]' WHERE "zones" = '';
UPDATE "azure_pke_node_pools" SET "zones"=concat('["', replace("zones", ',', '","'), '"]') WHERE NOT "zones" LIKE '%[%';
ALTER TABLE "azure_pke_node_pools" ALTER COLUMN "zones" TYPE json USING "zones"::json;
