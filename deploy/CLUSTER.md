# 多服务器负载均衡

本模式让多台 Sub2API 共同处理模型请求，沿用同一套 PostgreSQL、Redis、账号、用户和计费数据。没有新增数据库迁移；默认关闭，单机行为保持兼容。

```text
客户端 → 香港 Nginx（HTTPS）→ 香港 HAProxy（127.0.0.1:8088）
                              ├─ 美国 Sub2API primary
                              ├─ 香港 Sub2API worker
                              └─ 更多 Sub2API worker
                                    ↓
                          共用美国 PostgreSQL / Redis
```

## 路由与任务

| 请求/任务 | 分配方式 |
|---|---|
| Messages、Chat Completions、Gemini、Embeddings、同步图片 | 按最少连接及节点权重分配 |
| Responses HTTP / WebSocket，含根路径和 Codex 别名 | 根据 API Key + 显式会话标识一致性哈希；无会话标识时按 API Key |
| 管理后台、用户网页、支付回调、插件管理 | 固定 primary |
| 异步/批量图片、视频任务、本地文件、Live/Realtime | 固定 primary |
| 账号定时刷新、过期处理、监控聚合、定时检测、报表/备份 | primary 执行 |
| 请求计费队列、使用记录、共享并发限制、缓存订阅 | 每个节点执行 |

Responses 优先使用 `X-Sub2API-Session-ID`，其次 `conversation_id`，再其次 `session_id`（长度 1–256 字节）。同一会话必须在每次 HTTP 请求及 WebSocket 重连时使用相同标识；没有标识时，同一 API Key 固定节点。自定义标识只参与入口分配，不改变上游 API 的会话契约。不同会话可使用同一个 API Key 分到不同节点。不要每轮随机生成新标识。Nginx 默认可能忽略含下划线的头，使用原生头时须配置 `underscores_in_headers on`。普通 Chat Completions 等请求即使用同一个 Key，也能同时分到多台机器。权重 100:100 表示相等权重，不保证每一分钟请求数严格各半。

节点状态变化或增删节点可能改变会话归属。已建立的流/连接始终留在原节点，不能迁移；中断后需要客户端重连，部分 `previous_response_id` 会话可能需要完整上下文。代理配置关闭重试，避免已发送的生成/扣费请求被自动重放。

集群模式共享 Gemini 内容摘要、Claude 协议转 OpenAI 的缓存标识、Responses 账号与响应归属、opaque turn state；不保存完整提示词，不共享进程内 socket。摘要链批量查询限制为最近 256 个前缀，每次只发一个 Redis MGET。连接专属（store=false）续接的归属节点或进程改变时，HTTP 返回 409 `cluster_continuation_unavailable`，WebSocket 以策略错误关闭；客户端须携带完整上下文且移除 `previous_response_id` 后重试。此时不会自动把增量当完整对话重放。上游持久化的响应沿用原有恢复流程。

这是应用层横向扩容，不是数据库双主，也不是完整高可用：primary 的后台、支付、异步任务，数据库、Redis、香港入口仍有单点。primary 宕机后，健康 worker 可以继续承接上述可分配的同步模型请求（前提是共享数据服务可用）。主节点管理功能恢复需要人工切换到准备好的替代节点。

## 部署前提

