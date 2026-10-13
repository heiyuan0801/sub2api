# ChatGPT2API 生图 Chat 入口修复包

已合并官方 `Wei-Shaw/sub2api` 的 `v0.2.16`，上游基线提交为
`01842576b3d5c1e2545b5a99ad555d36e28ed306`。
定制构建版本为 `0.2.16-chat-image-fix.1`，保留先前的 Chat 生图兼容修复。

## 修复行为

客户端可以直接使用 `gpt-image-2`、`gpt-image-2.5`、
`gpt-image-2.5-flare`、`gpt-image-2.5-sunburst` 请求
`POST /v1/chat/completions`，不需要添加别名。

原先 OpenAI Chat 入口在选择账号之前统一拒绝 `gpt-image-*`。
此包移除了这项统一拒绝，并在传统调度、高级调度和实际转发处检查兼容性。
仅放行配置了自定义 Base URL、使用 OpenAI API Key、且解析出的上游协议为
Chat Completions 的账号。OAuth、默认官方 API、`api.openai.com` 和
Responses 上游仍不参与此类 Chat 生图请求。

普通文本模型以及原有 Images / Responses 入口没有修改。
生图分组权限和生图并发限制继续生效，账号模型映射也按实际上游模型检查。

调度缓存的账号投影保留 `base_url` 和 `openai_capabilities`，确保缓存路径也能
正确判断原生 Chat 生图兼容性。合并版同时保留 v0.2.16 新增的 Gemini 显式缓存
候选字段，元数据使用 `sched:meta:v3:` 命名空间。旧版投影会自动触发数据库
回退和重建，无需清空 Redis 或修改账号配置。

## 后台配置

1. 上游账号选择 **OpenAI / API Key**，填写真实 ChatGPT2API 地址与它的密钥。
2. 启用账号的文本端点能力，使其支持 **Chat Completions**。
3. 已有“自动探测：Chat Completions”可以继续使用。
   若自动探测结果为 Responses 或未探测，而确定要使用 ChatGPT2API 的 Chat 入口，
   则选择“强制 Chat Completions”。
4. 分组开启“允许生图”，并允许需要使用的实际模型名。

不要将 Sub2API 自身的公开地址填为其上游，否则会形成循环转发。

## 请求示例

```bash
curl "$SUB2API_URL/v1/chat/completions" \
  -H "Authorization: Bearer $SUB2API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-image-2","messages":[{"role":"user","content":"画一张山水风景插画"}],"stream":false}'
```

`stream: true` 使用 Chat Completions SSE 格式，图片内容由 ChatGPT2API 返回在
`choices[0].message.content` 或流式 `delta.content` 中。

此包负责转发，超分仍由 ChatGPT2API 执行。部署前面修改过的 ChatGPT2API 并开启
后台超分后，省略超分字段会继承其默认 4K；在请求根级设置
`"upscale_target": "2k"` 可选择 2K，设置 `"upscale": false` 可返回原图。
这些字段会保留并传给上游，不会转换成 Responses 请求。

## 包内容与部署

- `sub2api-0.2.16-chat-image-fix.1-source.zip`：合并后的源码和回归测试。
- `sub2api-0.2.16-chat-image-fix.1-linux-amd64.tar.gz`：Linux x86_64 可执行文件，已内嵌前端。
- `chat-image-fix.patch`：针对上述基线的源码补丁。
- `SHA256SUMS`：下载包的校验值。

在服务器执行 `uname -m`：`x86_64` 对应 amd64，`aarch64` 对应 arm64。
解压二进制包后，`sub2api` 即为程序，并附带该版本的 `resources` 和许可证。
保留原来的配置、环境变量、data 目录、PostgreSQL 和 Redis，按现有部署方式替换程序。

项目启动时会执行其原有数据库迁移。v0.2.16 新增迁移 242–245，涉及分组多语言
字段与 Gemini 缓存表，均为添加字段或表。升级前应备份数据库、现有程序、配置
和 Redis；保留二进制回滚入口。Chat 生图定制本身没有增加数据库迁移。

源码也可用项目原有 Dockerfile 构建：

```bash
docker build \
  --build-arg GOPROXY=https://proxy.golang.org,direct \
  --build-arg GOSUMDB=sum.golang.org \
  --build-arg VERSION=0.2.16-chat-image-fix.1 \
  -t sub2api:chat-image-fix .
```

在原有 Compose 配置中将 Sub2API 服务的镜像改为本地构建的
`sub2api:chat-image-fix`，保留原有数据卷和环境变量。
本机没有 Docker，本次未构建 Docker 镜像；提供的是源码和 Linux 二进制包。

## 验证

针对性回归覆盖原模型名、自动与强制 Chat 模式、传统与高级调度、普通 JSON、
SSE、超分参数透传、禁用生图分组和不兼容上游拒绝。
新增回归使用真实 Redis 缓存投影路径，覆盖旧缓存回退和重建、原生 Chat 放行、
强制 Chat、Responses 与官方地址拒绝、端点能力限制。
2026-10-13 的合并验证：130 个针对性测试项通过，后端完整 unit 测试与
`go vet -tags=unit ./...` 通过，前端 2,869 项测试和生产构建通过。
缓存回归同时验证上游 Gemini 显式缓存字段和旧版原生 Chat 投影回退。

原 `0.2.15-chat-image-fix.2` 已完成 Linux amd64 systemd 部署及真实 Chat 生图
验证。v0.2.16 的服务器部署状态应以对应部署记录为准，不沿用旧版验收结果。
直接使用官方后台自动更新会替换定制二进制，后续升级仍需先合并保留这些修复。

## 源码构建

```bash
cd frontend
pnpm install --frozen-lockfile
pnpm run build
cd ../backend
CGO_ENABLED=0 go build -tags embed -trimpath -o sub2api ./cmd/server
go test -tags=unit ./internal/service ./internal/handler \
  -run 'GPTImageChat|ChatCompletionsImageModel|NonOpenAIChatCompletions'
```

所需工具链以仓库为准：Go 1.27.2、Node 24、pnpm 9。
