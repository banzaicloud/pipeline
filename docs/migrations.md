# Database Schema migrations

As of our first public release using Gorm's auto-migrate feature in production is not acceptable anymore.
Instead we have to write schema migrations manually.

Pipeline supports both MySQL and PostgreSQL, replace the `[dialect]` part with the corresponding SQL dialect's name (`mysql`/`postgres`).

An example migration looks like this:

```sql
/* database/migrations/[dialect]/UNIXTIMESTAMP_short_description.up.sql */

CREATE TABLE my_table /* ... */;
```

When developing multiple features in parallel being able to roll back to a certain version helps the development:

```sql
/* database/migrations/[dialect]/UNIXTIMESTAMP_short_description.down.sql */

DROP TABLE my_table /* ... */;
```

Before submitting migrations, please:

1. Disable auto-migrate
2. Run the migration manually
3. Test that the change works

## Using golang-migrate

[Golang-migrate](https://github.com/golang-migrate/migrate) is a CLI tool that can be used to create and apply database migrations.
If you need to write a migration script, please use it to test the migration locally.

Issue commands from the pipeline source directory.
```
cd $GOPATH/src/github.com/banzaicloud/pipeline
```

Download the `golang-migrate` tool to the `./bin` directory.
```
make bin/migrate
```

Check the version of your database in case of MySQL:
```bash
bin/migrate  -source "file://$(pwd)/database/migrations/mysql" -database "mysql://$MYSQL_USER:$MYSQL_PW@tcp(127.0.0.1:3306)/pipeline" version
```

And in case of PostgreSQL:

```bash
bin/migrate -source "file://database/migrations/postgres" -database "postgres://$POSTGRES_USER:$POSTGRES_PW@127.0.0.1:5432/pipeline?sslmode=disable" version
```

__NOTE: The only two difference between MySQL and PostgreSQL migrations are the directory holding the migration scripts and the database URI, so we don't replicate the documentation for both databases. Please replace those two parameters in every corresponding command.__

When using the migrate tool for the first time, you probably have a dirty version: use the `force` command to go to the latest version before your migration script.
`version` is a unix timestamp that's generated for the migration scripts (example: 1547126472).
Find the latest timestamp in `database/migrations/mysql` and force that version (if the `find` command doesn't work for some reason, find the latest timestamp manually):

```
find database/migrations/mysql -type f -exec basename {} ';' | cut -d_ -f1 | sort -r | head -1
bin/migrate  -source "file://$(pwd)/database/migrations/mysql" -database "mysql://$MYSQL_USER:$MYSQL_PW@tcp(127.0.0.1:3306)/pipeline" force $last_version
```

Create a migration script with a unix timestamp, and a title (example: `title="add_service_mesh"`).
The command should create 2 files in the `database/migrations/mysql` directory: `$timestamp_$title.up.sql` and `$timestamp_$title.down.sql`).
```
bin/migrate create  -ext sql -dir database/migrations/mysql -format "unix" "$title"
```

Write your migration scripts in the generated files.
Disable auto-migrate in Pipeline's `config.toml` file.

To apply your new migration script on the database, run the `up` command (1 means moving up one version):
```
bin/migrate  -source "file://$(pwd)/database/migrations/mysql" -database "mysql://$MYSQL_USER:$MYSQL_PW@tcp(127.0.0.1:3306)/pipeline" up 1
```

Run Pipeline, and test if your changes are working on the updated schema.

To revert back to the previous version, use the `down` command:
```
bin/migrate  -source "file://$(pwd)/database/migrations/mysql" -database "mysql://$MYSQL_USER:$MYSQL_PW@tcp(127.0.0.1:3306)/pipeline" down 1  
```
