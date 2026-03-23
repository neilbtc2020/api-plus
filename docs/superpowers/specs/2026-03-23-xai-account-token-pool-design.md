# xAI Account Token Pool Design

## Goal
在不新增渠道类型的前提下，为现有 `xAI` 渠道增加一种 `账号 Token` 鉴权模式，支持管理员以“每行一个 token”的方式录入 Grok 网页账号 token，并复用现有多 key 能力实现账号池调度、`401/403` 永久禁用、`429` 临时冷却与自动恢复，同时保持现有 `xAI API Key` 模式零回归。

## Architecture
- 渠道类型仍然保持为 `xAI`，不新增 `Grok` 或 `Grok2API` 渠道类型。
- 在 `xAI` 渠道的 `OtherSettings` 中新增显式配置 `xai_auth_mode`，用于区分 `api_key` 与 `account_token` 两种鉴权模式。
- 两种模式共用同一个 `xAI` 渠道能力入口与模型清单；差异仅体现在上游请求 URL / Header / Cookie 组装，以及 token 池调度与错误处理。
- 账号池不新增数据库表，直接复用现有 `Channel.Key`、`ChannelInfo`、`OtherInfo`：
  - `Key`：存储原始 token 列表（每行一个）。
  - `ChannelInfo.MultiKeyStatusList`：记录永久失效 token 状态。
  - `OtherInfo`：记录 token 冷却到期时间等临时状态。

## Components

### 1. 配置与持久化层
- 在 `dto.ChannelOtherSettings` 中新增 `XAIAuthMode string`。
- 在 `controller/channel.go` 的 `validateChannel` 中为 `xAI` 增加模式校验：
  - `api_key`：沿用当前校验逻辑。
  - `account_token`：允许“每行一个 token”，做去空白、去空行、去重与基础格式校验。
- 保持 `multi_to_single` / `batch` / `single` 现有录入模式不变，避免新增专门的账号池模型。

### 2. xAI 鉴权模式分流层
- 在 `relay/channel/xai/` 中增加模式分流逻辑，将当前 `Adaptor` 拆分为：
  - `api_key` 模式：继续走现有 `api.x.ai` 官方 API 逻辑。
  - `account_token` 模式：走 Grok 网页账号 token 分支。
- 建议新增辅助文件：
  - `auth_mode.go`：鉴权模式解析。
  - `account_pool.go`：账号池选择与状态迁移。
  - `account_request.go`：账号模式请求头、Cookie、URL 与错误分类。

### 3. token 池状态机
账号模式下，每一行 token 视为一个池内账号，状态分为：
- **正常**：可被选中。
- **永久失效**：收到 `401/403` 后，写入 `ChannelInfo.MultiKeyStatusList`，不再参与调度。
- **临时冷却**：收到 `429` 后，在 `OtherInfo` 中记录 `cooldown_until`，冷却期间跳过，过期后自动恢复。

#### 状态存储约定
- 永久失效：继续使用现有多 key 状态。
- 临时冷却：在 `OtherInfo` 中增加类似结构：
  - `xai_account_cooldown_until: { "0": 1711111111, "3": 1711111188 }`

### 4. token 选择规则
在 `account_token` 模式下新增专用 token 选择逻辑：
1. 读取当前渠道的 token 列表。
2. 按现有 `random / polling` 模式遍历候选 token。
3. 跳过：
   - 已 `auto_disabled` 的 token。
   - `cooldown_until > now` 的 token。
4. 对于 `cooldown_until <= now` 的 token，执行懒恢复并重新纳入候选。
5. 选出第一个可用 token 发起请求。

### 5. 失败与恢复策略
账号模式下，同一渠道内优先切换下一个 token，而不是立即把错误抛给外层渠道切换逻辑：
- `401/403`：将当前 token 标记为永久失效，然后继续尝试下一个 token。
- `429`：将当前 token 标记为临时冷却，然后继续尝试下一个 token。
- `5xx` / 网络错误：不禁用 token，交由现有外层重试逻辑处理。

