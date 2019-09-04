run:
	go run main.go

gcloud-build:
	go mod tidy
	go mod vendor
	gcloud builds submit --tag gcr.io/dht-test-249818/dht-test-entries-2

gcloud-deploy:
	gcloud beta run deploy dht-test-entries-2 --image gcr.io/dht-test-249818/dht-test-entries-2 --platform managed --region us-central1 --timeout 30s --set-env-vars=GOOGLE_CLOUD_PROJECT=dht-test-249818

gcloud-list-services:
	gcloud beta run services list --platform managed

deploy: gcloud-build
	make gcloud-deploy
