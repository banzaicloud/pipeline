ALTER TABLE oracle_buckets MODIFY COLUMN status_msg varchar(255);
ALTER TABLE alibaba_buckets DROP COLUMN status varchar(255);
ALTER TABLE alibaba_buckets DROP COLUMN status_msg text;
