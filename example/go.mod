module github.com/SigNoz/zap_otlp/example

go 1.19

replace (
	github.com/SigNoz/zap_otlp => ../
	github.com/SigNoz/zap_otlp/zap_otlp_encoder => ../zap_otlp_encoder
	github.com/SigNoz/zap_otlp/zap_otlp_sync => ../zap_otlp_sync
)

require (
	github.com/SigNoz/zap_otlp v0.0.0-00010101000000-000000000000
	github.com/SigNoz/zap_otlp/zap_otlp_encoder v0.0.0-00010101000000-000000000000
	github.com/SigNoz/zap_otlp/zap_otlp_sync v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel v1.15.1
	go.opentelemetry.io/otel/sdk v1.15.1
	go.opentelemetry.io/otel/trace v1.15.1
	go.uber.org/zap v1.24.0
	google.golang.org/grpc v1.55.0
)

require (
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/goleak v1.2.1 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	golang.org/x/net v0.8.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/text v0.8.0 // indirect
	google.golang.org/genproto v0.0.0-20230306155012-7f2fa6fef1f4 // indirect
	google.golang.org/protobuf v1.30.0 // indirect
)
