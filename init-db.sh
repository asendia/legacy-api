# ENV: dev
psql -d project_legacy -f data/schema.sql
# ENV: test
psql -d postgres -f data/seed_test.sql
psql -d project_legacy_test -f data/schema.sql
psql -d project_legacy_test -f data/schema_test.sql
