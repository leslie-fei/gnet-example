module gnet-example

go 1.22

replace github.com/panjf2000/gnet/v2 => ../gnet

require github.com/panjf2000/gnet/v2 v2.0.0-00010101000000-000000000000

require (
	github.com/panjf2000/ants/v2 v2.9.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.21.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
)
