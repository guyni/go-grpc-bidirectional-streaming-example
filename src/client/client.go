package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"io"
	"log"
	"math/rand"
	"net/url"

	pb "github.com/pahanini/go-grpc-bidirectional-streaming-example/src/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"golang.stackrox.io/grpc-http1/client"

	"time"
)

func main() {
	// if useWebSocket is false, use plain old gRPC client
	useWebSocket := flag.Bool("useWebSocket", true, "gRPC over websocket")
	useTLS := flag.Bool("useTLS", true, "use TLS")
	target := flag.String("target", ":50005", "gRPC target URI")
	token := flag.String("token", "", "authentication token")
	flag.Parse()

	rand.Seed(time.Now().Unix())
	var tlsConfig *tls.Config
	var cred credentials.TransportCredentials

	if *useTLS {
		tlsConfig = &tls.Config{
			InsecureSkipVerify:    true,
			VerifyPeerCertificate: printPeerCertificate,
		}
		cred = credentials.NewTLS(tlsConfig)
	} else {
		cred = insecure.NewCredentials()
	}
	var conn *grpc.ClientConn
	var err error
	if *useWebSocket {
		opts := []client.ConnectOption{}
		// use insecure.NewCredentials() here since in memory proxy is non tls
		opts = append(opts, client.DialOpts(grpc.WithTransportCredentials(insecure.NewCredentials())))
		opts = append(opts, client.UseWebSocket(true))
		opts = append(opts, client.UrlRewrite(addPrefix))
		// tlsConfig is for target gRPC server and opts is for local in memory proxy
		conn, err = client.ConnectViaProxy(context.TODO(), *target, tlsConfig, opts...)
	} else {
		// dail server directly
		conn, err = grpc.NewClient(*target, grpc.WithTransportCredentials(cred))
	}

	if err != nil {
		log.Fatalf("can not connect with server %v", err)
	}

	// create stream
	clientCtx := context.Background()
	if *token != "" {
		// set token to header
		md := metadata.Pairs("authorization", *token)
		clientCtx = metadata.NewOutgoingContext(clientCtx, md)
	}

	client := pb.NewMathClient(conn)
	stream, err := client.Max(clientCtx)
	if err != nil {
		log.Fatalf("openn stream error %v", err)
	}

	var max int32
	ctx := stream.Context()
	done := make(chan bool)

	// first goroutine sends random increasing numbers to stream
	// and closes it after 10 iterations
	go func() {
		for i := 1; i <= 10; i++ {
			// generates random number and sends it to stream
			rnd := int32(rand.Intn(i))
			req := pb.Request{Num: rnd}
			if err := stream.Send(&req); err != nil {
				log.Fatalf("can not send %v", err)
			}
			log.Printf("%d sent", req.Num)
			time.Sleep(time.Millisecond * 200)
		}
		if err := stream.CloseSend(); err != nil {
			log.Println(err)
		}
	}()

	// second goroutine receives data from stream
	// and saves result in max variable
	//
	// if stream is finished it closes done channel
	go func() {
		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				close(done)
				return
			}
			if err != nil {
				log.Fatalf("can not receive %v", err)
			}
			max = resp.Result
			log.Printf("new max %d received", max)
		}
	}()

	// third goroutine closes done channel
	// if context is done
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		close(done)
	}()

	<-done
	log.Printf("finished with max=%d", max)
}

func printPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	for i, item := range rawCerts {
		if cert, err := x509.ParseCertificate(item); err == nil {
			log.Printf("cert %d: subject name is : %s\n", i, cert.Subject.CommonName)
			log.Printf("subject: %v\n", cert.Subject)
			log.Printf("issuer: %v\n", cert.Issuer)
		} else {
			log.Println(err)
		}
	}
	return nil
}

func addPrefix(u *url.URL) *url.URL {
	u.Path = "/grpc-ws" + u.Path
	return u
}
