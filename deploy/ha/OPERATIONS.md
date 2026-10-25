# 集群运维与验收

这一层在现有双应用节点 + 独立仲裁的 HA 部署上增加节点控制、请求核对、任务恢复和运维工具。入口是管理员菜单「集群管理」(`/admin/cluster`)。单机模式显示未启用；所有接口继承管理员认证、审计、合规确认和面板限流。不会修改用户的合规确认记录。

## 能力和边界

| 功能 | 行为 |
|---|---|
| 节点与告警 | 每 3 秒更新节点心跳、角色、版本、配置摘要、数据库/Redis、复制、并发、堆内存和应用所在文件系统可用空间。心跳超过 20 秒视为离线。主节点每 15 秒生成告警及恢复记录。 |
| 通知 | 复用运维监控的邮件收件人、最低严重程度、静默策略、频率限制和通知模板；发送状态持久化。未开启邮件时只保留页面/日志告警。外部邮件投递不是恰好一次，故障边界可能重复。 |
| 请求账务核对 | HA 网关在调用上游前落请求记录；收到用量后，在原有原子扣费前保存计费证据。对照去重账本和用量日志，保留精确金额字符串。未知结果、漏结算、漏用量、金额差异需要人工确认；没有自动补扣、退款或重放生成请求。 |
| 异步生图 | HA 下结果/状态持久化到 PostgreSQL，输出仍使用已有共享 S3/R2。失去执行进程且未保存结果的任务标记执行中断，并将原请求加入核对列表。不会重复生成未知结果。 |
| 批量生图 | 根据数据库中已保存的 provider job ID 恢复丢失的 Redis 队列成员，保留已在执行/延迟/持锁的任务；仅恢复 submitted/running/indexing/settling，不自动重发 created/uploading。 |
| 节点控制 | 共享版本号防止覆盖别人刚保存的控制。限制最后一个健康节点进入维护；排空后主节点释放租约并变为 worker。每节点请求上限 1–20000，权重 1–256。 |
| 保留策略 | 普通已完成/已核对请求保留 30 天；未解决的请求与异常证据不清除。已匹配且无关联请求的证据保留 90 天；运维事件保留 90 天。每轮清理有批次上限。 |

工具要求 Python 3.10+、PostgreSQL 17+；数据库监控账号需要 `pg_read_all_stats`（或 `pg_monitor`）查看完整复制信息，没有权限时页面显示未知。

计费证据不保存 API Key 明文、请求正文或生成内容。原有扣费事务和去重机制仍负责资金变动。批量任务及支付订单保留各自的账本，不在普通网关请求表中重复结算。

本地故障模拟不能证明两台物理机与公网入口的生产可用性。生产还必须验证独立仲裁、跨地域网络延迟、外部入口切换、Linux watchdog、存储与证书、实际备份恢复。中途断掉的 SSE/WebSocket 不能转移到另一台；未知生成结果不能盲目重试。

## 安装入口控制代理

1. 使用新版 `render.py` 生成配置。HAProxy 管理 socket 仅在 `/run/sub2api-proxy/admin.sock`，不开放 TCP 管理端口。
2. 安装 `sub2api-ha-proxy.service` 与 `sub2api-ha-controls.service`。HAProxy 以 `haproxy` 用户、`sub2api` 组运行；controls 以 `sub2api` 用户运行。
3. 将 `proxy_agent.py` 放到 unit 指定目录，将 `nodes.json` 放到 `/etc/sub2api-ha/`。该文件列出允许控制的节点 ID。
4. 启动 controls 后，页面应显示「配置已生效」。代理每轮重放权重/连接上限，HAProxy 重启也能恢复控制。旧快照、未知节点和命令失败不会写生效确认。维护状态由应用健康检查负责。

所有应用节点和 runner 都升级到支持维护协议的版本后，才使用自动滚动工具。首次引入这些组件仍按照已有 HA 安装/维护流程进行。

## 部署检查和滚动替换

`operations.py` 不负责 SSH 或下载，不接受任意服务名。它只操作运行该命令的机器上的 `sub2api-ha.service`。API 地址必须使用能访问存活主节点的 HTTPS 入口，或者本地回环 HTTP。

```sh
# SUB2API_ADMIN_TOKEN 从当前管理员授权环境读取，勿写入 Git 或命令行。
python3 deploy/ha/operations.py doctor --api https://your-domain \
  --inventory /etc/sub2api-ha/inventory.json --tls /etc/sub2api-ha/tls

# 默认只展示计划，需提供可信构建产物的 SHA-256 和预期版本。
python3 deploy/ha/operations.py upgrade --api https://your-domain --node hk \
  --artifact /opt/sub2api/staging/sub2api --installed /opt/sub2api/sub2api \
  --sha256 YOUR_VERIFIED_SHA256 --version YOUR_VERSION
# 检查迁移向后兼容后，用相同参数加 --apply --schema-compatible 执行。
```

工具保留旧二进制，要求节点在线且未在维护中；排空并等待主角色转移后替换；新实例、版本及依赖心跳确认后才恢复流量。启动失败自动恢复旧二进制并保持维护状态，供管理员确认后恢复。备份文件不会自动删除。它不回滚数据库，也不能保证不可逆迁移后的旧二进制仍兼容。一次只更新一台，确认后再更新另一台。主角色转移会有短暂管理入口收敛时间。

