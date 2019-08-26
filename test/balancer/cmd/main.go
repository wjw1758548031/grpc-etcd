package main

import (
	"context"
	"flag"
	"fmt"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	balancer "test/balancer"
	"test/balancer/proto"
	"time"
)

var (
	//启动加载参数
	port = flag.Int("p", 8090, "grpc port")
)

func init() {
	flag.Parse()
	flag.VisitAll(func(flag *flag.Flag) {
		log.WithFields(log.Fields{
			"name":  flag.Name,
			"value": flag.Value,
		}).Info("flag params")
	})

}

//etcd 自己在linx上注册
//protoc --proto_path=  -I . --go_out=plugins=grpc:.  hello_world.proto 下载proto生成文件，实在找不到proto可以去csdn找执行文件
//这边的负载均衡是由服务端控制

func main() {
	r := balancer.EtcdRegister{
		EtcdAddrs:   []string{"47.102.102.47:2379"}, //etcd的服务端
		DialTimeout: 3,                              //连接超时
	}

	//删除etcdkey 和租约id
	defer r.Stop()

	//创建一个repc服务端，但是还没有任何的接收
	server := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_recovery.UnaryServerInterceptor(),
		)),
	)

	serverMetHod := &serverHod{}
	//服务注册
	proto.RegisterHelloWorldServer(server, serverMetHod)

	//创建本地连接
	sock, err := net.Listen("tcp", fmt.Sprintf(":%v", *port))
	balancer.PanicIfError("fail to listen port", err)

	//自己设置的服务名称和版本
	srvName, version := "greeting", "v1"
	info := balancer.ServerNodeInfo{
		Name:           srvName,
		Version:        version,
		Addr:           fmt.Sprintf("127.0.0.1:%d", *port), //地址
		Weight:         1,
		LastUpdateTime: time.Now(),
	}
	//服务注册在etcd上
	r.Register(info, 5)

	//携程去获取etcd的信息
	/*go func() {
		//断续器，每过多久去访问一次
		ticker := time.NewTicker(time.Second * 10)
		for {
			<-ticker.C
			registerInfo, err := r.GetServiceInfo()
			if err == nil {
				log.Println(fmt.Sprintf("register service ok:name=%v,addr=%v", registerInfo.Name, registerInfo.Addr))
			} else {
				log.Println(fmt.Sprintf("err:", err))
				return
			}
		}
	}()*/

	log.WithFields(log.Fields{
		"port": *port,
	}).Info("start server....")

	reflection.Register(server)
	if err := server.Serve(sock); err != nil {
		balancer.PanicIfError("fail to start grpc server", err)
	}

}

// server is used to implement helloworld.GreeterServer.
type serverHod struct{}

// SayHello implements helloworld.GreeterServer  //
func (s *serverHod) Hello(ctx context.Context, in *proto.HelloRequest) (*proto.HelloResponse, error) {
	log.Infof("%v: Receive is %s\n", time.Now(), in.Name)
	return &proto.HelloResponse{Id: int64(666)}, nil
}
