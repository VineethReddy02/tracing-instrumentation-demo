package main

import (
	"github.com/VineethReddy02/tracing-inst-demo/demo-otel/lib/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"log"
	"net/http"

)

var Servicename = "publisher"

func main() {
	cfg := &tracing.Config{
		ServiceName:             "publisher",
		OtelCollectorEndpoint:   "localhost:4317",
		//JaegerCollectorEndpoint: "http://localhost:14268/api/traces",
		SamplingRatio:           1,
	}

	_, err := tracing.InitProvider(cfg)
	if err != nil {
		err.Error()
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		_, span := otel.Tracer(Servicename).Start(r.Context(), "publisher")
		defer span.End()

		helloStr := r.FormValue("helloStr")
		println(helloStr)
	}

	otelHandler := otelhttp.NewHandler(http.HandlerFunc(handler), "publish")

	log.Fatal(http.ListenAndServe(":8082", otelHandler))
}

