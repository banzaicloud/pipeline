ALTER TABLE oracle_buckets MODIFY COLUMN status_msg text;
ALTER TABLE alibaba_buckets ADD COLUMN status varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL;
ALTER TABLE alibaba_buckets ADD COLUMN status_msg text COLLATE utf8mb4_unicode_ci DEFAULT NULL;
