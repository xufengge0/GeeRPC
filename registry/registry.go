package registry

import (
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)
// 服务注册中心
type GeeRegistry struct {
	timeout time.Duration
	mu      sync.Mutex
	servers map[string]*ServerItem
}
// 服务实例
type ServerItem struct {
	Addr  string
	start time.Time
}
const (
	defaultPath    = "/_geerpc_/registry"
	defaultTimeout = time.Minute * 5 // 默认超时时间
)

func New(timeout time.Duration) *GeeRegistry {
	return &GeeRegistry{
		timeout: timeout,
		servers: make(map[string]*ServerItem),
	}
}

var DefaultGeeRegistry = New(defaultTimeout)

// 添加服务实例，如果服务已经存在，则更新
func (r *GeeRegistry) putServer(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.servers[addr]
	if s == nil {
		r.servers[addr] = &ServerItem{Addr: addr, start: time.Now()}
	} else {
		s.start = time.Now() // 更新 start
	}
}

// 返回可用的服务列表，如果存在超时的服务，则删除
func (r *GeeRegistry) aliveServers() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var alive []string
	for addr, s := range r.servers {
		if s.start.Add(r.timeout).After(time.Now()) {
			alive = append(alive, addr)
		} else {
			delete(r.servers, addr)
		}
	}
	sort.Strings(alive)
	return alive
}

// 实现了http.Handler接口，对外提供服务发现和注册中心功能
func (r *GeeRegistry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodGet:
		// 返回所有可用的服务列表
		w.Header().Set("X-Geerpc-Servers", strings.Join(r.aliveServers(), ","))
	case http.MethodPost:
		addr := req.Header.Get("X-Geerpc-Server")
		if addr == "" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		r.putServer(addr)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
func (r *GeeRegistry) HandleHTTP(registryPath string) {
	http.Handle(registryPath, r)
	log.Println("rpc registry path:", registryPath)
}
func HandleHTTP() {
	DefaultGeeRegistry.HandleHTTP(defaultPath)
}

// 服务端向注册中心发送心跳，duration是心跳间隔时间 填0则使用默认时间
func Heartbeat(registry, addr string, duration time.Duration) {
	if duration == 0 {
		// 默认心跳周期时间
		duration = defaultTimeout - time.Duration(1)*time.Minute
	}
	var err error
	err = sendHeartbeat(registry, addr)
	go func() {
		t := time.NewTicker(duration) // 定时发送心跳
		for err == nil {
			<-t.C
			err = sendHeartbeat(registry, addr)
		}
	}()
}
// registry 服务中心地址 ,addr 服务实例地址
func sendHeartbeat(registry, addr string) error {
	log.Println("send heart beat to registry:", addr)

	// 发送 POST 请求, 服务端向注册中心发送心跳并且注册自己的地址
	httpclient := &http.Client{}
	req, _ := http.NewRequest("POST", registry, nil)
	req.Header.Set("X-Geerpc-Server", addr)

	if _, err := httpclient.Do(req); err != nil {
		log.Println("rpc server: heart beat err:", err)
		return err
	}
	return nil
}
