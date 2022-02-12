# Cleanup
psql -d postgres -f data/cleanup.sql
# EMV: dev
psql -d postgres -f data/seed.sql # Set a proper db password for production
psql -d project_legacy -f data/schema.sql
# ENV: test
psql -d postgres -f data/seed_test.sql
psql -d project_legacy_test -f data/schema.sql
psql -d project_legacy_test -f data/schema_test.sql
