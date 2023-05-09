package zap_otlp_sync

import (
	"context"

	"go.opentelemetry.io/otel/sdk/resource"
	collpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	cv1 "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	rv1 "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func NewOtlpSyncer(conn *grpc.ClientConn, batchSize int, resSchema string, res *resource.Resource) *OtelSyncer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if batchSize == 0 {
		batchSize = 100
	}

	var rattrs *rv1.Resource
	if res != nil {
		iter := res.Iter()
		attrs := make([]*cv1.KeyValue, 0, iter.Len())
		for iter.Next() {
			attr := iter.Attribute()
			attrs = append(attrs, &cv1.KeyValue{
				Key:   string(attr.Key),
				Value: &cv1.AnyValue{Value: &cv1.AnyValue_StringValue{StringValue: attr.Value.Emit()}},
			})
		}
		rattrs = &rv1.Resource{Attributes: attrs}
	}

	return &OtelSyncer{
		ctx,
		cancel,
		rattrs,
		resSchema,
		[][]byte{},
		collpb.NewLogsServiceClient(conn),
		batchSize,
	}
}

type OtelSyncer struct {
	ctx       context.Context
	close     context.CancelFunc
	res       *rv1.Resource
	resSchema string
	values    [][]byte

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
	if l.res != nil {
		rl.SchemaUrl, rl.Resource = l.resSchema, l.res
	}
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
