package zap_otlp_encoder

// adapted from https://github.com/uber-go/zap/blob/master/zapcore/json_encoder.go
// and https://github.com/uber-go/zap/blob/master/zapcore/console_encoder.go

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"sync"
	"time"
	"unicode/utf8"

	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	lpb "go.opentelemetry.io/proto/otlp/logs/v1"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

const _hex = "0123456789abcdef"

var bufferPool = buffer.NewPool()

var _otlpPool = sync.Pool{New: func() interface{} {
	return &otlpEncoder{}
}}

func getOTLPEncoder() *otlpEncoder {
	return _otlpPool.Get().(*otlpEncoder)
}

func putOTLPEncoder(enc *otlpEncoder) {
	enc.EncoderConfig = nil
	enc.buf = nil
	enc.spaced = false
	enc.openNamespaces = 0
	enc.reflectBuf = nil
	enc.reflectEnc = nil
	_otlpPool.Put(enc)
}

type otlpEncoder struct {
	*zapcore.EncoderConfig
	buf            *buffer.Buffer
	spaced         bool
	openNamespaces int

	log lpb.LogRecord

	// for encoding generic values by reflection
	reflectBuf *buffer.Buffer
	reflectEnc zapcore.ReflectedEncoder
}

// NewkvEncoder creates a key=value encoder
func NewOTLPEncoder(cfg zapcore.EncoderConfig) zapcore.Encoder {
	if cfg.SkipLineEnding {
		cfg.LineEnding = ""
	} else if cfg.LineEnding == "" {
		cfg.LineEnding = zapcore.DefaultLineEnding
	}

	// If no EncoderConfig.NewReflectedEncoder is provided by the user, then use default
	if cfg.NewReflectedEncoder == nil {
		cfg.NewReflectedEncoder = defaultReflectedEncoder
	}

	return &otlpEncoder{
		EncoderConfig: &cfg,
		buf:           bufferPool.Get(),
		log:           lpb.LogRecord{},
	}
}

func (enc *otlpEncoder) AddArray(key string, arr zapcore.ArrayMarshaler) error {
	// enc.addKey(key)
	return enc.AppendArray(arr)
}

func (enc *otlpEncoder) AddObject(key string, obj zapcore.ObjectMarshaler) error {
	// enc.addKey(key)
	return enc.AppendObject(obj)
}

func (enc *otlpEncoder) AddBinary(key string, val []byte) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: base64.StdEncoding.EncodeToString(val)}})
	// enc.AddString(key, base64.StdEncoding.EncodeToString(val))
}

func (enc *otlpEncoder) AddByteString(key string, val []byte) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: string(val)}})
	// enc.addKey(key)
	// enc.AppendByteString(val)
}

func (enc *otlpEncoder) AddBool(key string, val bool) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_BoolValue{BoolValue: val}})
	// enc.addKey(key)
	// enc.AppendBool(val)
}

func (enc *otlpEncoder) AddComplex128(key string, val complex128) {
	// enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: }})
	// enc.addKey(key)
	// enc.AppendComplex128(val)
}

func (enc *otlpEncoder) AddComplex64(key string, val complex64) {
	// enc.addKey(key)
	// enc.AppendComplex64(val)
}

func (enc *otlpEncoder) AddDuration(key string, val time.Duration) {
	// enc.addKey(key)
	// enc.AppendDuration(val)
}

func (enc *otlpEncoder) AddFloat64(key string, val float64) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: val}})
	// enc.addKey(key)
	// enc.AppendFloat64(val)
}

func (enc *otlpEncoder) AddFloat32(key string, val float32) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_DoubleValue{DoubleValue: float64(val)}})
	// enc.addKey(key)
	// enc.AppendFloat32(val)
}

func (enc *otlpEncoder) AddInt64(key string, val int64) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_IntValue{IntValue: val}})
	// enc.addKey(key)
	// enc.AppendInt64(val)
}

func (enc *otlpEncoder) resetReflectBuf() {
	if enc.reflectBuf == nil {
		enc.reflectBuf = bufferPool.Get()
		enc.reflectEnc = enc.NewReflectedEncoder(enc.reflectBuf)
	} else {
		enc.reflectBuf.Reset()
	}
}

