# legacy-api
Backend API code for [sejiwo.com](https://sejiwo.com/)

Note: this guide uses Google Cloud SQL but currently sejiwo.com uses SupaBase since it has more competitive pricing.

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
Why do I use template config? Because I put secrets in my `.env-test.yaml` & I don't want to accidentally commit it. Please let me know how to do it better.

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

# Additional SSL cert for Supabase
# Download from https://supabase.com/docs/guides/database/connecting-to-postgres#connecting-with-ssl
cat prod-ca-2021.crt | \
  gcloud secrets create "supabase_ssl_certificate" --replication-policy "automatic" --data-file -

# To send emails
echo -n "PUT_THE_MAILJET_API_KEY_HERE" | \
  gcloud secrets create "mailjet_api_key" --replication-policy "automatic" --data-file -

# To send emails
echo -n "PUT_THE_MAILJET_SECRET_KEY_HERE" | \
  gcloud secrets create "mailjet_secret_key" --replication-policy "automatic" --data-file -

# To send emails
echo -n "PUT_THE_SENDGRID_API_KEY_HERE" | \
  gcloud secrets create "sendgrid_api_key" --replication-policy "automatic" --data-file -
```
2. Give the secret manager read access to your project service account.
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
4. Deploy the Cloud Run service
```sh
# Copy env
cp .env-prod-template.yaml .env-prod.yaml
# Then edit the .env-prod.yaml, follow the comments provided in the file

gcloud run deploy legacy-api --source . \
  --region=asia-southeast1 --allow-unauthenticated --timeout 15s \
  --min-instances 0 --max-instances 100 --cpu 1 --memory 128Mi \
  --set-secrets DB_PASSWORD=db_password:latest,STATIC_SECRET=static_secret:latest,ENCRYPTION_KEY=encryption_key:latest,MAILJET_API_KEY=mailjet_api_key:latest,MAILJET_SECRET_KEY=mailjet_secret_key:latest,SENDGRID_API_KEY=sendgrid_api_key:latest \
  --env-vars-file .env-prod.yaml --update-labels service=legacy --tag=main
```
5. Deploy the scheduler
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

# Copy env
cp .env.prod-cloud-function-template.yaml .env-prod-cloud-function.yaml

# CloudFunctionForSchedulerWithStaticSecret: legacy-api-scheduler
gcloud functions deploy legacy-api-scheduler \
  --entry-point CloudFunctionForSchedulerWithStaticSecret --trigger-topic project-legacy-scheduler \
  --region asia-southeast1 --runtime go119 --memory 128Mi --timeout 15s --gen2 \
  --update-labels service=legacy --max-instances 10 \
  --set-secrets DB_PASSWORD=db_password:latest,STATIC_SECRET=static_secret:latest,ENCRYPTION_KEY=encryption_key:latest,MAILJET_API_KEY=mailjet_api_key:latest,MAILJET_SECRET_KEY=mailjet_secret_key:latest,SENDGRID_API_KEY=sendgrid_api_key:latest \
  --env-vars-file .env-prod-cloud-function.yaml
```
