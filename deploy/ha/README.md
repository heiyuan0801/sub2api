# 两个业务故障域的高可用

这是 `codex/multi-node-load-balancing` 的可选 HA 部署，沿用原数据库业务结构。启用前需要完整演练与数据迁移；生成配置不会部署、改 DNS、购买服务或覆盖现有数据。

## 拓扑与承诺边界

```text
用户 → 同一域名的两个地址 / 可选独立健康负载均衡（两边同时接流量）
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

同一域名直接配置两条 A/AAAA 记录，可以让两个 IP 都提供服务。DNS 轮询不保证均分或故障自动摘除。需要自动避开故障入口时，外部健康负载均衡必须独立于两台业务机，例如托管的 Cloudflare Load Balancing。随附三个 `cloudflare-*.example.json` 是 API 请求体参考：创建一个 HTTPS monitor、分别创建香港/美国 pool，再将两个 pool 加入 `default_pools`，使用 `steering_policy: random` 和相等的 `pool_weights`，使两个健康入口都参与分流。替换域名、IP 和返回的 ID 后由运维部署；本工具不调用 Cloudflare 或创建付费资源。60 秒检查间隔加重试可能让外部恢复耗时超过本地测试，不能直接沿用本地秒数作为生产 SLA。仅配置两个 A 记录不等同于健康自动切换。

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

集群管理页按节点显示系统 CPU 使用率与逻辑核心数、1/5/15 分钟负载、内存使用/总量、应用磁盘使用率、进程 RSS/Go 堆、协程数、运行时长和累计请求/5xx/拒绝数量。CPU 和内存取操作系统可见的系统视图，容器内可能是宿主机，并非容器配额；内存用量按总量减可用量计算。本机双节点实验运行在同一台物理机，因此两张卡片的系统指标会相近，应用内存和计数分别采集。

资源采集在独立后台循环中每 5 秒执行，不阻塞节点心跳或业务请求；通过现有 `cluster_nodes.snapshot` JSON 保存，无新增数据库迁移。CPU 需要两个采样点，采集失败/老节点缺少的指标显示“暂无数据”。页面每 10 秒刷新；心跳或资源采样超过 20 秒会标记过期，避免显示旧的健康状态。

连接区显示心跳、数据库写连接和 Redis 响应耗时、Redis 复制连接、HA 租约证明及入口配置确认。数据库耗时包含连接池等待与查询处理，并非纯网络 RTT；租约证明和入口确认也不代表公网入口的端到端可达性。数据库写入不可用时心跳无法落库，最终表现为心跳过期，不能用旧快照断言数据库当前健康。

集群事件、告警和部署检查按当前语言显示说明，中文模式同时翻译节点角色、请求状态和 Redis 主从角色。历史日志在展示时转换，原始内容保留在可展开详情；未识别的格式会提示查看原文。部署检查的“需人工验证”和“无法确认”不会被视为通过。

节点控制、请求账务核对、异步任务恢复、滚动升级、物理备份/PITR、保留和日志策略见 [OPERATIONS.md](OPERATIONS.md)。所有脚本均需显式执行，不会在安装前自行连接或更新服务器。

## 双入口完善与验收

`inventory.example.json` 的 `ingress` 是同一个域名和两台公网 IP。生成器额外输出双 DNS 记录、两个相同域名的 Nginx HTTPS 站点、检查清单以及可选 Cloudflare 双活请求体。两种入口方式择一部署；默认不修改 DNS，不申请证书，也不连接服务器。Nginx 模板使用 `/etc/letsencrypt/live/<domain>/`，需先在两边签发并持续续期；DNS-01 可避免双 A 下 HTTP-01 验证落到另一台。Cloudflare/CDN 模式必须另设受信任来源的真实客户端 IP，不能直接信任公网传入的 XFF。

`capacity` 显式列出 PostgreSQL 最大连接数、运维保留、每节点数据库/Redis 池和应用请求上限。生成器拒绝两节点池合计超出预算；Redis 额外保留 128 个连接给 Sentinel、运维和恢复。管理页部署检查也核对在线应用的数据库池合计是否超过数据库上限的 80%。预算只是连接上限，不是对真实业务容量的承诺。普通 gateway 使用 30 秒 `slowstart` 恢复；Responses 仍使用稳定一致性哈希，不因预热步进反复移动会话。两台入口必须同步同一份节点清单和权重；不要通过重放 POST 补偿错误。

`verify.py` 直接连接每个指定 IP，用域名执行 Host/SNI 与系统 CA 验证，依次检查 `/ingress-ready`、`/health`、`/home`。拒绝跳转，不发送账号、支付或生成请求；明文 HTTP 仅可在 `--local-lab` 的回环地址使用。

```sh
python3 deploy/ha/verify.py ingress --inventory /etc/sub2api-ha/ingress.json   --output /run/sub2api-ha/proof.json.verification
```

输出只含检查时间、入口身份、状态、总检查耗时和证书到期时间。任一入口失败也会写入结果并以非零状态退出。应用读取自己 proof 文件旁的 `.verification`，通过节点快照展示；5 分钟后不再显示当前通过。可选 `sub2api-ha-verify.service/timer` 每分钟运行一次，从业务机执行只证明该机视角的 HTTPS 可达性；还应从独立网络运行同一工具。此报告不自动通过物理隔离、独立仲裁或故障切换验收。

发布文件一致性：在两台机器分别对显式选择的应用、runner、插件二进制、公共静态文件生成清单（不要包含密钥与私密配置），将两个清单带到检查机器。清单须在 15 分钟内采集；未提供则显示未知。检查只覆盖所选文件，运行配置摘要另由现有节点同步检查负责。

```sh
python3 deploy/ha/verify.py manifest --root /opt/sub2api   --file sub2api --file cluster-runner --file plugins/example-plugin   --output /secure/node-us-manifest.json
python3 deploy/ha/verify.py ingress --inventory /secure/ingress.json   --manifest /secure/node-us-manifest.json --manifest /secure/node-hk-manifest.json   --restore-result /secure/isolated-restore/results.json   --output /run/sub2api-ha/proof.json.verification
```

`--restore-result` 只接收 `lab_restore.py` 实际完成后的报告，显示其原始演练时间和“本地实验”，不会把本地合成账本恢复当作生产业务数据已验收。省略此参数时恢复状态未知。定时命令如果不包含清单/恢复结果，会将这两项更新为未知，不沿用旧的通过状态。

性能面板包括当前 RPS、近 60 秒已成功入账样本的首字 P95 桶上界、上游 token 缓存读取比例、数据库池占用与平均等待、Redis 池连接/超时、本入口排队/活跃数及代理字节。无样本显示未知；超过最高首字桶显示大于 600 秒。流量只统计 HAProxy 到应用，不包含复制链路，也不是精确物理网络计量。RPS 是接受请求速率，不是生成成功率；数据库等待是刚过去采样区间内的均值。

热点计费证据写入在新请求时由两次数据库往返减为一次；重复请求才查询旧指纹，仍拒绝指纹冲突，不覆盖原证据。心跳的角色与复制查询合并为一次。不要为降低跨洋延迟关闭账务耐久性；真实美港吞吐、上游缓存率和最终用户首字时间仍需部署后压测。

本地验收命令在原运维测试之外增加：

```sh
python3 -m unittest discover -s deploy/ha -p 'test_*.py'
HAPROXY_BIN=/path/to/haproxy go -C backend test -race ./internal/clusterlb ./internal/clusterops
# 显式隔离测试库；测试为自身创建/删除独立 schema：
SUB2API_CLUSTER_TEST_DSN=... go -C backend test -race ./internal/repository -run 'TestCluster.*Postgres'
# 已启动的隔离实验：双入口登录、会话、24 并发和账务/指标核对
/absolute/venv/bin/python deploy/ha/lab_dual_ingress.py --dir /absolute/new-ha-lab   --restore-result /absolute/isolated-restore/results.json
```

覆盖真实双入口的会话一致性、SSE/WebSocket、失败不重放，真实 PostgreSQL 两个独立连接池竞争同一账单、证据落库未扣费和扣费后未记用量的中断窗口、支付重复通知回归、TLS 域名校验、文件漂移、恢复报告和过期状态。完整 `lab_ops.py`、`lab.py test` 与 `lab_restore.py` 继续负责双应用并发、限流排空、整组进程故障、Redis/数据库接管、失去仲裁和隔离恢复演练。

### 运行配置、持续检查与历史指标

新版本在已有 `settings` 表登记 `cluster_security_contract_v1`，不增加数据库表。
第一个配置完整的节点建立共享配置指纹；JWT 密钥、TOTP 加密密钥、数据库名、
Redis DB 编号、HA 与严格计费模式必须一致。节点 ID、角色、地址与容量允许不同。
每次心跳核对指纹，不一致的节点停止接收业务请求，并向守护进程请求排空。
管理页会保留其心跳和中文告警，以便识别和修复；不要把在线等同于可接流量。
指纹不证明两个不同地址连接了同一套 Redis，应继续保留故障切换与跨节点会话验收。

首次升级先在维护窗口核对两端配置，再逐个替换并检查 `/ready` 与 `/ingress-ready`。
配置错误时修复错误节点，**不要删除集群基准来迁就错误密钥**。
有意轮换 JWT 或改变关键配置时，需要全部节点排空并停止，由运维在确认新配置一致后，
在事务中删除该设置键，再仅启动一个可信节点建立新基准，然后逐个加入其他节点。
TOTP 密钥用于已有数据加密，不能仅修改配置；轮换需要独立的数据重加密方案。
旧节点未提供校验结果时会显示无法确认，不计入“可排空其他节点”的健康余量。

`verify.py ingress` 每分钟运行时，会用前一个未过期报告累计失败次数。
连续 3 次入口失败触发告警，恢复后解除；报告超过 5 分钟、证书不足 14 天、
发布文件不一致、恢复演练超过 30 天、共享存储验证超过 24 小时也进入现有告警/恢复通知。
主节点沿用已配置的运维邮件策略；没有配置收件人不会凭空发送通知。
主机启动有 5 分钟检查配置宽限期，不能据此判断备份已配置。

WAL 每 30 秒采样：最近失败、待归档 `.ready` 文件数、最早积压时间。
数据库角色需要读取 `pg_stat_archiver` 及 `pg_ls_archive_statusdir()` 的权限；
无权限显示未知。已成功归档且没有积压的空闲数据库，不会因上次归档时间较旧误报。
积压超过 5 分钟会告警；这只能验证归档交付，恢复可用性仍需 `lab_restore.py` 等隔离演练。

管理页新增 24 小时/7 天历史。Redis 按节点每 5 分钟保留一个近期采样快照，
最多 2016 个点，停止采样后键在 8 天自动过期；查询只返回最近 7 天。
它不是 5 分钟区间平均值，也不是账务记录。Redis 故障时历史可能丢失，前端保留空缺。
当前完成请求平均耗时包括响应流式传输；鉴权、路由排队、上游阶段来自已有处理器采样，
各自有样本数、覆盖范围可能不同且可能重叠，不能相加推导端到端耗时。

### 共享对象存储的双节点交叉验证

构建 `go build -o cluster-storage-probe ./cmd/cluster-storage-probe`（在 backend 中执行）。
工具使用现有 AWS SDK，每次只创建一个 1 KiB 随机 canary，限定路径
`.sub2api-ha-probe/<新 UUID>/<节点 ID>`，条件写入不会覆盖已有对象。
凭据只通过各主机私有环境配置提供：
`SUB2API_PROBE_S3_ENDPOINT`、`SUB2API_PROBE_S3_REGION`、`SUB2API_PROBE_S3_BUCKET`、
`SUB2API_PROBE_S3_ACCESS_KEY`、`SUB2API_PROBE_S3_SECRET_KEY`。
使用实际应用的存储配置和权限；不要把凭据放进命令参数或 Git。
默认要求 HTTPS；`--local-lab` 仅允许回环 HTTP，并将报告明确标为本地实验。

1. 在两端用同一新 UUID，分别执行（将变量替换为实际值）：
   `cluster-storage-probe --action write --write-probe --node us --challenge UUID --output us-write.json`，
   香港端使用 `--node hk --output hk-write.json`。
2. 交换无密钥的写入回执。美国端执行
   `cluster-storage-probe --action read --node us --receipt hk-write.json --output us-read.json`，
   香港端交叉读取美国回执生成 `hk-read.json`。工具校验内容摘要、挑战、时间及主机标识。
3. 汇总四份回执（失败时也执行汇总，生成失败报告，不能沿用旧的成功报告）：
   `python3 verify.py storage --receipt us-write.json --receipt hk-write.json --receipt us-read.json --receipt hk-read.json --output storage-result.json`。
4. `verify.py ingress` 增加 `--storage-result storage-result.json`，把结果写入两端私有
   `.verification` 文件。存储交叉检查建议每 12 小时执行一次；接入实际机器调度属于部署步骤。
5. 各端用自己的写入回执清理：
   `cluster-storage-probe --action cleanup --write-probe --receipt us-write.json`。
   不列出或清空桶；中途故障留下的 canary 可给专用前缀设置 2 天生命周期。

主机标识仅为主机名摘要，两个不同摘要不构成物理独立性证明。
本检查验证一个 canary 的共享读写，不替代已有图片/批量文件的迁移和引用核对。

### 本地网络与规模验收

`python3 deploy/ha/lab_network.py --dir <隔离实验目录> --output network-results.json --soak-seconds 120`
通过随机回环端口代理现有的两个本地入口，测试 8/24/48 并发、双向读取延迟、抖动、
转发前断连，核对成功响应、余额、usage 和计费证据；不会自动重试生成 POST。
随后进行有上限的持续请求。仅使用 `lab.py` 的虚拟模型和测试账户。
这不是系统级丢包模拟，也不施加 PostgreSQL 同步复制延迟，不能据此宣称美港跨机容量。
实际部署需在隔离环境分别测入口路径和数据库复制 RTT，并完成双机/仲裁失效演练。

`SUB2API_CLUSTER_TEST_DSN` 指向隔离测试 PostgreSQL，
`SUB2API_CLUSTER_SCALE_ROWS=1000000 go test ./internal/clusterops -run TestClusterScalePostgres -v`
会创建并删除随机 schema，生成请求、用量与计费证据，实际执行最近/待核对分页和
`EXPLAIN (ANALYZE, BUFFERS)`。CI 使用 10 万行，本地可测试 100 万至 200 万行。
既有索引可满足当前验收时不新增迁移；生产慢查询仍应按真实数据分布复核。
