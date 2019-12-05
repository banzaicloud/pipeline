ALTER TABLE "azure_pke_node_pools" ALTER COLUMN "roles" TYPE text USING "roles"::text;
ALTER TABLE "azure_pke_node_pools" ALTER COLUMN "zones" TYPE text USING "zones"::text;
