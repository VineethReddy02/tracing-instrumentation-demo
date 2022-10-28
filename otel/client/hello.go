package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/VineethReddy02/tracing-inst-demo/demo-otel/lib/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("ERROR: Expecting two arguments")
	}

	cfg := &tracing.Config{
		ServiceName:           "greeting-client",
		OtelCollectorEndpoint: "localhost:4317",
		//JaegerCollectorEndpoint: "http://localhost:14268/api/traces",
		SamplingRatio: 1,
	}

	tp, err := tracing.InitProvider(cfg)
	if err != nil {
		log.Fatalf("%v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Cleanly shutdown and flush telemetry when the application exits.
	defer func(ctx context.Context) {
		// Do not make the application hang when it is shutdown.
		ctx, cancel = context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}(ctx)

	greeting := os.Args[1]
	helloTo := os.Args[2]

	spanctx, span := tracing.Default().Start(context.Background(), "say-hello")
	defer span.End()

	span.SetAttributes(attribute.String("hello-to", helloTo))
	fmt.Println(span.SpanContext().SpanID())

	bag, err := baggage.Parse("greeting=" + greeting)
	if err != nil {
		log.Fatal(err)
	}
	bctx := baggage.ContextWithBaggage(spanctx, bag)
	helloStr := formatString(bctx, helloTo)
	printHello(bctx, helloStr)
}

func formatString(ctx context.Context, helloTo string) string {
	spanctx, span := tracing.Default().Start(ctx, "format-string")
	defer span.End()

	v := url.Values{}
	v.Set("helloTo", helloTo)
	url := "http://localhost:8081/format?" + v.Encode()
	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, err := http.NewRequestWithContext(spanctx, "GET", url, nil)
	if err != nil {
		span.SetStatus(codes.Error, "failed to create formatter request")
		span.RecordError(err)
		log.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, "failed to reach formatter service")
		span.RecordError(err)
		log.Fatal(err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	span.AddEvent("string-format")
	span.AddEvent("event", trace.WithAttributes(attribute.String("value", string(bodyBytes))))
	return string(bodyBytes)
}

func printHello(ctx context.Context, helloStr string) {
	spanctx, span := tracing.Default().Start(ctx, "print-hello")
	defer span.End()

	v := url.Values{}
	v.Set("helloStr", helloStr)
	url := "http://localhost:8082/publish?" + v.Encode()
	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	req, err := http.NewRequestWithContext(spanctx, "GET", url, nil)
	if err != nil {
		span.SetStatus(codes.Error, "failed to create a new publisher request")
		span.RecordError(err)
		log.Fatal(err)
	}

	if _, err := client.Do(req); err != nil {
		span.SetStatus(codes.Error, "failed to reach publisher service")
		span.RecordError(err)
		log.Fatal(err)
	}
}
