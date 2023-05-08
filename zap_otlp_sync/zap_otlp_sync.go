package zap_otlp_sync

import (
	"context"
	"encoding/json"

	collpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/grpc"
)

type ResourceAttributes map[string]interface{}

func NewOtlpSyncer(conn *grpc.ClientConn, resourceAttrs ResourceAttributes) *OtelSyncer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	return &OtelSyncer{
		ctx,
		cancel,
		resourceAttrs,
		[][]byte{},
		collpb.NewLogsServiceClient(conn),
	}
}

type OtelSyncer struct {
	ctx                context.Context
	close              context.CancelFunc
	ResourceAttributes ResourceAttributes
	values             [][]byte

	client collpb.LogsServiceClient
}

func (l OtelSyncer) pushToSigNoz() (err error) {
	rec := []*lpb.LogRecord{}

	for _, v := range l.values {
		// not assigning the empty body structs throws an error
		r := &lpb.LogRecord{Body: &v1.AnyValue{Value: &v1.AnyValue_StringValue{}}}
		err := json.Unmarshal(v, &r)
		if err != nil {
			return err
		}
		rec = append(rec, r)
	}
	// recently adding for otlp
	sl := &lpb.ScopeLogs{LogRecords: rec}
	// if l.scope != nil {
	// 	sl.SchemaUrl, sl.Scope = l.scopeSchema, l.scope
	// }

	rl := &lpb.ResourceLogs{ScopeLogs: []*lpb.ScopeLogs{sl}}
	// if l.res != nil {
	// 	rl.SchemaUrl, rl.Resource = l.resSchema, l.res
	// }
	_, _ = l.client.Export(context.Background(), &collpb.ExportLogsServiceRequest{
		ResourceLogs: []*lpb.ResourceLogs{rl},
	})

	return nil
}

func (l *OtelSyncer) Write(p []byte) (n int, err error) {
	l.values = append(l.values, p)
	return len(p), nil
}

func (l OtelSyncer) Sync() error {
	if err := l.pushToSigNoz(); err != nil {
		return err
	}

	return nil
}

func (l OtelSyncer) Close() error {
	l.close()
	return nil
}
