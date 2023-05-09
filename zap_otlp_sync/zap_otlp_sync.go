package zap_otlp_sync

import (
	"context"

	collpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type ResourceAttributes map[string]interface{}

func NewOtlpSyncer(conn *grpc.ClientConn, batchSize int, resourceAttrs ResourceAttributes) *OtelSyncer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if batchSize == 0 {
		batchSize = 100
	}

	return &OtelSyncer{
		ctx,
		cancel,
		resourceAttrs,
		[][]byte{},
		collpb.NewLogsServiceClient(conn),
		batchSize,
	}
}

type OtelSyncer struct {
	ctx                context.Context
	close              context.CancelFunc
	ResourceAttributes ResourceAttributes
	values             [][]byte

	client    collpb.LogsServiceClient
	batchSize int
}

func (l OtelSyncer) pushToSigNoz() (err error) {
	rec := []*lpb.LogRecord{}

	for _, v := range l.values {
		r := &lpb.LogRecord{}
		err = proto.Unmarshal([]byte(v), r)
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

func (l *OtelSyncer) Write(record []byte) (n int, err error) {
	// create deep copyy of the data
	data := make([]byte, len(record))
	copy(data, record)
	l.values = append(l.values, data)
	if len(l.values) >= l.batchSize {
		err := l.Sync()
		if err != nil {
			return 0, err
		}
		l.values = [][]byte{}
	}
	return len(record), nil
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
