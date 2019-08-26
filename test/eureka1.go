package main

import (
	"fmt"
	"github.com/go-chi/chi"
	"github.com/opentracing/opentracing-go"
	"github.com/openzipkin-contrib/zipkin-go-opentracing"
	"net/http"
	"net/http/pprof"
	"os"
)


//github.com/hudl/fargo v1.2.1-0.20180614092839-fce5cf495554 go mod弄这个就ok


func main(){
	s := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", "127.0.0.1", 8880),
		Handler: Router(chi.NewMux()),
	}
	host, _ := os.Hostname()
	if host == "" {
		host = "host"
	}
	conFig := Config{Name:"ceshi",Host:host,DingToken:"",Debug:true}
	NewTracer(conFig)
	s.ListenAndServe()
}

type Config struct {
	Name      string
	Host      string
	DingToken string
	Debug     bool
}

func NewTracer(cfg Config) (opentracing.Tracer, func(), error) {
	recorder := zipkintracer.NewRecorder(
		nil,
		false,
		cfg.Host,
		cfg.Name,
	)

	// Create our tracer.
	tracer, err := zipkintracer.NewTracer(
		recorder,
		zipkintracer.ClientServerSameSpan(true),
		zipkintracer.TraceID128Bit(true),
	)
	/*if err != nil {
		if err1 := collector.Close(); err1 != nil {
			errors.LogError(err1)
		}
		return nil, nil, err
	}*/

	opentracing.SetGlobalTracer(tracer)
	/*return tracer, func() {
		if err1 := collector.Close(); err1 != nil {
			errors.LogError(err1)
		}
	}, nil*/
	return nil, nil, err
}


//路由
func Router(mux *chi.Mux) *chi.Mux{
	//中间界
	/*mux.Use(
		httpser.SpanMiddFac(func(span opentracing.Span) interface{} {
			return &gContext{
				Res:  res.(Res),
				span: span,
			}
		}),
		recoverHigh,
		easyDebug,
		httpser.CORS,
	)*/


	mux.Get("/debug/pprof/", pprof.Index)
	mux.Get("/debug/pprof/allocs", pprof.Index)
	mux.Get("/debug/pprof/block", pprof.Index)
	mux.Get("/debug/pprof/goroutine", pprof.Index)
	mux.Get("/debug/pprof/heap", pprof.Index)
	mux.Get("/debug/pprof/mutex", pprof.Index)
	mux.Get("/debug/pprof/threadcreate", pprof.Index)

	mux.Get("/debug/pprof/cmdline", pprof.Cmdline)
	mux.Get("/debug/pprof/profile", pprof.Profile)
	mux.Get("/debug/pprof/symbol", pprof.Symbol)
	mux.Get("/debug/pprof/trace", pprof.Trace)

	//mux.Post("/ieo/project/list", IeoProject{}.ProjectList)

	mux.Group(func(r chi.Router) {
	//	r.Use(AuthMember)
		// 项目列表接口（申购成功）
		r.Post("/test/test", Test{}.test)
	})
	return mux
}

type Test struct{}

func (Test) test(writer http.ResponseWriter, request *http.Request){
	fmt.Println("进入1")
}