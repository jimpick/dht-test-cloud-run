
gcloud-build:
	gcloud builds submit --tag gcr.io/dht-test-249818/dht-test

gcloud-deploy:
	gcloud beta run deploy --image gcr.io/dht-test-249818/dht-test --platform managed
