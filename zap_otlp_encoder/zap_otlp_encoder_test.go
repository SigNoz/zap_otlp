package zap_otlp_encoder

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/SigNoz/zap_otlp"
	. "github.com/smartystreets/goconvey/convey"
	sdk "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	cv1 "go.opentelemetry.io/proto/otlp/common/v1"
	lv1 "go.opentelemetry.io/proto/otlp/logs/v1"
	"google.golang.org/protobuf/proto"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	lib    = "github.com/SigNoz/zap_otlp/example"
	libVer = "v0.1.0"
)

func TestOTLPEncodeEntry(t *testing.T) {

	// ctx for trace and span
	tp := sdk.NewTracerProvider()
	tracer := tp.Tracer(lib, trace.WithInstrumentationVersion(libVer))
	ctx := context.Background()
	var span trace.Span
	ctx, span = tracer.Start(ctx, "main")
	defer span.End()
	sid := span.SpanContext().SpanID()
	tid := span.SpanContext().TraceID()

	type bar struct {
		Key string  `json:"key"`
		Val float64 `json:"val"`
	}

	type foo struct {
		A string  `json:"aee"`
		B int     `json:"bee"`
		C float64 `json:"cee"`
		D []bar   `json:"dee"`
	}

	tests := []struct {
		name     string
		desc     string
		expected *lv1.LogRecord
		ent      zapcore.Entry
		fields   []zapcore.Field
	}{
		{
			name: "Test 1",
			desc: "entry with just body",
			expected: &lv1.LogRecord{
				TimeUnixNano:   1529426022000000099,
				SeverityNumber: lv1.SeverityNumber_SEVERITY_NUMBER_INFO,
				SeverityText:   lv1.SeverityNumber_name[int32(lv1.SeverityNumber_SEVERITY_NUMBER_INFO)],
				Body:           &cv1.AnyValue{Value: &cv1.AnyValue_StringValue{StringValue: "lob law"}},
			},
			ent: zapcore.Entry{
				Level:   zapcore.InfoLevel,
				Time:    time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
				Message: "lob law",
			},
		},
		{
			name: "Test 2",
			desc: "Add logger name",
			expected: &lv1.LogRecord{
				TimeUnixNano:   1529426022000000099,
				SeverityNumber: lv1.SeverityNumber_SEVERITY_NUMBER_INFO,
				SeverityText:   lv1.SeverityNumber_name[int32(lv1.SeverityNumber_SEVERITY_NUMBER_INFO)],
				Body:           &cv1.AnyValue{Value: &cv1.AnyValue_StringValue{StringValue: "lob law"}},
			},
			ent: zapcore.Entry{
				Level:      zapcore.InfoLevel,
				Time:       time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
				LoggerName: "bob",
				Message:    "lob law",
			},
		},
		{
			name: "Test 3",
			desc: "info entry with some fields",
			expected: &lv1.LogRecord{
				TimeUnixNano:   1529426022000000099,
				SeverityNumber: lv1.SeverityNumber_SEVERITY_NUMBER_INFO,
				SeverityText:   lv1.SeverityNumber_name[int32(lv1.SeverityNumber_SEVERITY_NUMBER_INFO)],
				Body:           &cv1.AnyValue{Value: &cv1.AnyValue_StringValue{StringValue: "lob law"}},
				Attributes: []*cv1.KeyValue{
					{Key: "so", Value: &cv1.AnyValue{Value: &cv1.AnyValue_StringValue{StringValue: "passes"}}},
					{Key: "answer", Value: &cv1.AnyValue{Value: &cv1.AnyValue_IntValue{IntValue: 42}}},
					{Key: "common_pie", Value: &cv1.AnyValue{Value: &cv1.AnyValue_DoubleValue{DoubleValue: 3.14}}},
					{Key: "a_float32", Value: &cv1.AnyValue{Value: &cv1.AnyValue_DoubleValue{DoubleValue: 2.7100000381469727}}},

					// TODO: complex/array/object support to be added
					// {Key: "complex_value", Value: &cv1.AnyValue{Value: &cv1.AnyValue_StringValue{StringValue: "3.14-2.71i"}}},
				},
			},
			ent: zapcore.Entry{
				Level:      zapcore.InfoLevel,
				Time:       time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
				LoggerName: "bob",
				Message:    "lob law",
			},
			fields: []zapcore.Field{
				zap.String("so", "passes"),
				zap.Int("answer", 42),
				zap.Float64("common_pie", 3.14),
				// zap.Int32("xyz", 1),
				zap.Float32("a_float32", 2.71),
				zap.Complex128("complex_value", 3.14-2.71i), // currently ignored by the encoder
				zap.Reflect("such", foo{ // currently ignored by the encoder
					A: "lol",
					B: 123,
					C: 0.9999,
					D: []bar{
						{"pi", 3.141592653589793},
						{"tau", 6.283185307179586},
					},
				}),
			},
		},
		{
			name: "Test 4",
			desc: "Log with traceId and spanId",
			expected: &lv1.LogRecord{
				TimeUnixNano:   1529426022000000099,
				SeverityNumber: lv1.SeverityNumber_SEVERITY_NUMBER_INFO,
				SeverityText:   lv1.SeverityNumber_name[int32(lv1.SeverityNumber_SEVERITY_NUMBER_INFO)],
				Body:           &cv1.AnyValue{Value: &cv1.AnyValue_StringValue{StringValue: "lob law"}},
				TraceId:        tid[:],
				SpanId:         sid[:],
			},
			ent: zapcore.Entry{
				Level:      zapcore.InfoLevel,
				Time:       time.Date(2018, 6, 19, 16, 33, 42, 99, time.UTC),
				LoggerName: "bob",
				Message:    "lob law",
			},
			fields: []zapcore.Field{
				zap_otlp.SpanCtx(ctx),
			},
		},
	}

	enc := NewOTLPEncoder(zap.NewProductionEncoderConfig())

	for _, tt := range tests {
		Convey(tt.name, t, func() {
			buf, err := enc.EncodeEntry(tt.ent, tt.fields)
			So(err, ShouldBeNil)

			data := strings.Split(string(buf.Bytes()), "#SIGNOZ#")

			// For debugging purpose uncomment the lines below
			// r := &lv1.LogRecord{}
			// err = proto.Unmarshal([]byte(data[1]), r)
			// So(err, ShouldBeNil)
			// fmt.Println(r)
			// fmt.Println(tt.expected)

			d, err := proto.Marshal(tt.expected)
			So(err, ShouldBeNil)
			So(d, ShouldResemble, []byte(data[1]))
			buf.Free()
		})
	}
}
