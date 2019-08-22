module github.com/jimpick/dht-test-cloud-run

require (
	cloud.google.com/go/logging v1.0.0
	contrib.go.opencensus.io/exporter/stackdriver v0.12.5
	github.com/ipfs/go-ds-leveldb v0.0.2
	github.com/ipfs/go-ipfs-config v0.0.6
	github.com/ipfs/go-ipns v0.0.1
	github.com/ipfs/go-log v0.0.1
	github.com/libp2p/go-eventbus v0.0.3 // indirect
	github.com/libp2p/go-libp2p v0.3.0
	github.com/libp2p/go-libp2p-core v0.2.2
	github.com/libp2p/go-libp2p-host v0.1.0
	github.com/libp2p/go-libp2p-kad-dht v0.1.1
	github.com/libp2p/go-libp2p-record v0.1.1
	go.opencensus.io v0.22.0
	google.golang.org/api v0.8.0 // indirect
)

replace github.com/libp2p/go-libp2p-kad-dht => github.com/jimpick/go-libp2p-kad-dht v0.1.2-0.20190821161555-3d9fb29fd492

replace github.com/libp2p/go-libp2p-core => github.com/libp2p/go-libp2p-core v0.2.1-0.20190815235124-d29813389b68
