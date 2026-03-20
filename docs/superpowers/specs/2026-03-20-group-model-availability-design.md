# 分组模型可用性可视化设计

- 日期：2026-03-20
- 状态：已确认设计，待进入 implementation plan
- 适用范围：API Plus 后台与用户端 Web 界面

## 1. 背景

当前代码中已经存在两类相关能力，但没有一个专门的“分组 × 模型可用性”可视化界面：

1. **渠道健康检查**
   - 后端：`GET /api/channel/health`
   - 前端：`web/src/components/table/channels/modals/ChannelHealthModal.jsx`
   - 能看渠道是否可用、响应时间、余额
   - 不能按“分组 × 模型”展示

2. **模型/分组静态可用信息**
   - 配置来源：`abilities`
   - 当前已有“模型的可用分组”等静态展示
   - 不能直观看“某个分组下每个模型最近的真实请求表现”

用户需要一个新的可视化页面，用来查看**当前每个分组下每个模型的可用性**。

## 2. 目标

提供一个独立页面，以**分组**为主视角，展示该分组下每个模型：

1. 是否配置可用
2. 最近 20 条**全局**请求结果
3. 最近一次主动重测结果

同时在首页提供入口卡片，点击进入该页面。

## 3. 已确认的产品决策

### 3.1 页面入口

- 新增一个独立页面：`/model-availability`
- 首页提供入口卡片
- V1 不强制增加侧边栏入口

### 3.2 目标用户

- 普通用户可以访问
- 但只能看到**自己有权限的分组/模型**
- 管理员/root 可看到全部分组/模型

### 3.3 主视角

- 采用“**按分组展示每个模型**”的布局
- 不采用“全量矩阵优先”作为主页面
- 页面结构：
  - 左侧：分组列表
  - 右侧：当前分组下模型列表

### 3.4 最近请求结果的展示口径

- 统计维度：**全局**
- 键：`group + model_name`
- 窗口：最近 20 条请求
- 展示方式：
  - 成功请求：绿色
  - 失败请求：红色
- 如果没有请求数据：
  - 仍显示绿色
  - 但需要标注“默认绿 / 暂无请求”，避免误解为真实探测成功

### 3.5 可用性含义

该页面同时体现两层信息：

1. **配置可用性**
   - 基于 `abilities`
   - 表示该分组是否理论上可以调用该模型

2. **运行表现**
   - 基于最近 20 条全局请求日志
   - 以绿/红点阵反映近期真实请求结果

此外保留：

3. **主动重测**
   - 用户可手动对某个 `group + model` 发起重测
   - 展示最近一次 probe 结果

## 4. 非目标

V1 不做以下能力：

- 不在首页直接展示完整矩阵
- 不把页面放回模型管理页内部
- 不做复杂的跨分组对比矩阵主视图
- 不对 `auto` 单独做虚拟统计页签
- 不做“所有渠道都成功才算可用”的严格健康度判定

## 5. 现有代码可复用点

### 5.1 配置可用性

- `service.GetUserUsableGroups(userGroup)`
- `model.GetGroupEnabledModels(group)`
- `abilities` 表本身就是分组与模型的配置可用性来源

### 5.2 请求结果日志

`logs` 表已经记录：

- `group`
- `model_name`
- `created_at`

并可通过 `type` 区分：

- `LogTypeConsume`：成功请求
- `LogTypeError`：失败请求

### 5.3 主动探测能力

现有已有渠道测试能力：

- `controller/channel-test.go`
- `testChannel(...)`
- `GET /api/channel/health`

V1 不直接复用健康检查 UI，但可以复用其测试逻辑。

## 6. 方案比较

### 方案 A：纯实时查询

页面打开时直接实时聚合：

- 读可见分组
- 读该分组可用模型
- 从日志中实时回溯最近 20 条结果

优点：

- 实现直接
- 不需要额外缓存结构

缺点：

- 数据量增长后性能不稳
- 首次页面打开可能慢

### 方案 B：全量预聚合

后台周期性维护完整快照。

优点：

- 页面快
- 表现稳定

缺点：

- 需要新缓存/新表/新同步机制
- V1 成本较高

### 方案 C：混合方案（推荐）

- 配置可用性：实时取 `abilities`
- 最近 20 条请求结果：优先读取按分组缓存的页面快照
- 支持手动刷新快照
- 支持单模型主动 probe

结论：**V1 采用方案 C**

## 7. 最终页面设计

## 7.1 首页入口

首页新增一个入口卡片：

- 标题：模型可用性
- 描述：查看每个分组下各模型最近 20 次请求的成功/失败情况
- 操作：进入页面

## 7.2 独立页面结构

顶部区域：

- 最近刷新时间
- 手动刷新按钮
- 搜索模型
- 仅看失败模型
- 仅看最近有请求模型

主体区域：

- 左侧分组列表
- 右侧当前分组的模型列表

## 7.3 模型行字段

每个模型一行展示：

1. `model_name`
2. `recent_results`
   - 最近 20 次请求的绿/红点阵
3. `config_available`
   - 是 / 否
4. `probe status`
   - 最近一次主动重测结果：通过 / 失败 / 未检测
5. 操作按钮
   - 立即重测

## 7.4 视觉规则

- 成功点：绿色
- 失败点：红色
- 默认补点：绿色，但需有说明
- 探测成功：绿色 tag
- 探测失败：红色 tag
- 未检测：黄色或灰色 tag

## 8. `auto` 分组处理

V1 **不单独为 `auto` 做虚拟统计页签**。

原因：

