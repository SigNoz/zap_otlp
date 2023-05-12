package zap_otlp_sync

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/instrumentation"
	"go.opentelemetry.io/otel/sdk/resource"
	collpb "go.opentelemetry.io/proto/otlp/collector/logs/v1"
	cv1 "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	rv1 "go.opentelemetry.io/proto/otlp/resource/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type OtelSyncer struct {
	ctx       context.Context
	close     context.CancelFunc
	res       *rv1.Resource
	resSchema string
	scope     *cv1.InstrumentationScope
	values    [][]byte
	queue     chan []byte

	client        collpb.LogsServiceClient
	batchSize     int
	sendBatch     chan bool
	closeExporter chan bool
	valueMutex    sync.Mutex
}

type Options struct {
	BatchSize          int
	BatchIntervalInSec int
	ResourceSchema     string
	Scope              *instrumentation.Scope
	Resource           *resource.Resource
}

func NewOtlpSyncer(conn *grpc.ClientConn, options Options) *OtelSyncer {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if options.BatchSize == 0 {
		options.BatchSize = 100
	}

	if options.BatchIntervalInSec == 0 {
		options.BatchIntervalInSec = 5
	}

	var rattrs *rv1.Resource
	if options.Resource != nil {
		iter := options.Resource.Iter()
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

	var instrumentationScope *cv1.InstrumentationScope
	if options.Scope != nil {
		instrumentationScope = &cv1.InstrumentationScope{
			Name:    options.Scope.Name,
			Version: options.Scope.Version,
		}
	}

	syncer := &OtelSyncer{
		ctx:           ctx,
		close:         cancel,
		res:           rattrs,
		resSchema:     options.ResourceSchema,
		scope:         instrumentationScope,
		values:        [][]byte{},
		queue:         make(chan []byte, options.BatchSize), // keeping queue size as batch size
		client:        collpb.NewLogsServiceClient(conn),
		batchSize:     options.BatchSize,
		sendBatch:     make(chan bool),
		closeExporter: make(chan bool),
		valueMutex:    sync.Mutex{},
	}

	go syncer.processQueue(options.BatchIntervalInSec)

	return syncer
}

func (l *OtelSyncer) processQueue(intervalInSec int) {
	ticker := time.NewTicker(time.Duration(intervalInSec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-l.closeExporter:
			fmt.Println("shutting down syncer batcher")
			return
		case <-ticker.C:
			l.pushData()
		case data := <-l.queue:
			l.valueMutex.Lock()
			l.values = append(l.values, data)
			shouldExport := len(l.values) >= l.batchSize
			l.valueMutex.Unlock()
			if shouldExport {
				l.pushData()
			}
		case <-l.sendBatch:
			l.pushData()
		}
	}
}

func (l *OtelSyncer) pushData() (err error) {
	rec := []*lpb.LogRecord{}

	l.valueMutex.Lock()
	for _, v := range l.values {
		r := &lpb.LogRecord{}
		err = proto.Unmarshal([]byte(v), r)
		if err != nil {
			l.valueMutex.Unlock()
			return err
		}
		rec = append(rec, r)
	}
	l.values = [][]byte{} // clean the values
	l.valueMutex.Unlock()

	if len(rec) == 0 {
		return nil
	}

	sl := &lpb.ScopeLogs{LogRecords: rec}
	if l.scope != nil {
		sl.SchemaUrl, sl.Scope = l.resSchema, l.scope
	}

	rl := &lpb.ResourceLogs{ScopeLogs: []*lpb.ScopeLogs{sl}}
	if l.res != nil {
		rl.SchemaUrl, rl.Resource = l.resSchema, l.res
	}
	_, err = l.client.Export(context.Background(), &collpb.ExportLogsServiceRequest{
		ResourceLogs: []*lpb.ResourceLogs{rl},
	})
	// TODO: how to handle partial failure and error
	if err != nil {
		return err
	}

	return nil
}

func (l *OtelSyncer) Write(record []byte) (n int, err error) {
	// create deep copyy of the data
	data := make([]byte, len(record))
	copy(data, record)

	// just add the data to the queue
	l.queue <- data
	return len(data), nil
}

func (l *OtelSyncer) Sync() error {
	l.sendBatch <- true
	return nil
}

func (l *OtelSyncer) Close() error {
	err := l.Sync()
	if err != nil {
		return err
	}
	l.closeExporter <- true
	l.close()
	return nil
}
