package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/VineethReddy02/tracing-inst-demo/demo-otel/lib/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

var ServiceName = "formatter"

func main() {
	cfg := &tracing.Config{
		ServiceName:           "formatter",
		OtelCollectorEndpoint: "localhost:4317",
		//JaegerCollectorEndpoint: "http://localhost:14268/api/traces",
		SamplingRatio: 1,
	}

	_, err := tracing.InitProvider(cfg)
	if err != nil {
		log.Fatal(err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		span := trace.SpanFromContext(ctx)
		bag := baggage.FromContext(ctx)

		_, spanFormat := otel.Tracer(ServiceName).Start(ctx, "format")
		defer spanFormat.End()

		greeting := bag.Member("greeting").Value()
		if greeting == "" {
			greeting = "Hello"
		}

		helloTo := r.FormValue("helloTo")
		helloStr := fmt.Sprintf("%s, %s!", greeting, helloTo)
		fmt.Println("Formatted string: " + helloStr)
		span.AddEvent("string-format")
		span.AddEvent(helloStr)
		w.Write([]byte(helloStr))
	}

	otelHandler := otelhttp.NewHandler(http.HandlerFunc(handler), "format")

	log.Fatal(http.ListenAndServe(":8081", otelHandler))
}
