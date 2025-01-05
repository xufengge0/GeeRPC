/* 
手动实现注册中心服务发现
*/
package xclient

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

type SelectMode int

const (
	RandomSelect     SelectMode = iota // 随机选择
	RoundRobinSelect                   // 轮询选择
)
// 服务发现接口
type Discovery interface {
	Refresh() error                      // 更新服务器节点
	Update(servers []string) error       // 手动更新服务器节点
	Get(mode SelectMode) (string, error) // 根据负载均衡策略，选择一个健康的服务器节点
	GetAll() ([]string, error)           // 返回所有的服务器节点
}
// 服务发现结构体
type MultiServerDiscovery struct {
	r       *rand.Rand
	mu      sync.Mutex
	servers []string // 服务器节点
	index   int 	 // 当前选择的服务器节点的下标
}

func NewMultiServerDiscovery(servers []string) *MultiServerDiscovery {
	d := &MultiServerDiscovery{
		servers: servers,
		r:       rand.New(rand.NewSource(time.Now().Unix())),
	}
	d.index = d.r.Intn(math.MaxInt32 - 1)
	return d
}

var _ Discovery = (*MultiServerDiscovery)(nil)

func (d *MultiServerDiscovery) Refresh() error {
	return nil
}
func (d *MultiServerDiscovery) Update(servers []string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.servers = servers
	return nil
}
// 根据负载均衡策略，返回给客户端一个健康的服务器节点
func (d *MultiServerDiscovery) Get(mode SelectMode) (string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := len(d.servers)
	if n == 0 {
		return "", errors.New("rpc discovery: no available servers")
	}
	switch mode {
	case RandomSelect:
		return d.servers[d.r.Intn(n)], nil
	case RoundRobinSelect:
		s := d.servers[d.index%n]
		d.index = (d.index + 1) % n
		return s, nil
	}
	return "", errors.New("rpc discovery: not supported select mode")
}
// 返回客户端所有的服务器节点
func (d *MultiServerDiscovery) GetAll() ([]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	servers := make([]string, len(d.servers), len(d.servers))
	copy(servers, d.servers)
	return servers, nil
}
