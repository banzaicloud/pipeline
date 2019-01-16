# Database Schema migrations

As of our first public release using Gorm's auto-migrate feature in production is not acceptable anymore.
Instead we have to write schema migrations manually.

An example migration looks like this:

```sql
/* database/migrations/UNIXTIMESTAMP_short_description.up.sql */

CREATE TABLE my_table /* ... */;
```

When developing multiple features in parallel being able to roll back to a certain version helps the development:

```sql
/* database/migrations/UNIXTIMESTAMP_short_description.down.sql */

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

Create a migration script with a unix timestamp, and a title (example: `title="add_service_mesh"`).
The command should create 2 files in the `database/migrations` directory: `$timestamp_$title.up.sql` and `$timestamp_$title.down.sql`).
```
bin/migrate create  -ext sql -dir database/migrations/ -format "unix" "$title"
```

Write your migration scripts in the generated files.
Disable auto-migrate in Pipeline's `config.toml` file.

Check the version of your database.
```
bin/migrate  -source "file://$(pwd)/database/migrations" -database "mysql://$MYSQL_USER:$MYSQL_PW@tcp(127.0.0.1:3306)/pipeline" version
```

When using the migrate tool for the first time, you probably have a dirty version: use the `force` command to go to the latest version before your migration script.
`version` is a unix timestamp that's generated for the migration scripts (example: 1547126472).
Find the latest timestamp in `database/migrations` and run this command:

```
bin/migrate  -source "file://$(pwd)/database/migrations" -database "mysql://$MYSQL_USER:$MYSQL_PW@tcp(127.0.0.1:3306)/pipeline" force $last_version
```

To apply your new migration script on the database, run the `up` command (1 means moving up one version):
```
bin/migrate  -source "file://$(pwd)/database/migrations" -database "mysql://$MYSQL_USER:$MYSQL_PW@tcp(127.0.0.1:3306)/pipeline" up 1
```

Run Pipeline, and test if your changes are working on the updated schema.

To revert back to the previous version, use the `down` command:
```
bin/migrate  -source "file://$(pwd)/database/migrations" -database "mysql://$MYSQL_USER:$MYSQL_PW@tcp(127.0.0.1:3306)/pipeline" down 1  
```

TODO: write acceptance tests with migrations?
