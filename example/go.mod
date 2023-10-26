module github.com/SigNoz/zap_otlp/example

go 1.19

replace (
	github.com/SigNoz/zap_otlp => ../
	github.com/SigNoz/zap_otlp/zap_otlp_encoder => ../zap_otlp_encoder
	github.com/SigNoz/zap_otlp/zap_otlp_sync => ../zap_otlp_sync
)

require (
	github.com/SigNoz/zap_otlp v0.1.0
	github.com/SigNoz/zap_otlp/zap_otlp_encoder v0.0.0-00010101000000-000000000000
	github.com/SigNoz/zap_otlp/zap_otlp_sync v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.16.0
	go.opentelemetry.io/otel/sdk v1.16.0
	go.opentelemetry.io/otel/trace v1.16.0
	go.uber.org/zap v1.25.0
	google.golang.org/grpc v1.57.1
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0 // indirect
	go.opentelemetry.io/otel/metric v1.16.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	go.uber.org/goleak v1.2.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230530153820-e85fd2cbaebc // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
