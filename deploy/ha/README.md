# 两个业务故障域的高可用

这是 `codex/multi-node-load-balancing` 的可选 HA 部署，沿用原数据库业务结构。启用前需要完整演练与数据迁移；生成配置不会部署、改 DNS、购买服务或覆盖现有数据。

## 拓扑与承诺边界

```text
用户 → 独立托管的全局负载均衡（优先香港，失败后美国）
        ├─ 香港：Nginx TLS → HAProxy → 两边 Sub2API
        └─ 美国：Nginx TLS → HAProxy → 两边 Sub2API

美国：PostgreSQL/Patroni + Redis + etcd + Sentinel + cluster-runner
香港：PostgreSQL/Patroni + Redis + etcd + Sentinel + cluster-runner
第三故障域：etcd + Sentinel（不存业务 PostgreSQL，不承载唯一入口）
独立对象存储：图片结果、需要跨节点读取的持久文件
```

目标：美国或香港任一整机停止后，另一边在健康检测、数据库/Redis 选举和应用租约切换后继续服务；后台、登录和支付回调会跟随当选应用主节点。第三仲裁点单独故障时，美国与香港仍有多数票。两个仲裁成员同时不可达时拒绝服务，避免双主。第三点必须是独立机器/故障域，把第三进程放到其中一台业务机上不成立。

这不是零中断或零数据损失的通用保证。已建立的 SSE/WebSocket 无法迁移；故障中的请求可能得到连接错误、503 或明确的会话恢复错误。入口不重放 POST。客户端应按原 API 的幂等语义重试；支付以签名、订单状态和数据库事务去重，禁止因切换自动造补单。

PostgreSQL 采用 `synchronous_mode: true`、一个同步副本、`synchronous_mode_strict: false`。正常双机时等待同步副本；单机失败后允许唯一存活主库继续提交。故障后到副本重新同步完成之间只有一份新数据，不能承受第二次数据盘丢失。取消/超时的事务结果也可能不确定。跨美港同步提交会增加网络往返延迟，真实吞吐必须另测。

Redis Sentinel 使用异步复制。故障切换可能丢失最近的缓存、会话元数据、短期并发槽、余额预留或队列更新；AOF 不等于同步复制。数据库账本为准，缓存会重建，连接专属会话可能需要完整上下文。HA 模式强制严格预留并同步执行用量任务，缩小进程内排队窗口，但上游完成、客户端接收和本地入账不是一个分布式事务，故障瞬间的在途账务仍需对账，不能宣称严格零漏计/零超额。若这些临时状态必须零丢失，需要额外的共识存储/持久任务协议，不能仅靠 Sentinel 保证。

## 新增实现

- `cluster-runner` 使用三个 etcd 端点、30 秒租约和线性一致事务，决定子进程的 `primary/worker` 角色。角色切换会重启该节点应用，其他健康节点继续处理请求。
- 续租/仲裁检查失败先撤销本地证明、停止子进程，再释放租约。角色证明有效期 5 秒；应用每 250 毫秒独立检查，runner 意外死亡时也会停止后台任务。每个 HTTP 请求和 readiness 都校验证明。
- 生产 runner 强制 mTLS 和 systemd 10 秒 watchdog；`KillMode=control-group` 清理子进程和插件进程。Patroni 必须配 `watchdog.mode=required`、可工作的 Linux `/dev/watchdog`，不满足时不得上线。单纯软件租约不能防止任意进程暂停、系统时钟异常或虚拟机冻结，生产需要验证节点 fencing，不能复制本地 `local_lab` 选项。
- Redis 使用官方客户端 Sentinel 自动发现、切换连接；Sentinel 与数据节点可使用不同认证，TLS 校验发现后的地址。关闭 HA Redis 客户端的隐式命令重试，避免不确定写操作被重复执行。
- HAProxy 的管理 backend 包含所有候选节点，以 `/readyz?node=...&role=primary` 只选择当选主节点；普通请求仍分流。每台入口都有数据库 writer 代理，通过 Patroni `/primary` 选主，主库下线时关闭旧连接。
- `/ingress-ready` 同时检查管理、普通推理、Responses 三类路由都有可用节点，仅公开 `ready/unavailable`，供外部入口健康检查；`/readyz` 仍对外屏蔽。

