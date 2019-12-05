ALTER TABLE "organizations" ADD COLUMN "normalized_name" text UNIQUE;

UPDATE organizations SET normalized_name = REPLACE(REPLACE(name, '.', '-'), '@', '-') WHERE normalized_name = '' OR normalized_name IS NULL;
