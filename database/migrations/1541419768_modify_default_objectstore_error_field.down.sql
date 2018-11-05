ALTER TABLE azure_buckets MODIFY COLUMN status_msg varchar(255);
ALTER TABLE amazon_buckets MODIFY COLUMN status_msg varchar(255);
ALTER TABLE google_buckets MODIFY COLUMN status_msg varchar(255);
