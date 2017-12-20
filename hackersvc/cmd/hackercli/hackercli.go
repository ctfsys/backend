package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"google.golang.org/grpc"

	"github.com/apache/thrift/lib/go/thrift"
	lightstep "github.com/lightstep/lightstep-tracer-go"
	stdopentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"sourcegraph.com/sourcegraph/appdash"
	appdashot "sourcegraph.com/sourcegraph/appdash/opentracing"

	"github.com/go-kit/kit/log"

	"github.com/ctfsys/backend/hackersvc/pkg/hackerservice"
	"github.com/ctfsys/backend/hackersvc/pkg/hackertransport"
	hackerthrift "github.com/ctfsys/backend/hackersvc/thrift/gen-go/hackersvc"
)

func main() {
	fs := flag.NewFlagSet("hackercli", flag.ExitOnError)
	var (
		httpAddr       = fs.String("http-addr", "", "HTTP address of hackersvc")
		grpcAddr       = fs.String("grpc-addr", "", "gRPC address of hackersvc")
		thriftAddr     = fs.String("thrift-addr", "", "Thrift address of hackersvc")
		thriftProtocol = fs.String("thrift-protocol", "binary", "binary, compact, json, simplejson")
		thriftBuffer   = fs.Int("thrift-buffer", 0, "0 for unbuffered")
		thriftFramed   = fs.Bool("thrift-framed", false, "true to enable framing")
		zipkinURL      = fs.String("zipkin-url", "", "Enable Zipkin tracing via a collector URL e.g. http://localhost:9411/api/v1/spans")
		lightstepToken = flag.String("lightstep-token", "", "Enable LightStep tracing via a LightStep access token")
		appdashAddr    = flag.String("appdash-addr", "", "Enable Appdash tracing via an Appdash server host:port")
		method         = fs.String("method", "ping", "ping")
	)
	fs.Usage = usageFor(fs, os.Args[0]+" [flags] [<arg>...]")
	fs.Parse(os.Args[1:])
	// if len(fs.Args()) != 1 {
	// 	fs.Usage()
	// 	os.Exit(1)
	// }

	// We test different tracers here, but we should pick one and stick with
	// it.
	var tracer stdopentracing.Tracer
	{
		if *zipkinURL != "" {
			collector, err := zipkin.NewHTTPCollector(*zipkinURL)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
			defer collector.Close()

			var (
				debug       = false
				hostPort    = "localhost:80"
				serviceName = "hackersvc"
			)
			recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)
			tracer, err = zipkin.NewTracer(recorder)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		} else if *lightstepToken != "" {
			tracer = lightstep.NewTracer(lightstep.Options{
				AccessToken: *lightstepToken,
			})
			defer lightstep.FlushLightStepTracer(tracer)
		} else if *appdashAddr != "" {
			tracer = appdashot.NewTracer(appdash.NewRemoteCollector(*appdashAddr))
		} else {
			tracer = stdopentracing.GlobalTracer() // no-op
		}
	}

	// Again, we try out a couple of transports here, but we'll probably just
	// stick with one in the end.
	var (
		svc hackerservice.Service
		err error
	)
	if *httpAddr != "" {
		svc, err = hackertransport.NewHTTPClient(*httpAddr, tracer, log.NewNopLogger())
	} else if *grpcAddr != "" {
		conn, err := grpc.Dial(*grpcAddr, grpc.WithInsecure(), grpc.WithTimeout(time.Second))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v", err)
			os.Exit(1)
		}
		defer conn.Close()
		svc = hackertransport.NewGRPCClient(conn, tracer, log.NewNopLogger())
	} else if *thriftAddr != "" {
		var protocolFactory thrift.TProtocolFactory
		switch *thriftProtocol {
		case "compact":
			protocolFactory = thrift.NewTCompactProtocolFactory()
		case "simplejson":
			protocolFactory = thrift.NewTSimpleJSONProtocolFactory()
		case "json":
			protocolFactory = thrift.NewTJSONProtocolFactory()
		case "binary", "":
			protocolFactory = thrift.NewTBinaryProtocolFactoryDefault()
		default:
			fmt.Fprintf(os.Stderr, "error: invalid protocol %q\n", *thriftProtocol)
			os.Exit(1)
		}
		var transportFactory thrift.TTransportFactory
		if *thriftBuffer > 0 {
			transportFactory = thrift.NewTBufferedTransportFactory(*thriftBuffer)
		} else {
			transportFactory = thrift.NewTTransportFactory()
		}
		if *thriftFramed {
			transportFactory = thrift.NewTFramedTransportFactory(transportFactory)
		}
		transportSocker, err := thrift.NewTSocket(*thriftAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		transport, err := transportFactory.GetTransport(transportSocker)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := transport.Open(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		defer transport.Close()
		client := hackerthrift.NewHackerServiceClientFactory(transport, protocolFactory)
		svc = hackertransport.NewThriftClient(client)
	} else {
		fmt.Fprintf(os.Stderr, "error: no remote address specified\n")
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch *method {
	case "ping":
		v, err := svc.Ping(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "ping = %v", v)
	default:
		fmt.Fprintf(os.Stderr, "error: invalid method %q\n", method)
		os.Exit(1)
	}
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, " %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}