1. 所有节点运行**相同的新版本**。先将原主节点开启集群模式，再启动工作节点；旧单机版本启动时会清理其他进程的并发槽位，不能混用。
2. 用 WireGuard 等私有加密隧道连通服务器。示例 `10.70.0.1/10.70.0.2` 是占位地址，不是现有服务器配置。生成器仅接受私网/本机 IP，地址为私网本身不代表链路已加密。数据库和 Redis 只允许节点私网地址访问。
3. 共用同一个 PostgreSQL 数据库、Redis 实例及 DB 编号、现有 `jwt.secret`、`totp.encryption_key`。不得在新节点生成替代密钥。集群模式缺少显式密钥会拒绝启动。
4. 一套集群只运行一个 primary。primary 也参与模型请求分流；其他节点配置为 worker。工作节点拒绝管理、支付和本地任务接口的直接访问。
5. 每台节点都需要一致的插件程序/版本、定价及自定义文件、上游代理可达性。后台安装插件只写主节点本地目录，需要同步安装到工作节点并滚动重启。相对路径下的文件、宿主机代理地址和 loopback 代理不能直接跨主机复制使用。
6. 节点新增会增加数据库连接池总量；按全部节点合计配置 `database.max_open_conns`、Redis pool 和数据库 `max_connections`。先实测美港 RTT 和上游请求表现，再逐步增加香港权重；40 核不代表上游额度或生成速度翻倍。

## 1. 配置应用节点

保留原来的 `config.yaml` 和 data 目录；新节点复制受保护的配置及需要的插件/自定义资源，修改数据库/Redis 地址为共享数据服务的隧道地址。不要在每台机器新建一套业务数据库。

参照 `cluster/primary.env.example` 和 `cluster/worker.env.example` 建立各机 `/etc/sub2api/cluster.env`（权限 600），再安装 `cluster/sub2api.service.conf` 为 systemd drop-in。原有 service 的 User、WorkingDirectory、配置路径和密钥 drop-in 保持不变。

配置也可写入 YAML：

```yaml
cluster:
  enabled: true
  node_id: hk-worker
  role: worker
  drain_seconds: 10
  shutdown_timeout_seconds: 120
```

节点 ID 必须与服务器清单一致。各节点订阅设置变更事件，同时每 10 秒核对完整快照；最近 30 秒未成功读取快照时 readiness 失败。私网 `/readyz?details=1` 显示节点、活动请求数、配置摘要版本、最近同步时间和失败次数。配置值与密钥不出现在状态中。订阅变更同步删除 Redis 缓存并通知其他节点；每 5 秒检查持久化版本号，补偿断线漏通知。认证缓存继续使用已有的通知与 TTL。跨节点变更不是强一致事务，插件、本地文件及进程配置仍需单独同步。

`cluster.billing_failure_mode` 默认为 `closed`：强制开启余额预留，无法估算费用或预留依赖异常时拒绝新的收费生成；已知免费、非计量请求和原有订阅计费规则保留。固定费用的订阅/API Key 预留也拒绝依赖异常，并释放此前已取得的预留。显式设为 `open` 可沿用原降级策略，此时网络异常可能绕过预留保护。该策略不等于最大消费额保证：生成费用仍由实际 token 用量结算，已有估算上限与在途计费语义保留。

## 容量与延迟检查

- PostgreSQL 每节点默认上限 256，Redis 池每节点默认 1024；这些是上限，不是启动即占满。配置总预算时把所有业务节点、后台任务和运维连接一起算入，保留余量，不能按机器数简单复制满额配置。
- 清单节点可加 `max_connections`（1–20000，省略/0 不额外限制），为每个 HAProxy backend 的该服务器设置并发和队列上限；排队超时为 10 秒。gateway 与 responses 是独立 backend，预算须合计，避免让慢节点持续积压。
- 从两台机器分别测共享 PostgreSQL/Redis 的延迟、连接等待、首字时间 P50/P95、会话续接失败数及上游缓存命中率；先提高小比例权重再加流量。这里只能测本机代理和缓存协议，不能据此推算真实美港吞吐。
- 保留进程内 HTTP/WS 连接池及认证热缓存。只有需要跨节点一致的元数据放 Redis；上游模型缓存率仍取决于账号、模型、前缀和供应商策略。

首次复制到新机时应使用现有完整配置完成启动，不能暴露安装向导。先从私网验证：

```sh
curl --fail 'http://10.70.0.2:8080/readyz?node=hk-worker&role=worker'
```

`/health` 是存活检查；`/readyz` 会检查 PostgreSQL、Redis、节点身份及下线状态，不返回连接信息或密钥。任一依赖失败返回 503。该路径由 HAProxy 拦截，不对外暴露。

