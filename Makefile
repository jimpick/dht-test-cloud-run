run:
	LIBP2P_ALLOW_WEAK_RSA_KEYS=1 IPFS_LOGGING_FMT=nocolor go run main.go

test:
	go test -v

gcloud-build:
	go mod tidy
	go mod vendor
	gcloud builds submit --tag gcr.io/dht-test-249818/dht-test-2

gcloud-deploy:
	gcloud beta run deploy dht-test-2 --image gcr.io/dht-test-249818/dht-test-2 --platform managed --region asia-northeast1 --concurrency 1 --timeout 30s
	gcloud beta run deploy dht-test-2 --image gcr.io/dht-test-249818/dht-test-2 --platform managed --region us-central1 --concurrency 1 --timeout 30s
	gcloud beta run deploy dht-test-2 --image gcr.io/dht-test-249818/dht-test-2 --platform managed --region us-east1 --concurrency 1 --timeout 30s
	gcloud beta run deploy dht-test-2 --image gcr.io/dht-test-249818/dht-test-2 --platform managed --region europe-west1 --concurrency 1 --timeout 30s

gcloud-list-services:
	gcloud beta run services list --platform managed
