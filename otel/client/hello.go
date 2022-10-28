package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	xhttp "github.com/VineethReddy02/tracing-inst-demo/demo-otel/lib/http"
	"github.com/VineethReddy02/tracing-inst-demo/demo-otel/lib/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Errorf("ERROR: Expecting two arguments")
	}

	cfg := &tracing.Config{
		ServiceName:           "greeting-client",
		OtelCollectorEndpoint: "localhost:4317",
		//JaegerCollectorEndpoint: "http://localhost:14268/api/traces",
		SamplingRatio: 1,
	}

	tp, err := tracing.InitProvider(cfg)
	if err != nil {
		fmt.Errorf("%v", err)
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
		fmt.Println("E1")
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
		fmt.Errorf("%v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		span.SetStatus(codes.Error, "failed to reach formatter service")
		span.RecordError(err)
		fmt.Errorf("%v", err)
	}

	var formattedString []byte
	_, err = resp.Body.Read(formattedString)
	if err != nil {
		fmt.Errorf("failed to read the formatter response %v", err)
	}

	span.AddEvent("string-format")
	span.AddEvent("event", trace.WithAttributes(attribute.String("value", string(formattedString))))
	return string(formattedString)
}

func printHello(ctx context.Context, helloStr string) {
	ctx, span := tracing.Default().Start(ctx, "print-hello")
	defer span.End()

	v := url.Values{}
	v.Set("helloStr", helloStr)
	url := "http://localhost:8082/publish?" + v.Encode()
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		span.SetStatus(codes.Error, "failed to create a new publisher request")
		span.RecordError(err)
		fmt.Errorf("%v", err)
	}

	if _, err := xhttp.Do(req); err != nil {
		span.SetStatus(codes.Error, "failed to reach publisher service")
		span.RecordError(err)
		fmt.Errorf("%v", err)
	}
}
