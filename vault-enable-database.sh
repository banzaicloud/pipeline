#!/bin/sh

# This script showcases how to setup Vault in the docker-compose local environment

vault secrets enable database

set -euo pipefail

vault write database/config/my-mysql-database \
    plugin_name=mysql-database-plugin \
    connection_url="root:example@tcp(db:3306)/" \
    allowed_roles="pipeline"

vault write database/roles/pipeline \
    db_name=my-mysql-database \
    creation_statements="GRANT ALL ON *.* TO '{{name}}'@'%' IDENTIFIED BY '{{password}}';" \
    default_ttl="10m" \
    max_ttl="24h"
