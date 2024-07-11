package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"

	pb "github.com/pahanini/go-grpc-bidirectional-streaming-example/src/proto"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	ghs "golang.stackrox.io/grpc-http1/server"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedMathServer
}

func (s server) Max(srv pb.Math_MaxServer) error {

	log.Println("start new server")
	var max int32
	ctx := srv.Context()

	for {

		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// receive data from stream
		req, err := srv.Recv()
		if err == io.EOF {
			// return will close stream from server side
			log.Println("exit")
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}

		// continue if number reveived from stream
		// less than max
		if req.Num <= max {
			continue
		}

		// update max and send it to stream
		max = req.Num
		resp := pb.Response{Result: max}
		if err := srv.Send(&resp); err != nil {
			log.Printf("send error %v", err)
		}
		log.Printf("send new max=%d", max)
	}
}

func main() {
	useWebSocket := flag.Bool("useWebSocket", true, "gRPC over websocket")
	flag.Parse()

	// create listener
	lis, err := net.Listen("tcp", ":50005")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// create grpc server
	s := grpc.NewServer(grpc.StreamInterceptor(streamInterceptor))
	pb.RegisterMathServer(s, &server{})

	if *useWebSocket {
		downgradingSrv := &http.Server{}
		var h2Srv http2.Server
		_ = http2.ConfigureServer(downgradingSrv, &h2Srv)
		opts2 := []ghs.Option{ghs.PreferGRPCWeb(true)}
		downgradingSrv.Handler = h2c.NewHandler(ghs.CreateDowngradingHandler(s, http.NotFoundHandler(), opts2...), &h2Srv)
		downgradingSrv.Handler = authenticationMiddleware(downgradingSrv.Handler)
		downgradingSrv.Serve(lis)
	} else {
		// start regular gRPC server ...
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}
}

// websocket middleware to verify token
func authenticationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("## headers")
		for k, v := range r.Header {
			log.Printf("%s, %s\n", k, v)
		}
		token := r.Header.Get("authorization")
		log.Printf("## token: %s\n", token)
		if token != "abcd1234" {
			log.Println("Invalid token")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// gRPC stream interceptor to verify token in header
func streamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// authentication (token verification)
	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Errorf(codes.Unauthenticated, "Missing metadata")
	}
	authHeader, ok := md["authorization"]
	if !ok {
		log.Println("Authorization token is not supplied")
		return status.Errorf(codes.Unauthenticated, "Authorization token is not supplied")
	}

	token := authHeader[0]
	log.Printf("token: %s\n", token)
	// hardcode token here
	if token != "abcd1234" {
		log.Println("Invalid token")
		return status.Errorf(codes.Unauthenticated, "Invalid token")
	}
	return handler(srv, ss)
}
