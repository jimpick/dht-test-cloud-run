
gcloud-build:
	gcloud builds submit --tag gcr.io/dht-test-249818/dht-test

gcloud-deploy:
	gcloud beta run deploy dht-test --image gcr.io/dht-test-249818/dht-test --platform managed --region asia-northeast1
	gcloud beta run deploy dht-test --image gcr.io/dht-test-249818/dht-test --platform managed --region us-central1
	gcloud beta run deploy dht-test --image gcr.io/dht-test-249818/dht-test --platform managed --region us-east1
	gcloud beta run deploy dht-test --image gcr.io/dht-test-249818/dht-test --platform managed --region europe-west1

gcloud-list-services:
	gcloud beta run services list --platform managed