var nullLiteralBytes = []byte("null")

// Only invoke the standard JSON encoder if there is actually something to
// encode; otherwise write JSON null literal directly.
func (enc *otlpEncoder) encodeReflected(obj interface{}) ([]byte, error) {
	if obj == nil {
		return nullLiteralBytes, nil
	}
	enc.resetReflectBuf()
	if err := enc.reflectEnc.Encode(obj); err != nil {
		return nil, err
	}
	enc.reflectBuf.TrimNewline()
	return enc.reflectBuf.Bytes(), nil
}

func (enc *otlpEncoder) AddReflected(key string, obj interface{}) error {
	// valueBytes, err := enc.encodeReflected(obj)
	// if err != nil {
	// 	return err
	// }
	// enc.addKeyVal(key)
	// _, err = enc.buf.Write(valueBytes)
	return nil
}

func (enc *otlpEncoder) OpenNamespace(key string) {
	// enc.addKey(key)
	// enc.buf.AppendByte('{')
	// enc.openNamespaces++
}

func (enc *otlpEncoder) AddString(key, val string) {
	enc.addKeyVal(key, &v1.AnyValue{Value: &v1.AnyValue_StringValue{StringValue: val}})
	// enc.addKey(key)
	// enc.AppendString(val)
}

func (enc *otlpEncoder) AddTime(key string, val time.Time) {
	// enc.addKey("timestamp")
	// enc.AppendTime(val)
}

func (enc *otlpEncoder) AddUint64(key string, val uint64) {
	// enc.addKey(key)
	// enc.AppendUint64(val)
}

func (enc *otlpEncoder) AppendArray(arr zapcore.ArrayMarshaler) error {
	// enc.addElementSeparator()
	// enc.buf.AppendByte('[')
	// err := arr.MarshalLogArray(enc)
	// enc.buf.AppendByte(']')
	return nil
}

func (enc *otlpEncoder) AppendObject(obj zapcore.ObjectMarshaler) error {
	// Close ONLY new openNamespaces that are created during
	// AppendObject().
	// old := enc.openNamespaces
	// enc.openNamespaces = 0
	// enc.addElementSeparator()
	// enc.buf.AppendByte('{')
	// err := obj.MarshalLogObject(enc)
	// enc.buf.AppendByte('}')
	// enc.closeOpenNamespaces()
	// enc.openNamespaces = old
	// return err
	return nil
}

func (enc *otlpEncoder) AppendBool(val bool) {
	enc.addElementSeparator()
	enc.buf.AppendBool(val)
}

func (enc *otlpEncoder) AppendByteString(val []byte) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddByteString(val)
	enc.buf.AppendByte('"')
}

// appendComplex appends the encoded form of the provided complex128 value.
// precision specifies the encoding precision for the real and imaginary
// components of the complex number.
func (enc *otlpEncoder) appendComplex(val complex128, precision int) {
	enc.addElementSeparator()
	// Cast to a platform-independent, fixed-size type.
	r, i := float64(real(val)), float64(imag(val))
	enc.buf.AppendByte('"')
	// Because we're always in a quoted string, we can use strconv without
	// special-casing NaN and +/-Inf.
	enc.buf.AppendFloat(r, precision)
	// If imaginary part is less than 0, minus (-) sign is added by default
	// by AppendFloat.
	if i >= 0 {
		enc.buf.AppendByte('+')
	}
	enc.buf.AppendFloat(i, precision)
	enc.buf.AppendByte('i')
	enc.buf.AppendByte('"')
}

func (enc *otlpEncoder) AppendDuration(val time.Duration) {
	cur := enc.buf.Len()
	if e := enc.EncodeDuration; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeDuration is a no-op. Fall back to nanoseconds to keep
		// JSON valid.
		enc.AppendInt64(int64(val))
	}
}

func (enc *otlpEncoder) AppendInt64(val int64) {
	enc.addElementSeparator()
	enc.buf.AppendInt(val)
}

func (enc *otlpEncoder) AppendReflected(val interface{}) error {
	valueBytes, err := enc.encodeReflected(val)
	if err != nil {
		return err
	}
	enc.addElementSeparator()
	_, err = enc.buf.Write(valueBytes)
	return err
}

