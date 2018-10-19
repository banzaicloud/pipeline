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

TODO: Use golang-migrate for running migrations?
TODO: write acceptance tests with migrations?
