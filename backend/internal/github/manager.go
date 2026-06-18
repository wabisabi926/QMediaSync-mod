package github

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// ConnectionType 连接类型枚举
type ConnectionType string

const (
	ConnectionTypeDirect      ConnectionType = "direct"       // 直连
	ConnectionTypeProxy       ConnectionType = "proxy"        // 用户代理
	ConnectionTypeGitHubProxy ConnectionType = "github_proxy" // GitHub代理URL
	ConnectionTypeFailed      ConnectionType = "failed"       // 连接失败
)

// GitHubAccess GitHub访问配置
type GitHubAccess struct {
	Type       ConnectionType // 当前使用的连接类型
	Client     *http.Client   // HTTP客户端
	ProxyURL   string         // 代理URL（仅记录类型，不记录密码）
	LastTested time.Time      // 上次测试时间
	Cached     bool           // 是否为缓存结果
}

// Manager GitHub访问管理器
type Manager struct {
	sync.RWMutex
	current     *GitHubAccess
	testTimeout time.Duration // 测试超时时间
	cacheValid  time.Duration // 缓存有效期
	httpProxy   string        // HTTP代理
}

const (
	// GithubProxyURL 内置GitHub代理URL（系统加速节点）
	GithubProxyURL = "https://gh.llkk.cc"
)

var defaultManager *Manager

// InitManager 初始化GitHub访问管理器
// httpProxy - HTTP代理地址
func InitManager(httpProxy string) {
	defaultManager = &Manager{
		testTimeout: 3 * time.Second,  // 3秒测试超时
		cacheValid:  10 * time.Minute, // 缓存10分钟
		httpProxy:   httpProxy,
	}
}

// UpdateConfig 更新管理器的代理配置
func UpdateConfig(httpProxy string) {
	defaultManager.Lock()
	defer defaultManager.Unlock()

	defaultManager.httpProxy = httpProxy

	// 清除缓存，以便使用新配置
	defaultManager.current = nil
	log.Printf("GitHub管理器配置已更新，缓存已清除")
}

// GetManager 获取管理器实例
func GetManager() *Manager {
	if defaultManager == nil {
		// 使用空字符串初始化，后续可以通过UpdateConfig更新
		InitManager("")
	}
	return defaultManager
}

// TestConnection 测试指定方式的连接是否可用
func (m *Manager) TestConnection(connType ConnectionType, proxyURL string) bool {
	client := &http.Client{
		Timeout: m.testTimeout,
	}

	// 根据类型配置代理
	var transport *http.Transport
	if connType == ConnectionTypeProxy && proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			log.Printf("代理URL解析失败: %v", err)
			return false
		}
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	} else if connType == ConnectionTypeGitHubProxy && proxyURL != "" {
		// GitHub代理URL模式：将请求发送到代理服务器
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			log.Printf("GitHub代理URL解析失败: %v", err)
			return false
		}
		transport = &http.Transport{
			Proxy: http.ProxyURL(proxy),
		}
	}

	if transport != nil {
		client.Transport = transport
	}

	resp, err := client.Get("https://api.github.com/repos/qicfan/qmediasync/releases")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200
}

// GetBestConnection 获取最佳连接方式（带缓存）
func (m *Manager) GetBestConnection() (*GitHubAccess, error) {
	m.RLock()
	// 检查缓存是否有效
	if m.current != nil && time.Since(m.current.LastTested) < m.cacheValid {
		m.RUnlock()
		m.current.Cached = true // 标记为缓存
		log.Printf("使用缓存的GitHub连接: %s", m.current.Type)
		return m.current, nil
	}
	m.RUnlock()

	m.Lock()
	defer m.Unlock()

	// 双重检查，避免重复测试
	if m.current != nil && time.Since(m.current.LastTested) < m.cacheValid {
		m.current.Cached = true // 标记为缓存
		return m.current, nil
	}

	// 1. 测试用户代理(优先使用用户代理，因为直连可能无法下载安装包)
	if m.httpProxy != "" {
		if m.TestConnection(ConnectionTypeProxy, m.httpProxy) {
			proxy, err := url.Parse(m.httpProxy)
			if err == nil {
				m.current = &GitHubAccess{
					Type: ConnectionTypeProxy,
					Client: &http.Client{
						Timeout:   30 * time.Second,
						Transport: &http.Transport{Proxy: http.ProxyURL(proxy)},
					},
					ProxyURL:   m.httpProxy,
					LastTested: time.Now(),
					Cached:     false,
				}
				log.Printf("GitHub连接方式: 用户代理")
				return m.current, nil
			}
		}
		// 如果用户配置了代理但代理不可用，直接返回错误
		// 参考原始TestGithub逻辑：如果proxy != ""，返回failed
		return nil, fmt.Errorf("用户配置的代理不可用: %s", m.httpProxy)
	}

	// 2. 测试直连
	if m.TestConnection(ConnectionTypeDirect, "") {
		m.current = &GitHubAccess{
			Type:       ConnectionTypeDirect,
			Client:     &http.Client{Timeout: 30 * time.Second}, // 使用较长超时
			LastTested: time.Now(),
			Cached:     false,
		}
		log.Printf("GitHub连接方式: 直连")
		return m.current, nil
	}

	// 3. 测试GitHub代理URL（仅在用户未配置代理时）
	// 使用内置的GitHub加速节点
	if m.TestConnection(ConnectionTypeGitHubProxy, GithubProxyURL) {
		proxy, err := url.Parse(GithubProxyURL)
		if err == nil {
			m.current = &GitHubAccess{
				Type: ConnectionTypeGitHubProxy,
				Client: &http.Client{
					Timeout:   30 * time.Second,
					Transport: &http.Transport{Proxy: http.ProxyURL(proxy)},
				},
				ProxyURL:   GithubProxyURL,
				LastTested: time.Now(),
				Cached:     false,
			}
			log.Printf("GitHub连接方式: GitHub代理URL (%s)", GithubProxyURL)
			return m.current, nil
		}
	}

	// 4. 全部失败
	return nil, fmt.Errorf("无法连接到GitHub，请检查网络或代理设置")
}

// GetClient 获取配置好的HTTP客户端
func (m *Manager) GetClient() (*http.Client, error) {
	access, err := m.GetBestConnection()
	if err != nil {
		return nil, err
	}
	return access.Client, nil
}

// GetClientWithCache 强制使用缓存的连接（不测试）
func (m *Manager) GetClientWithCache() (*http.Client, error) {
	m.RLock()
	defer m.RUnlock()

	if m.current == nil {
		return nil, fmt.Errorf("没有可用的GitHub连接")
	}

	return m.current.Client, nil
}

// ClearCache 清除缓存，下次调用会重新测试
func (m *Manager) ClearCache() {
	m.Lock()
	defer m.Unlock()
	m.current = nil
	log.Printf("GitHub连接缓存已清除")
}
