#!/bin/sh
set -euo pipefail

echo "Waiting Vault to launch on vault:8200..."
while ! nc -z vault 8200; do
  sleep 0.1 # wait for 1/10 of the second before check again
done
echo "Vault launched"

echo "Waiting DB to launch on db:3306..."
while ! nc -z db 3306; do
  sleep 0.1 # wait for 1/10 of the second before check again
done
echo "DB launched"

vault secrets enable database

vault write database/config/my-mysql-database \
    plugin_name=mysql-database-plugin \
    connection_url="root:example@tcp(db:3306)/" \
    allowed_roles="pipeline"

vault write database/roles/pipeline \
    db_name=my-mysql-database \
    creation_statements="GRANT ALL ON *.* TO '{{name}}'@'%' IDENTIFIED BY '{{password}}';" \
    default_ttl="10m" \
    max_ttl="24h"