## 生产准备

1. 准备美国、香港和第三故障域之间的私有加密隧道及防火墙。只允许三点访问 etcd 2379/2380、Sentinel 26379；只允许业务节点访问 PostgreSQL 5432、Redis 6379、Patroni 8008、应用 8080。HAProxy 8088 和 writer 15432 只监听 loopback。不要开放数据服务到公网。
2. 两边安装相同 PostgreSQL 主版本、Patroni、Redis/Sentinel、etcd、HAProxy 3.2+、Sub2API 和 runner。版本以本地验证记录为准。每个 etcd 节点证书包含其隧道 IP SAN；客户端证书单独签发，私钥与配置权限 600。
3. 原安装的 JWT、TOTP、其他加密密钥、数据库名/账号、Redis DB 编号必须一致；沿用数据库持久设置。不要用示例覆盖现有 `config.yaml`。先在离线副本演练备份恢复、基线校验和主备复制，再另行批准生产切换。本次没有业务 schema 迁移，但将单机数据库改为主备是数据运维操作，仍需要备份。
4. 插件二进制、版本、配置和静态文件部署到两边，并验证相同哈希。后台安装/升级插件只改变当前主节点，不能作为双机发布；应通过同一部署包同步。临时文件和进行中的插件会话可能中断。异步图片使用已有 S3 配置，必须是独立高可用服务；单台业务机上的 MinIO/NFS 不能解决该机故障。批量任务输入/结果等其他本地持久目录需迁移到跨故障域存储，未完成前不得把这些能力列入已验证 HA 范围。
5. 系统时间使用 NTP；生产禁用 `local_lab`。Patroni watchdog 应在可控环境验证真实重置行为。给数据库、Redis、watchdog 与应用进程保留 CPU/内存，避免过载导致误切换。

构建：

```sh
cd backend
go build -o /secure/build/cluster-runner ./cmd/cluster-runner
go build -o /secure/build/cluster-config ./cmd/cluster-config
go build -tags embed -o /secure/build/sub2api ./cmd/server
```

分支的 GitHub CI 会构建 Linux amd64 的 `sub2api`、`cluster-runner` 和 `cluster-config`，连同本目录模板、版本信息与二进制校验和一起保存为 Actions 构建产物。CI 用三个独立 etcd 进程验证单主租约与接管，用真实 HAProxy 验证分流、会话、流式请求与健康切换，并运行部署清单校验。它不替代下面的完整故障演练或生产验收，也不会发布发行版或更新服务器。

用 `inventory.example.json` 填实际隧道地址。`render.py` 从受保护环境读取 `HA_DB_PASSWORD`、`HA_REPLICATION_PASSWORD`、`HA_REDIS_PASSWORD`、`HA_SENTINEL_PASSWORD`，要求 32 位以上生成密码。不要把密码写入命令历史或提交仓库。生成器只接受新建的仓库外输出目录：

```sh
python3 deploy/ha/render.py --inventory /secure/ha/inventory.json \
  --output /secure/ha/review-bundle --generator /secure/build/cluster-config
```

每节点会得到 etcd、Sentinel 配置；业务节点还得到 Patroni、Redis、HAProxy、runner 和 `app-overlay.json`。后者仅包含应合并进现有配置的字段。数据库口令指 Patroni 管理用户，应用应继续使用现有最小权限用户。bootstrap 仅用于新集群；对已有数据的接管需要按 Patroni 流程单独演练。Sentinel/Redis 会重写自身配置，恢复节点时不能覆盖它们已经保存的主从拓扑。

服务使用发行版的 etcd/Redis/Sentinel/Patroni 服务单元，分别配置上述路径、数据目录所有者及开机启动。参考应用 `sub2api-ha.service` 和代理 `sub2api-ha-proxy.service`，按实际二进制路径调整。runner 与原来的单机 Sub2API service 不能同时启动。节点证书、proof 目录归应用用户；数据目录分别归各服务用户。设置日志轮转及磁盘配额，保留可恢复的原服务配置。