func (enc *otlpEncoder) AppendString(val string) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.safeAddString(val)
	enc.buf.AppendByte('"')
}

func (enc *otlpEncoder) AppendTimeLayout(time time.Time, layout string) {
	enc.addElementSeparator()
	enc.buf.AppendByte('"')
	enc.buf.AppendTime(time, layout)
	enc.buf.AppendByte('"')
}

func (enc *otlpEncoder) AppendTime(val time.Time) {
	cur := enc.buf.Len()
	if e := enc.EncodeTime; e != nil {
		e(val, enc)
	}
	if cur == enc.buf.Len() {
		// User-supplied EncodeTime is a no-op. Fall back to nanos since epoch to keep
		// output JSON valid.
		enc.AppendInt64(val.UnixNano())
	}
}

func (enc *otlpEncoder) AppendUint64(val uint64) {
	enc.addElementSeparator()
	enc.buf.AppendUint(val)
}

func (enc *otlpEncoder) AddInt(k string, v int)         { enc.AddInt64(k, int64(v)) }
func (enc *otlpEncoder) AddInt32(k string, v int32)     { enc.AddInt64(k, int64(v)) }
func (enc *otlpEncoder) AddInt16(k string, v int16)     { enc.AddInt64(k, int64(v)) }
func (enc *otlpEncoder) AddInt8(k string, v int8)       { enc.AddInt64(k, int64(v)) }
func (enc *otlpEncoder) AddUint(k string, v uint)       { enc.AddUint64(k, uint64(v)) }
func (enc *otlpEncoder) AddUint32(k string, v uint32)   { enc.AddUint64(k, uint64(v)) }
func (enc *otlpEncoder) AddUint16(k string, v uint16)   { enc.AddUint64(k, uint64(v)) }
func (enc *otlpEncoder) AddUint8(k string, v uint8)     { enc.AddUint64(k, uint64(v)) }
func (enc *otlpEncoder) AddUintptr(k string, v uintptr) { enc.AddUint64(k, uint64(v)) }
func (enc *otlpEncoder) AppendComplex64(v complex64)    { enc.appendComplex(complex128(v), 32) }
func (enc *otlpEncoder) AppendComplex128(v complex128)  { enc.appendComplex(complex128(v), 64) }
func (enc *otlpEncoder) AppendFloat64(v float64)        { enc.appendFloat(v, 64) }
func (enc *otlpEncoder) AppendFloat32(v float32)        { enc.appendFloat(float64(v), 32) }
func (enc *otlpEncoder) AppendInt(v int)                { enc.AppendInt64(int64(v)) }
func (enc *otlpEncoder) AppendInt32(v int32)            { enc.AppendInt64(int64(v)) }
func (enc *otlpEncoder) AppendInt16(v int16)            { enc.AppendInt64(int64(v)) }
func (enc *otlpEncoder) AppendInt8(v int8)              { enc.AppendInt64(int64(v)) }
func (enc *otlpEncoder) AppendUint(v uint)              { enc.AppendUint64(uint64(v)) }
func (enc *otlpEncoder) AppendUint32(v uint32)          { enc.AppendUint64(uint64(v)) }
func (enc *otlpEncoder) AppendUint16(v uint16)          { enc.AppendUint64(uint64(v)) }
func (enc *otlpEncoder) AppendUint8(v uint8)            { enc.AppendUint64(uint64(v)) }
func (enc *otlpEncoder) AppendUintptr(v uintptr)        { enc.AppendUint64(uint64(v)) }

func (enc *otlpEncoder) Clone() zapcore.Encoder {
	clone := enc.clone()
	clone.buf.Write(enc.buf.Bytes())
	return clone
}

func (enc *otlpEncoder) clone() *otlpEncoder {
	clone := getOTLPEncoder()
	clone.EncoderConfig = enc.EncoderConfig
	clone.spaced = enc.spaced
	clone.openNamespaces = enc.openNamespaces
	clone.buf = bufferPool.Get()
	return clone
}

var levelMap = map[string]uint{
	"trace": 1,
	"debug": 5,
	"info":  9,
	"warn":  13,
	"error": 17,
	"fatal": 21,
}

