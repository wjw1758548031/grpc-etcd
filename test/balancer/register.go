package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
	"go.etcd.io/etcd/clientv3"
	"errors"
)

// NewEtcdRegister return a EtcdRegister, param: etcd endpoints addrs. Just for simple use
func NewEtcdRegister(etcdAddrs []string) *EtcdRegister {
	return &EtcdRegister{
		EtcdAddrs:   etcdAddrs,
		DialTimeout: 3,
	}
}

// EtcdRegister ...
type EtcdRegister struct {
	EtcdAddrs   []string		//etcd的地址
	DialTimeout int

	// server
	srvInfo     ServerNodeInfo
	srvTTL      int64
	cli         *clientv3.Client
	leasesID    clientv3.LeaseID
	keepAliveCh <-chan *clientv3.LeaseKeepAliveResponse
	closeCh     chan struct{}
}

// ServerNodeInfo regiested to etcd  注册为etcd 应该算是重写
type ServerNodeInfo struct {
	Name           string    // 服务名称
	Version        string    // 服务版本号，用于服务升级过程中，配置兼容问题
	HostName       string    // 主机名称
	Addr           string    // 服务的地址, 格式为 ip:port，参见 https://github.com/grpc/grpc/blob/master/doc/naming.md
	Weight         uint16    // 服务权重
	LastUpdateTime time.Time // 更新时间使用租约机制
	MetaData       string    // 后续考虑替换成 interface{} 推荐 json 格式，服务端与客户端可以约定相关格式
}

// Errors ...
var (
	ErrNoEtcAddrs = errors.New("no etcd addrs provide")
)

// Register a service base on ServerNodeInfo。
func (r *EtcdRegister) Register(srvInfo ServerNodeInfo, ttl int64) (chan<- struct{}, error) {
	// check etcd addrs 没有地址 直接退出
	if len(r.EtcdAddrs) <= 0 {
		return nil, ErrNoEtcAddrs
	}

	var err error
	//new一个etcd客户端
	r.cli, err = clientv3.New(clientv3.Config{
		Endpoints:   r.EtcdAddrs,
		DialTimeout: time.Duration(r.DialTimeout) * time.Second,
	})

	if err != nil {
		return nil, err
	}
	//把本地的微服务信息给r
	r.srvInfo = srvInfo
	//租约过期时间
	r.srvTTL = ttl
	//时间上下文释放
	leaseCtx, cancel := context.WithTimeout(context.Background(), time.Duration(r.DialTimeout)*time.Second)
	//创建租约
	leaseResp, err := r.cli.Grant(leaseCtx, ttl)
	cancel()
	if err != nil {
		return nil, err
	}
	//赋值租约id
	r.leasesID = leaseResp.ID
	//使租约变成永久
	r.keepAliveCh, err = r.cli.KeepAlive(context.Background(), leaseResp.ID)
	if err != nil {
		return nil, err
	}

	r.closeCh = make(chan struct{})

	// Registe the svc At FIRST TIME
	err = r.register()
	if err != nil {
		return nil, err
	}

	//不停的刷新,时自己的微服务地址送达到etcd
	go r.keepAlive()

	return r.closeCh, nil
}

// Stop registe process
func (r *EtcdRegister) Stop() {
	r.closeCh <- struct{}{}
}

// GetServiceInfo used get service info from etcd. Used for TEST
func (r *EtcdRegister) GetServiceInfo() (ServerNodeInfo, error) {
	//连接不上etcd的话后面的程序都会走不了  这里是查询
	resp, err := r.cli.Get(context.Background(), BuildRegPath(r.srvInfo))
	if err != nil {
		return r.srvInfo, err
	}

	infoRes := ServerNodeInfo{}
	for idx, val := range resp.Kvs {
		log.Printf("[%d] %s %s\n", idx, string(val.Key), string(val.Value))
	}

	// only return one recorde
	if resp.Count >= 1 {
		err = json.Unmarshal(resp.Kvs[0].Value, &infoRes)
		if err != nil {
			return infoRes, err
		}
	}

	return infoRes, nil
}

// register service into to etcd
func (r *EtcdRegister) register() error {
	r.srvInfo.LastUpdateTime = time.Now()
	regData, err := json.Marshal(r.srvInfo)
	if err != nil {
		return err
	}
	//把本机的信息put到etcd 给的值是结构的序列化信息，etcd会进行解析
	_, err = r.cli.Put(context.Background(), BuildRegPath(r.srvInfo), string(regData), clientv3.WithLease(r.leasesID))

	return err
}

// unregister service from etcd
func (r *EtcdRegister) unregister() error {
	_, err := r.cli.Delete(context.Background(), BuildRegPath(r.srvInfo))

	return err
}



// keepAlive ...
func (r *EtcdRegister) keepAlive() {
	// 后续考虑将更新时间设定某个范围的浮动
	ticker := time.NewTicker(time.Second * time.Duration(r.srvTTL))
	for {
		select {
		case <-r.closeCh:
			// log.Printf("unregister %s\n", r.srvInfo.Addr)
			//删除key值
			err := r.unregister()
			if err != nil {
				log.Printf("unregister %s failed. %s\n", r.srvInfo.Addr, err.Error())
			}
			// clean lease
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			//删除指定的租约id
			_, err = r.cli.Revoke(ctx, r.leasesID)
			if err != nil {
				log.Printf("Revoke lease %d for %s failed. %s\n", r.leasesID, r.srvInfo.Addr, err.Error())
			}

			return
		case <-r.keepAliveCh:
			// Just do nothing, closeCh should revoke lease
			// log.Printf("recv keepalive %s, %d\n", r.srvInfo.Addr, len(r.keepAliveCh))
		case <-ticker.C:
			// Timeout, no check out and just put
			// log.Printf("register %s\n", r.srvInfo.Addr)
			err := r.register()
			if err != nil {
				log.Printf("register %s failed. %s\n", r.srvInfo.Addr, err.Error())
			}

		}
	}
}

// BuildPrefix has last "/"
func BuildPrefix(info ServerNodeInfo) string {
	return fmt.Sprintf("/%s/%s/", info.Name, info.Version)
}

func BuildRegPath(info ServerNodeInfo) string {
	return fmt.Sprintf("%s%s",
		BuildPrefix(info), info.Addr)
}

func PanicIfError(msg string, err error) {
	if err != nil {
		log.Panicf("%v, err=%v", msg, err)
	}
}