package main

import (
	"flag"
	"fmt"
	"google.golang.org/grpc/resolver"
	"log"

	//grpclb "github.com/wwcd/grpc-lb/etcdv3"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"test/balancer/proto"
	"time"
	balancer "test/balancer"
)


var (
	svc = flag.String("service", "hello_service", "service name")
	reg = flag.String("reg", "47.102.102.47:2379", "register etcd address")
)

func main() {
	srvName, version := "greeting", "v1"
	r := balancer.NewEtcdResolver([]string{"47.102.102.47:2379"}, srvName, version, int64(10))
	//注册
	resolver.Register(r)
	target := fmt.Sprintf("%s:///%s", r.Scheme(), srvName)
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	//创建连接
	conn, err := grpc.DialContext(ctx, target,
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithBalancerName(roundrobin.Name),
	)
	balancer.PanicIfError("fail to dial grpc server", err)
	//关闭连接
	defer conn.Close()
	client := proto.NewHelloWorldClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()
	req := proto.HelloRequest{
		Name: "Tom wjw",
	}
	resp, err := client.Hello(ctx, &req)
	balancer.PanicIfError("fail to call sayHello", err)
	log.Printf("resp:%v", resp.Id)

}

//普通访问rpc方式
func click(){
	//conn, err := grpc.Dial("127.0.0.1:50059",grpc.WithInsecure())//可通 验证过
	conn, err := grpc.Dial("127.0.0.1:50059", grpc.WithInsecure(),grpc.WithBalancerName(roundrobin.Name))
	if err != nil {
		panic(err)
	}

	//创建客户端
	client := proto.NewHelloWorldClient(conn)

	for {
		resp, err := client.Hello(context.Background(), &proto.HelloRequest{Name: "hellowjw"}, grpc.FailFast(true))
		if err != nil {
			panic(err)
		} else {
			fmt.Println(resp)
		}

		<-time.After(time.Second)
	}
}

/*

标准普通连接的方式
func main() {
	target := "127.0.0.1:8090"

	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	conn, err := grpc.DialContext(ctx, target,
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithBalancerName(roundrobin.Name),
	)

	util.PanicIfError("fail to dial grpc server", err)
	defer conn.Close()

	client := hello.NewHelloServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*500)
	defer cancel()

	req := hello.HelloRequest{
		Name: "Tom Clay",
	}
	resp, err := client.SayHello(ctx, &req)
	util.PanicIfError("fail to call sayHello", err)
	log.Printf("resp:%v", resp.Reply)
}

 */