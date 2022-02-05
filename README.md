# legacy-api

## Prerequisites
- [Go 1.7](https://go.dev/doc/install)
- [Postgresql 14.1](https://www.postgresql.org/download/)
- [pgAdmin4](https://www.pgadmin.org/download/) (Optional, to manage the database or use psql instead)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (Optional, for generating db structs from data/schema.sql & data/query.sql)
- [gcloud cli](https://cloud.google.com/sdk/docs/install) (Optional, for deploying the api to Google Cloud Platform)

## Development
### Database setup
After installing go & postgresql

```sh
# Start postgres (M1 Mac), google how to start postgres in your env
pg_ctl -D /opt/homebrew/var/postgres start
# Use psql to connect to init the db
# EMV: dev
psql -d postgres -f data/seed.sql # Set a proper db password for production
psql -d project_legacy -f data/schema.sql
# ENV: test
psql -d postgres -f data/seed_test.sql
psql -d project_legacy_test -f data/schema.sql
psql -d project_legacy_test -f data/schema_test.sql

# Clone this repo somewhere
git clone git@github.com:asendia/legacy-api.git
# Go to the root directory of this repo
cd legacy-api
```

### Testing
You can run test in each of the packages available in this repo. For example if you want to test data package, from the root directory of this repo:

```sh
cd api
go test
```

### Running the app in localhost
From the root directory of this repo

```sh
ENVIRONMENT=dev go run cmd/main.go # Or just use vscode debug feature
```

### API call example
```sh
# Insert a legacy message
curl -XPOST "localhost:8080/legacy-api" -d '{"action":"insert-message", "data":{"inactivePeriodDays":30,"reminderIntervalDays":1,"messageContent":"This is real API content","emailCreator":"mock@mock","emailReceivers":["inka@kentut.com","inkamemang@kentut"]}}' -H "authorization: Bearer [YOUR_VALID_TOKEN]"

# Select a message by id
curl -XPOST "localhost:8080/legacy-api" -d '{"action":"select-message","data":{"id":"[PUT_THE_ID_HERE]","emailCreator":"mock@mock"}}' -H "authorization: Bearer [YOUR_VALID_TOKEN]"

# Extend a message
curl localhost:8080/legacy-api-user-secret?action=extend-message&id=[MESSAGE_ID_UUID]&secret=[EXTENSION_SECRET]
```

## Deployment

### Google Cloud
1. Create the secrets needed to run the apps
```sh
echo -n "PUT_THE_DB_PASSWORD_HERE" | \
  gcloud secrets create "db_password" --replication-policy "automatic" --data-file -

# This one needs to be exactly 69 characters length
echo -n "PUT_THE_STATIC_SECRET_HERE" | \
  gcloud secrets create "static_secret" --replication-policy "automatic" --data-file -

# 32 characters length for AES encryption
echo -n "PUT_THE_ENCRYPTION_KEY_HERE" | \
  gcloud secrets create "encryption_key" --replication-policy "automatic" --data-file -
```
2. Give the secret manager read access to your project service account. You can actually see the exact command when you deploy the function using commands in the next step
```sh
gcloud projects add-iam-policy-binding [YOUR_GCLOUD_PROJECT_NAME] --member='serviceAccount:[YOUR_GCLOUD_PROJECT_NAME]@appspot.gserviceaccount.com' --role='roles/secretmanager.secretAccessor'
```
3. Prepare the DB
```
# Connect to db
gcloud sql connect project-legacy-db
# Run these sql queries in this order
# 1. seed.sql - edit the db password here
# Switch to project_legacy database
\c project_legacy
# 2. schema.sql
```
4.  Deploy the functions
```sh
# CloudFunctionForFrontendWithNetlifyJWT: legacy-api
gcloud functions deploy legacy-api --allow-unauthenticated \
  --entry-point CloudFunctionForFrontendWithNetlifyJWT --trigger-http \
  --region asia-southeast1 --runtime go116 --memory 128MB --timeout 15s \
  --update-labels service=legacy --max-instances 100 \
  --set-secrets DB_PASSWORD=db_password:latest,STATIC_SECRET=static_secret:latest,ENCRYPTION_KEY=encryption_key:latest \
  --env-vars-file .env-prod.yaml

# CloudFunctionForFrontendWithUserSecret: legacy-api-secret
gcloud functions deploy legacy-api-secret --allow-unauthenticated \
  --entry-point CloudFunctionForFrontendWithUserSecret --trigger-http \
  --region asia-southeast1 --runtime go116 --memory 128MB --timeout 15s \
  --update-labels service=legacy --max-instances 100 \
  --set-secrets DB_PASSWORD=db_password:latest,STATIC_SECRET=static_secret:latest,ENCRYPTION_KEY=encryption_key:latest \
  --env-vars-file .env-prod.yaml

# CloudFunctionForSchedulerWithStaticSecret: legacy-api-scheduler
gcloud functions deploy legacy-api-scheduler --allow-unauthenticated \
  --entry-point CloudFunctionForSchedulerWithStaticSecret --trigger-http \
  --region asia-southeast1 --runtime go116 --memory 128MB --timeout 15s \
  --update-labels service=legacy --max-instances 100 \
  --set-secrets DB_PASSWORD=db_password:latest,STATIC_SECRET=static_secret:latest,ENCRYPTION_KEY=encryption_key:latest \
  --env-vars-file .env-prod.yaml

# If you want to upload manually, zip for Google Cloud Function
zip -r legacy-api.zip api/ data/ secure/ simple/ .env functionForFrontendWithNetlifyJWT.go functionForFrontendWithUserSecret.go functionForSchedulerWithStaticSecret.go go.mod go.sum
```