## 双入口

两边 Nginx 使用相同域名的有效证书，参照 `../cluster/nginx-location.conf` 转发到各自 HAProxy。为 `/ingress-ready` 禁止缓存；禁止 CDN 缓存登录、管理、支付和生成接口，保留 WebSocket 与 SSE 支持及所需长连接超时。

外部健康负载均衡必须独立于两台业务机，例如托管的 Cloudflare Load Balancing。随附三个 `cloudflare-*.example.json` 是 API 请求体参考：创建一个 HTTPS monitor、分别创建香港/美国 pool，再将 `default_pools` 按香港、美国排序，`steering_policy: off`。替换域名、IP 和返回的 ID 后由运维部署；本工具不调用 Cloudflare 或创建付费资源。60 秒检查间隔加重试可能让外部恢复耗时超过本地测试，不能直接沿用本地秒数作为生产 SLA。仅配置两个 A 记录不等同于健康自动切换。

## 本地真实故障实验

`lab.py` 在独立目录和固定 loopback 高位端口运行三个真实 etcd 成员、两个 Patroni/PostgreSQL、Redis 主从和三个 Sentinel、两份应用+runner、两个入口以及一个模拟外部入口。模型提供方是 `lab_fixture.py`，不访问真实模型或支付商。

在 Patroni 的 Python 环境中执行，显式指定工具路径：

```sh
export SUB2API_HA_APP=/absolute/build/sub2api
export SUB2API_HA_RUNNER=/absolute/build/cluster-runner
export SUB2API_HA_GENERATOR=/absolute/build/cluster-config
export SUB2API_HA_ETCD=/absolute/tools/etcd
export SUB2API_HA_PATRONI=/absolute/venv/bin/patroni
export SUB2API_HA_PG_BIN=/absolute/postgresql/bin
export SUB2API_HA_REDIS=/absolute/redis-server
export SUB2API_HA_HAPROXY=/absolute/haproxy
/absolute/venv/bin/python deploy/ha/lab.py start --dir /absolute/new-ha-lab
/absolute/venv/bin/python deploy/ha/lab.py test --dir /absolute/new-ha-lab
/absolute/venv/bin/python deploy/ha/lab.py test-witness --dir /absolute/new-ha-lab
/absolute/venv/bin/python deploy/ha/lab.py status --dir /absolute/new-ha-lab
# 测试完成后停止本实验的进程，保留数据与结果：
/absolute/venv/bin/python deploy/ha/lab.py stop --dir /absolute/new-ha-lab
```

入口 `127.0.0.1:28180`。每次创建须空目录和空闲端口；凭证仅保存在实验目录，目录权限 700。测试组按“整台主机”的故障域杀掉应用、Patroni/PostgreSQL、Redis、入口、Sentinel、etcd，再检查恢复。最后停止两个 etcd 成员验证失去多数票时两应用都被隔离。结果写 `results.json`。

本机实验不等于真实物理机断电、跨洋网络分区或 Cloudflare、硬件 watchdog、对象存储、真实支付回调的验证。上线门槛包括：实机 fencing、外部域名切换、数据库复制恢复、真实账号请求、支付回调重试/去重、异步任务及文件可读性逐项验收。

参考：[Patroni 复制语义](https://patroni.readthedocs.io/en/latest/replication_modes.html)、[watchdog](https://patroni.readthedocs.io/en/latest/watchdog.html)、[Patroni 健康接口](https://patroni.readthedocs.io/en/latest/rest_api.html)、[Redis Sentinel](https://redis.io/docs/latest/operate/oss_and_stack/management/sentinel/)、[Cloudflare LB API](https://developers.cloudflare.com/api/resources/load_balancers/methods/create/)。

## 集群运维

节点控制、请求账务核对、异步任务恢复、滚动升级、物理备份/PITR、保留和日志策略见 [OPERATIONS.md](OPERATIONS.md)。所有脚本均需显式执行，不会在安装前自行连接或更新服务器。
