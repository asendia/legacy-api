# legacy-api

## Prerequisites
- [Go 1.7](https://go.dev/doc/install)
- [Postgresql 14.1](https://www.postgresql.org/download/)
- [sqlc](https://docs.sqlc.dev/en/latest/overview/install.html) (Optional, for generating db structs from data/schema.sql & data/query.sql)
- [pgAdmin4](https://www.pgadmin.org/download/) (Optional, to manage the database or use psql instead)
- [gcloud cli](https://cloud.google.com/sdk/docs/install) (Optional, for deploying the api to Google Cloud Platform)
- [cloud_sql_proxy](https://cloud.google.com/sql/docs/mysql/connect-admin-proxy) (Optional, proxy to connect to cloud sql)

## Development
### Database setup
After installing go & postgresql
```sh
./init-db.sh # Prepare dev database - set proper passwords & secrets for production
```

### Testing
This is integration test, you will need to run the database first before running the test
```sh
cp .env-test-template.yaml .env-test.yaml
go test ./...
```

### Running the app in localhost
From the root directory of this repo
```sh
# You need to specify the env because the default value is "test"
# and I use the env to customize static file directories
ENVIRONMENT=dev go run cmd/main.go # Or just use vscode debug feature
```

### API call examples
1. Install [thunder client](https://www.thunderclient.com/), a vscode extension similar to postman
2. Import `thunder-collection_legacy-api.json` from thunder client

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

# To send emails
echo -n "PUT_THE_MAILJET_PRIVATE_KEY_HERE" | \
  gcloud secrets create "mailjet_private_key" --replication-policy "automatic" --data-file -

# To send emails
echo -n "PUT_THE_SENDGRID_PRIVATE_KEY_HERE" | \
  gcloud secrets create "sendgrid_private_key" --replication-policy "automatic" --data-file -
```
2. Give the secret manager read access to your project service account. You can actually see the exact command when you deploy the function using commands in the next step
```sh
gcloud projects add-iam-policy-binding [YOUR_GCLOUD_PROJECT_NAME] --member='serviceAccount:[YOUR_GCLOUD_PROJECT_NAME]@appspot.gserviceaccount.com' --role='roles/secretmanager.secretAccessor'
```
3. Prepare the DB
```sh
# Create an sql instance
gcloud sql instances create project-legacy-db \
  --database-version=POSTGRES_14 \
  --tier=db-f1-micro \
  --region=asia-southeast1 \
  --storage-size=10
# Set the root password
gcloud sql users set-password postgres \
  --instance=project-legacy-db \
  --password=[DB_ROOT_PASSWORD]
# Active cloud sql proxy, get the connection name from Google Cloud dashboard
# It looks like this: project-name:asia-southeast1:project-legacy-db
cloud_sql_proxy -instances=[CONNECTION NAME]=tcp:127.0.0.1:5678
# Connect to db
psql -h 127.0.0.1 -p 5678 -U postgres

##################################################################
# Copy paste the query in data/seed.sql, edit the PASSWORD field #
##################################################################

# Switch to project_legacy database
\c project_legacy

###########################################
# Copy paste the query in data/schema.sql #
###########################################
```
4. Create the production config file
```sh
cp .env-prod-template.yaml .env-prod.yaml
# Then edit the .env-prod.yaml, follow the comments provided in the file
```
5. Deploy the functions
```sh
# CloudFunctionForFrontendWithNetlifyJWT: legacy-api
gcloud functions deploy legacy-api --allow-unauthenticated \
  --entry-point CloudFunctionForFrontendWithNetlifyJWT --trigger-http \
  --region asia-southeast1 --runtime go116 --memory 128MB --timeout 15s \
  --update-labels service=legacy --max-instances 100 \
  --set-secrets DB_PASSWORD=db_password:latest,ENCRYPTION_KEY=encryption_key:latest \
  --env-vars-file .env-prod.yaml

# CloudFunctionForFrontendWithUserSecret: legacy-api-secret
gcloud functions deploy legacy-api-secret --allow-unauthenticated \
  --entry-point CloudFunctionForFrontendWithUserSecret --trigger-http \
  --region asia-southeast1 --runtime go116 --memory 128MB --timeout 15s \
  --update-labels service=legacy --max-instances 100 \
  --set-secrets DB_PASSWORD=db_password:latest,ENCRYPTION_KEY=encryption_key:latest \
  --env-vars-file .env-prod.yaml

# If you want to upload manually, zip for Google Cloud Function
zip -r legacy-api.zip api/ data/ mail/ secure/ simple/ .env functionForFrontendWithNetlifyJWT.go functionForFrontendWithUserSecret.go functionForSchedulerWithStaticSecret.go go.mod go.sum
```
6. Deploy the scheduler
```sh
# Create a pub/sub topic - this might take a while
gcloud pubsub topics create project-legacy-scheduler

# Create a google cloud scheduler
gcloud scheduler jobs create pubsub SendReminderMessages --location asia-southeast1 --schedule "22 19 * * *" \
  --topic project-legacy-scheduler --attributes action=send-reminder-messages \
  --description "Send reminder messages daily" --time-zone "Asia/Jakarta"
gcloud scheduler jobs create pubsub SendTestaments --location asia-southeast1 --schedule "38 19 * * *" \
  --topic project-legacy-scheduler --attributes action=send-testaments \
  --description "Send reminder messages daily" --time-zone "Asia/Jakarta"

# CloudFunctionForSchedulerWithStaticSecret: legacy-api-scheduler
gcloud functions deploy legacy-api-scheduler \
  --entry-point CloudFunctionForSchedulerWithStaticSecret --trigger-topic project-legacy-scheduler \
  --region asia-southeast1 --runtime go116 --memory 128MB --timeout 15s \
  --update-labels service=legacy --max-instances 100 \
  --set-secrets DB_PASSWORD=db_password:latest,STATIC_SECRET=static_secret:latest,ENCRYPTION_KEY=encryption_key:latest,MAILJET_PRIVATE_KEY=mailjet_private_key:latest,SENDGRID_PRIVATE_KEY=sendgrid_private_key:latest \
  --env-vars-file .env-prod.yaml
```