func (enc *otlpEncoder) level(l int) lpb.SeverityNumber {
	// In OpenTelemetry smaller numerical values in each range represent less
	// important (less severe) events. Larger numerical values in each range
	// represent more important (more severe) events.
	//
	// SeverityNumber range|Range name
	// --------------------|----------
	// 1-4                 |TRACE
	// 5-8                 |DEBUG
	// 9-12                |INFO
	// 13-16               |WARN
	// 17-20               |ERROR
	// 21-24               |FATAL
	//
	// Logr verbosity levels decrease in significance the greater the value.
	if l < 0 {
		l = 0
	}
	if l > int(lpb.SeverityNumber_SEVERITY_NUMBER_WARN4) {
		l = int(lpb.SeverityNumber_SEVERITY_NUMBER_WARN4)
	}
	return lpb.SeverityNumber(int(lpb.SeverityNumber_SEVERITY_NUMBER_WARN4) - l)
}

func (enc *otlpEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	final := enc.clone()
	// final.buf.AppendByte('{')

	if final.LevelKey != "" && final.EncodeLevel != nil {
		final.log.SeverityNumber = lpb.SeverityNumber(levelMap[ent.Level.String()])
		final.log.SeverityText = ent.Level.String()
		// final.addKey("SeverityNumber")
		// cur := final.buf.Len()
		// // final.EncodeLevel(ent.Level, final)
		// if cur == final.buf.Len() {
		// 	// User-supplied EncodeLevel was a no-op. Fall back to strings to keep
		// 	// output JSON valid.
		// 	final.AppendUint(levelMap[ent.Level.String()])
		// }

		// final.addKey("SeverityText")
		// final.AppendString(ent.Level.String())
	}
	if final.TimeKey != "" {
		// final.AddTime(final.TimeKey, ent.Time)
		final.log.TimeUnixNano = uint64(ent.Time.UnixNano())
	}
	if ent.LoggerName != "" && final.NameKey != "" {
		final.log.Attributes = append(enc.log.Attributes, &v1.KeyValue{
			Key: "logger",
			Value: &v1.AnyValue{
				Value: &v1.AnyValue_StringValue{
					StringValue: ent.LoggerName,
				},
			},
		})

		// final.addKey(final.NameKey)
		// cur := final.buf.Len()
		// nameEncoder := final.EncodeName

		// // if no name encoder provided, fall back to FullNameEncoder for backwards
		// // compatibility
		// if nameEncoder == nil {
		// 	nameEncoder = zapcore.FullNameEncoder
		// }

		// nameEncoder(ent.LoggerName, final)
		// if cur == final.buf.Len() {
		// 	// User-supplied EncodeName was a no-op. Fall back to strings to
		// 	// keep output JSON valid.
		// 	final.AppendString(ent.LoggerName)
		// }
	}
	// if ent.Caller.Defined {
	// 	if final.CallerKey != "" {
	// 		final.addKey(final.CallerKey)
	// 		cur := final.buf.Len()
	// 		final.EncodeCaller(ent.Caller, final)
	// 		if cur == final.buf.Len() {
	// 			// User-supplied EncodeCaller was a no-op. Fall back to strings to
	// 			// keep output JSON valid.
	// 			final.AppendString(ent.Caller.String())
	// 		}
	// 	}
	// 	if final.FunctionKey != "" {
	// 		final.addKey(final.FunctionKey)
	// 		final.AppendString(ent.Caller.Function)
	// 	}
	// }
	if final.MessageKey != "" {
		final.log.Body = &v1.AnyValue{
			Value: &v1.AnyValue_StringValue{
				StringValue: ent.Message,
			},
		}
		// final.addKey(enc.MessageKey)
		// final.addKey("message")
		// final.AppendString(ent.Message)
	}
	// if enc.buf.Len() > 0 {
	// 	final.addElementSeparator()
	// 	final.buf.Write(enc.buf.Bytes())
	// }
	addFields(final, fields)
	// final.closeOpenNamespaces()
	// if ent.Stack != "" && final.StacktraceKey != "" {
	// 	final.AddString(final.StacktraceKey, ent.Stack)
	// }
	// final.buf.AppendByte('}')
	// final.buf.AppendString(final.LineEnding)

	b, _ := json.Marshal(final.log)

	final.buf.AppendString(string(b))
	ret := final.buf
	putOTLPEncoder(final)

	return ret, nil
}

