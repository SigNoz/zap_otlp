module github.com/SigNoz/zap_otlp/zap_otlp_encoder

go 1.19

replace github.com/SigNoz/zap_otlp => ../

require (
	github.com/SigNoz/zap_otlp v0.1.0
	github.com/smartystreets/goconvey v1.8.1
	go.opentelemetry.io/otel/sdk v1.16.0
	go.opentelemetry.io/otel/trace v1.16.0
	go.opentelemetry.io/proto/otlp v1.0.0
	go.uber.org/zap v1.27.0
	google.golang.org/protobuf v1.33.0
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/smarty/assertions v1.15.0 // indirect
	go.opentelemetry.io/otel v1.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.8.0 // indirect
)