## 连续备份与时间点恢复

现有后台逻辑备份/S3 定时及保留设置继续有效。新增物理备份工具用于 WAL/PITR；逻辑 `pg_dump` 不能代替它。参考 [PostgreSQL 17 PITR 文档](https://www.postgresql.org/docs/17/continuous-archiving.html)。

在独立备份存储挂载点，为本集群专用地创建 `BACKUP_ROOT` 和其 `wal/`，仅 PostgreSQL 账号可读写。不要使用活跃 PGDATA、`pg_wal` 或与其他项目共享的目录。两台数据库节点必须都能访问同一份 WAL 归档。网络文件系统/对象存储的可用性需另行验证；不把本机目录视为异机备份。

PostgreSQL 需要启用 `archive_mode=on`，归档命令示例：

```text
/usr/bin/python3 /opt/sub2api/deploy/ha/backup.py archive --source %p --wal-dir /mnt/sub2api-backup/wal --name %f
```

归档使用同步写入和不可覆盖提交，已有同名不同内容的 WAL 会报错。通过 Patroni 配置设置归档参数，按 PostgreSQL 要求进行滚动重启；不要只修改被 Patroni 管理的临时配置文件。

`backup.env`（权限 0600，不入 Git）配置 `PGHOST`、`PGPORT`、`PGUSER`、`PGPASSFILE`、`PG_BIN` 和 `BACKUP_ROOT`。复制账号须有 replication 权限，pg_hba 允许备份连接。安装对应 backup service/timer 后，每日生成 `base-TIMESTAMP`，调用 `pg_verifybackup`，完成后写入带 manifest 校验和的 `.verified.json`。

```sh
python3 deploy/ha/backup.py basebackup --pg-bin /usr/lib/postgresql/17/bin \
  --destination /mnt/sub2api-backup/base-YYYYMMDDTHHMMSS
python3 deploy/ha/backup.py prepare-restore --pg-bin /usr/lib/postgresql/17/bin \
  --base /mnt/sub2api-backup/base-YYYYMMDDTHHMMSS \
  --destination /srv/sub2api-restore-drill \
  --wal-dir /mnt/sub2api-backup/wal --target-time '2026-10-25T01:00:00+00:00' --port 29432
```

恢复仅接受新目录并验证备份，重写监听/端口/认证路径，清除生产复制连接配置，默认不启动。达到 PITR 目标后暂停回放；人工确认账本、用户余额、图片对象引用，再决定后续操作。存在表空间/符号链接时工具拒绝自动恢复，须单独规划映射。恢复目录和备份包含业务敏感数据，保持 0700 并清理不再需要的演练副本。

## 保留和磁盘日志

```sh
# 先查看计划；保留至少两份完整校验通过的基线。
python3 deploy/ha/retention.py --root /mnt/sub2api-backup \
  --pg-bin /usr/lib/postgresql/17/bin --days 14 --keep 2
# 审阅后加 --apply。对应 retention timer 为可选自动清理入口。
python3 deploy/ha/backup.py disk --destination /mnt/sub2api-backup
```

只清理本工具创建并验证的 `base-*`，以及早于所有保留基线所需起始 LSN 的 WAL 整段。保留 timeline history；未知、未完成或校验失败的备份会阻止清理。PITR 最早可恢复时间由最旧保留基线决定。长期归档请另外保留整套基线及其 WAL，不能只保留一份旧目录并继续清 WAL。

`sub2api-ha.logrotate` 只覆盖 `/var/log/sub2api-ha/*.log`，不针对 PostgreSQL data 目录。copytruncate 有极小的轮转丢日志窗口；审计记录保留在数据库。应用 stdout 使用 journald 时，由运维按整机策略设置 journald 容量限制，不能仅轮转文件日志。PostgreSQL 可使用 `logging_collector`、`log_rotation_age=1d`、`log_rotation_size=100MB` 配合独立日志目录保留策略。不要关闭错误日志或删除活跃 `pg_wal`。

## 可重复的本地验收

- `go test -tags=unit ./...`；关键集群包 `go test -race`。
- `SUB2API_CLUSTER_TEST_DSN=... go test ./internal/clusterops -run TestClusterOperationsPostgres`：在独立 schema 检查控制冲突、故障请求/异步生图、账务差异与人工核对。
- `python3 -m unittest discover -s deploy/ha -p 'test_*.py'`。
- 初始化 `lab.py` 的独立本地环境后：`python3 deploy/ha/lab_ops.py --dir /absolute/local/lab`。验证真实并发、账本金额、代理权重、限流、流式排空、滚动替换和失败回滚；控制写入使用隔离测试 SQL，不替用户接受管理合规条款。
- `python3 deploy/ha/lab.py test --dir /absolute/local/lab`：分别停止 US/HK 整组进程、仲裁失联、恢复路由与计费。
- `python3 deploy/ha/lab_restore.py --dir /new/isolated/path --pg-bin /path/to/pg17/bin`：实际创建 PostgreSQL、归档 WAL、物理备份、恢复到指定时间，检查后续交易被排除；结束自动停止演练数据库。