- 当前日志记录的 `group` 是实际使用的 `using_group`
- 也就是请求落到哪个真实分组，就记哪个真实分组
- 如果强行单独做 `auto`，会与现有日志口径不一致

因此 V1 页面按**真实分组**展示，例如：

- `vip`
- `default`
- `team-a`

而不是额外聚合出一个逻辑 `auto` 分组页。

## 9. 后端接口设计

## 9.1 读取页面数据

新增：

`GET /api/model-availability`

查询参数建议：

- `group`
- `keyword`
- `only_failed`
- `only_with_logs`

返回结构建议：

```json
{
  "success": true,
  "data": {
    "groups": [
      { "name": "vip", "selected": true },
      { "name": "default", "selected": false }
    ],
    "selected_group": "vip",
    "refreshed_at": 1774002000,
    "items": [
      {
        "model_name": "gpt-4o",
        "config_available": true,
        "recent_results": [
          { "status": "success", "defaulted": false },
          { "status": "fail", "defaulted": false },
          { "status": "success", "defaulted": true }
        ],
        "success_count": 18,
        "fail_count": 2,
        "probe": {
          "status": "success",
          "checked_at": 1774001888,
          "message": "检测成功"
        }
      }
    ]
  }
}
```

## 9.2 刷新页面快照

新增：

`POST /api/model-availability/refresh`

请求体：

```json
{
  "group": "vip"
}
```

作用：

- 立即重新计算这个分组的页面快照
- 不主动访问上游
- 仅重算：
  - 分组可用模型
  - 最近 20 条请求点阵
  - 最近 probe 结果

## 9.3 单模型主动重测

新增：

`POST /api/model-availability/probe`

请求体：

```json
{
  "group": "vip",
  "model_name": "gpt-4o"
}
```

逻辑：

- 找到这个 `group + model` 对应的可用渠道
- 顺序或有限并发执行测试
- 任一渠道成功，则本次 probe 判为成功
- 结果写入短期缓存

返回：

- `status`
- `checked_at`
- `message`
- `response_time_ms`

## 10. 数据计算设计

## 10.1 配置可用性

来源：

- `service.GetUserUsableGroups(...)`
- `model.GetGroupEnabledModels(group)`

规则：

- 普通用户：只保留自己可见分组
- 管理员/root：可返回全部分组

## 10.2 最近 20 条请求的聚合

来源表：`logs`

使用：

- `LogTypeConsume` => success
- `LogTypeError` => fail

V1 聚合方式：

1. 先确定当前分组
2. 查询该分组最近一批日志（例如最近 1000~2000 条）
3. 在内存中按 `model_name` 聚合
4. 为每个模型取最近 20 条

选择该方案的原因：

- 避免依赖窗口函数
- 兼容 SQLite / MySQL 5.7 / PostgreSQL
- 改造成本低

## 10.3 无数据补绿规则

若模型无请求：

- 直接补 20 个默认绿色点
- 每个点带 `defaulted = true`

若模型仅有 N 条请求（N < 20）：

- 真实 N 条按成功/失败展示
- 不足的部分补默认绿点

## 10.4 主动 probe

probe 不参与“最近 20 条日志”点阵统计。

原因：

- 点阵语义应保持为“真实用户请求表现”
- probe 作为独立状态展示更清晰

## 11. 缓存设计

## 11.1 页面快照缓存

建议 key：

- `model_availability_snapshot:<group>`

内容：

- 当前分组下模型列表
- 每个模型最近 20 条点阵
- 最近刷新时间

TTL：

- 60 秒

行为：

- 页面优先读缓存
- 手动刷新强制覆盖缓存

## 11.2 probe 缓存

建议 key：

- `model_availability_probe:<group>:<model>`

内容：

- status
- checked_at
- message
- response_time_ms

TTL：

- 5 分钟

## 12. 权限设计

### 普通用户

- 只返回其可用分组
- 只返回这些分组下的配置可用模型

### 管理员 / root

- 可查看全部分组
- 可执行全部分组下的 probe / refresh

## 13. 错误处理

### 日志聚合失败

- 有旧缓存：返回旧缓存并提示“数据可能已过期”
- 无缓存：返回接口错误

### probe 失败

- 行内显示失败状态
- 保留失败原因摘要

### 空状态

- 用户没有可见分组：展示空状态
- 分组没有可用模型：展示“该分组暂无可用模型”

## 14. 前端实现边界

新增内容：

- 新页面：`/model-availability`
- 首页入口卡片
- 页面级 hook
- 模型列表组件
- 点阵展示组件

复用内容：

- 现有 API helper
- 现有权限上下文
- 现有 Semi UI 组件体系

V1 不做：

- 全局矩阵视图
- 高级统计图表
- 渠道级 drill-down 展开
- `auto` 虚拟分组汇总

## 15. 测试与验收

### 后端

1. 分组权限过滤正确
2. 最近 20 条聚合正确
3. consume/error 映射正确
4. 无数据补绿正确
5. probe 任一渠道成功即通过
6. 三数据库兼容

### 前端

1. 首页入口可跳转
2. 分组切换正常
3. 绿/红点渲染正确
4. 搜索与筛选正常
5. 手动刷新与 probe 正常
6. 空状态与失败状态正常

## 16. 推荐实施顺序

1. 后端读取接口
2. 后端 refresh / probe 接口
3. 前端独立页面
4. 首页入口
5. 缓存与交互优化

## 17. 进入实现前的下一步

本设计确认后，下一步应编写 implementation plan，明确：

- 后端路由、controller、service、model 的拆分
- 前端页面、hook、组件的拆分
- 缓存与测试方案