#### 池耗尽行为
- 如果所有 token 都只是冷却中：返回“临时饱和”错误，不自动禁用整个渠道。
- 如果所有 token 都永久失效：允许沿用现有渠道自动禁用逻辑。

#### 自动恢复
第一版采用**懒恢复**：每次选 token 时检查冷却时间，过期即删除冷却记录，无需新增定时任务。

### 6. 能力与路由范围
- 保持 `xAI` 渠道为统一能力入口。
- 两种鉴权模式都应覆盖同一套 xAI 能力：
  - `/v1/chat/completions`
  - `/v1/responses`
  - `/v1/images/generations`
  - `/v1/images/edits`
  - `/v1/videos/...`
- 仓库当前已有 `xAI` 的文本 / responses / 图片适配；本次设计要求把 `xAI` 视频链也补齐，避免“模式支持但能力缺失”的不一致。

### 7. xAI 视频链补齐
- 在 `relay/channel/task/` 下新增 `xai` 任务适配器。
- 在 `relay/relay_adaptor.go` 中为 `ChannelTypeXai` 注册视频 task adaptor。
- `api_key` 与 `account_token` 两种模式共用视频任务入口，只在具体请求构造时分流上游鉴权与请求头。

### 8. 前端交互
在 `web/src/components/table/channels/modals/EditChannelModal.jsx` 中为 `xAI` 渠道新增显式鉴权模式开关：
- `API Key 模式`
- `账号 Token 模式`

切换到 `账号 Token 模式` 后：
- 渠道密钥提示改为“每行一个 token”。
- 允许继续使用现有多 key / 批量录入逻辑。
- 不影响现有 `xAI API Key` 渠道的创建与编辑行为。

## Data Flow
1. 管理员在 `xAI` 渠道选择 `账号 Token 模式`，并在密钥框内粘贴多行 token。
2. 后端保存到 `Channel.Key`，多 key 元数据仍保存在 `ChannelInfo`。
3. 请求进入 `xAI` 适配器后，根据 `xai_auth_mode` 选择 API key 分支或账号 token 分支。
4. 账号 token 分支从池中挑选一个可用 token，组装账号模式请求并发往对应上游。
5. 上游返回后：
   - 成功：直接透传并统计用量。
   - `401/403`：禁用当前 token，尝试下一个 token。
   - `429`：冷却当前 token，尝试下一个 token。
   - 其他失败：按现有外层重试逻辑继续处理。

## Error Handling
- 账号 token 模式下，必须区分“账号级失败”和“渠道级失败”，避免 `429` 导致整个渠道被误禁用。
- 当池内仍有冷却 token 时，应返回“临时饱和”而不是“无可用 token 永久失效”。
- 当前端选择 `账号 Token 模式` 但未提供有效 token 时，创建/更新渠道直接失败。
- `xAI API Key` 模式行为保持不变。

## Testing
至少覆盖以下测试：
- `xai_auth_mode` 解析与默认值行为。
- 账号 token 输入切分：去空白、去空行、去重。
- token 池选择逻辑：
  - 跳过永久失效 token。
  - 跳过冷却 token。
  - 冷却过期后自动恢复。
  - `random / polling` 两种模式都能正确工作。
- 状态流转：
  - `401/403` -> 永久禁用。
  - `429` -> 临时冷却。
  - `5xx` / 网络错误 -> 不禁用。
- 池耗尽：
  - 全部冷却中时不禁用整个渠道。
  - 全部永久失效时允许渠道进入自动禁用。
- `xAI` 适配器：
  - `API Key` 模式不回归。
  - `账号 Token` 模式会走不同请求构造。
  - 文本 / responses / 图片 / 视频路由分发正确。
- 前端：
  - `xAI` 渠道出现鉴权模式开关。
  - 切换到账号模式后，文案与输入提示正确。
  - 旧的 `API Key` 模式编辑行为零回归。

## Non-Goals
- 不新增 `Grok` / `Grok2API` 渠道类型。
- 不新建独立账号池数据库表。
- 不把账号 token 模式泛化到 OpenAI、Codex 或其他渠道。
- 第一版不引入专门的后台恢复定时任务；冷却恢复采用懒恢复。
