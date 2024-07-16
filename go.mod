module github.com/pahanini/go-grpc-bidirectional-streaming-example

go 1.21

toolchain go1.22.3

require (
	golang.org/x/net v0.27.0
	golang.stackrox.io/grpc-http1 v0.3.12
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.1
)

require (
	github.com/golang/glog v1.2.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	nhooyr.io/websocket v1.8.11 // indirect
)

replace golang.stackrox.io/grpc-http1 v0.3.12 => github.com/spectrocloud/go-grpc-http1 v0.0.1