func (enc *otlpEncoder) truncate() {
	enc.buf.Reset()
}

func (enc *otlpEncoder) closeOpenNamespaces() {
	for i := 0; i < enc.openNamespaces; i++ {
		enc.buf.AppendByte('}')
	}
	enc.openNamespaces = 0
}

func (enc *otlpEncoder) addKeyVal(key string, val *v1.AnyValue) {
	enc.log.Attributes = append(enc.log.Attributes, &v1.KeyValue{
		Key:   key,
		Value: val,
	})
	// enc.addElementSeparator()
	// enc.buf.AppendByte('"')
	// enc.safeAddString(key)
	// enc.buf.AppendByte('"')
	// enc.buf.AppendByte(':')
	// if enc.spaced {
	// 	enc.buf.AppendByte(' ')
	// }
}

func (enc *otlpEncoder) addElementSeparator() {
	last := enc.buf.Len() - 1
	if last < 0 {
		return
	}
	switch enc.buf.Bytes()[last] {
	case '{', '[', ':', ',', ' ':
		return
	default:
		enc.buf.AppendByte(',')
		if enc.spaced {
			enc.buf.AppendByte(' ')
		}
	}
}

func (enc *otlpEncoder) appendFloat(val float64, bitSize int) {
	switch {
	case math.IsNaN(val):
		enc.buf.AppendString(`"NaN"`)
	case math.IsInf(val, 1):
		enc.buf.AppendString(`"+Inf"`)
	case math.IsInf(val, -1):
		enc.buf.AppendString(`"-Inf"`)
	default:
		enc.buf.AppendFloat(val, bitSize)
	}
}

// safeAddString JSON-escapes a string and appends it to the internal buffer.
// Unlike the standard library's encoder, it doesn't attempt to protect the
// user from browser vulnerabilities or JSONP-related problems.
func (enc *otlpEncoder) safeAddString(s string) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.AppendString(s[i : i+size])
		i += size
	}
}

// safeAddByteString is no-alloc equivalent of safeAddString(string(s)) for s []byte.
func (enc *otlpEncoder) safeAddByteString(s []byte) {
	for i := 0; i < len(s); {
		if enc.tryAddRuneSelf(s[i]) {
			i++
			continue
		}
		r, size := utf8.DecodeRune(s[i:])
		if enc.tryAddRuneError(r, size) {
			i++
			continue
		}
		enc.buf.Write(s[i : i+size])
		i += size
	}
}

// tryAddRuneSelf appends b if it is valid UTF-8 character represented in a single byte.
func (enc *otlpEncoder) tryAddRuneSelf(b byte) bool {
	if b >= utf8.RuneSelf {
		return false
	}
	if b >= 0x20 && b != '\\' && b != '"' {
		enc.buf.AppendByte(b)
		return true
	}
	switch b {
	case '\\', '"':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte(b)
	case '\n':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('n')
	case '\r':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('r')
	case '\t':
		enc.buf.AppendByte('\\')
		enc.buf.AppendByte('t')
	default:
		// Encode bytes < 0x20, except for the escape sequences above.
		enc.buf.AppendString(`\u00`)
		enc.buf.AppendByte(_hex[b>>4])
		enc.buf.AppendByte(_hex[b&0xF])
	}
	return true
}

func (enc *otlpEncoder) tryAddRuneError(r rune, size int) bool {
	if r == utf8.RuneError && size == 1 {
		enc.buf.AppendString(`\ufffd`)
		return true
	}
	return false
}

func addFields(enc zapcore.ObjectEncoder, fields []zapcore.Field) {
	for i := range fields {
		fields[i].AddTo(enc)
	}
}

func defaultReflectedEncoder(w io.Writer) zapcore.ReflectedEncoder {
	enc := json.NewEncoder(w)
	// For consistency with our custom JSON encoder.
	enc.SetEscapeHTML(false)
	return enc
}
