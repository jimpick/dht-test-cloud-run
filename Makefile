test:
	go test -v

gcloud-build:
	go mod tidy
	go mod vendor
	gcloud builds submit --tag gcr.io/dht-test-249818/dht-test

gcloud-deploy:
	gcloud beta run deploy dht-test --image gcr.io/dht-test-249818/dht-test --platform managed --region asia-northeast1 --concurrency 1
	gcloud beta run deploy dht-test --image gcr.io/dht-test-249818/dht-test --platform managed --region us-central1 --concurrency 1
	gcloud beta run deploy dht-test --image gcr.io/dht-test-249818/dht-test --platform managed --region us-east1 --concurrency 1
	gcloud beta run deploy dht-test --image gcr.io/dht-test-249818/dht-test --platform managed --region europe-west1 --concurrency 1

gcloud-list-services:
	gcloud beta run services list --platform managed