## 2. 配置香港入口

安装 HAProxy **3.2 或以上**。在已有 HTTPS Nginx 后面使用 HAProxy，监听仅限 loopback。参考 `cluster/nginx-location.conf` 合并到现有站点；保留证书/域名，关闭流式缓冲和生成请求重试。若前面有 CDN，先配置受信任的 real-IP 来源。

应用节点在系统设置中关闭“直接信任转发 IP”兼容模式，清空自定义客户端 IP 头；`server.trusted_proxies` 只包含 HAProxy 实际连接源地址。该兼容开关已经保存到数据库时，以后台设置为准，单改环境变量不会覆盖已保存值。

从仓库根目录构建工具（只生成配置，不连接服务器）：

```sh
cd backend
go build -o ../cluster-config ./cmd/cluster-config
cd ..
cp deploy/cluster/nodes.example.json /secure/path/nodes.json
# 编辑节点隧道地址、ID、角色、权重；disabled:true 可停用 worker。
./cluster-config -config /secure/path/nodes.json > /tmp/sub2api-haproxy.cfg
haproxy -c -f /tmp/sub2api-haproxy.cfg
```

确认配置校验通过后，由运维将其安装到 HAProxy 使用的配置路径并 reload。不要覆盖承载其他业务的 HAProxy 配置；可为 Sub2API 单独运行一个 HAProxy 服务。生成的配置没有 TLS 私钥、API Key 或数据库凭证，也不记录请求 URL/请求头。

独立服务模板见 `cluster/sub2api-haproxy.service`，配置路径为 `/etc/sub2api/haproxy.cfg`。模板假设软件包已创建 `haproxy` 用户且程序位于 `/usr/sbin/haproxy`；按实际安装位置调整，并确保该用户可读取配置。正常 reload 前会校验配置，旧进程继续完成已有连接。只应在部署验收通过后启用服务、切换 Nginx。

健康检查每 2 秒一次，连续失败 2 次摘除，连续成功 2 次恢复。优雅停机默认先等待 10 秒传播，再等待请求完成，最长 120 秒；systemd `TimeoutStopSec=180` 留出收尾时间。若调大排空期限，须同时调大 systemd/Docker 停止超时。不要用 `kill -9` 做滚动升级。

## 3. 增减节点与更新

- 新增：准备同版本 worker 和相同依赖配置，私网 readyz 通过后添加到 JSON，校验并 reload。
- 降低/增加流量：修改对应 `weight`（1–256），校验并 reload。
- 下线：设置 worker 的 `disabled:true` 并 reload，等待流式请求结束再停机；或者正常发送 SIGTERM，由应用先撤销 readiness 再排空。
- 更新：逐台摘流、替换程序、恢复 readiness；同一时间保留至少一个健康节点。替换 primary 时，管理/支付/本地任务会短时不可用。
- 全部节点退出集群后，才可恢复单机模式。禁止一个节点以单机模式连接仍在服务的集群 Redis。

定价、权限、计费记录、并发控制的最终数据仍归共享服务所有。生产切换前应验证真实客户端的普通生成、图片、Responses 续接、支付回调和插件账号；本地代理测试不替代这些验收。

## 本地测试

```sh
cd backend
go test -tags unit ./internal/config ./internal/server ./internal/service -run 'Test(Cluster|Node)'
HAPROXY_BIN=/path/to/haproxy go test ./internal/clusterlb -v
```

真实 HAProxy 测试会启动两个临时 HTTP 节点，检查管理面固定路由、最少连接、Responses 亲和性、SSE 首块、WebSocket 升级、故障摘除及恢复，不调用模型或支付服务。

参考：[HAProxy 3.2 配置手册](https://docs.haproxy.org/3.2/configuration.html)、[Go HTTP Shutdown 的连接处理](https://go.dev/src/net/http/server.go)。
