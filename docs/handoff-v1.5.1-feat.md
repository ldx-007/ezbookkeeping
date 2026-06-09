# Handoff Document — v1.5.1-feat 分支

> 本文档记录了在 `v1.5.1-feat` 分支上所做的功能开发，供后续在此基础上继续开发时参考。

---

## 项目概述

[ezbookkeeping](https://github.com/mayswind/ezbookkeeping) 是一个使用 **Go**（后端）+ **Vue 3**（前端）开发的记账软件。

- **后端框架**: Go + xorm（ORM），按时间序列（transaction_time）分页查询交易
- **前端**: Vue 3 + Vite + Vuetify（桌面端）+ Framework7（移动端）
- **构建**: Vite 构建前端，Go 构建后端，Docker 多架构镜像（linux/amd64 + linux/arm64）
- **CI/CD**: GitHub Actions 构建并推送到 GHCR，另有 Gitea Actions 配置
- **Docker 镜像**: `ghcr.io/mayswind/ezbookkeeping:v1.5.1-feat`（每次 push 自动更新）

---

## 功能说明：交易列表按金额排序

在原有按交易时间排序的基础上，增加了按**金额升序**和**金额降序**排序的功能。桌面端和移动端均支持。

### 实现原理

#### 后端（Go）

| 文件 | 改动 |
|------|------|
| `pkg/models/transaction.go:231,254` | 请求结构体 `TransactionListByMaxTimeRequest` 和 `TransactionListInMonthByPageRequest` 新增 `AmountSortOrder` 字段，`form:"amount_sort_order"` |
| `pkg/api/transactions.go:180,204,280,295` | 将 `amountSortOrder` 逐层传递到 service 层；"按月查询"和"按最大时间查询"两个接口都支持 |
| `pkg/api/transactions.go:2873` | `getTransactionResponseListResult` 新增 `disableTimeSort ...bool` 变参：当按金额排序时，跳过后端默认的时间排序（`sort.Sort(result)`），保持数据库返回的顺序 |
| `pkg/services/transactions.go:2771` | 新增 `getAmountSortOrder()` 方法：根据传入的排序参数返回 SQL ORDER BY 子句。`"asc"` → `"amount asc, transaction_time desc"`，`"desc"` → `"amount desc, transaction_time desc"`，空值或不合法 → `"transaction_time desc"`（默认行为） |
| `pkg/services/transactions.go:391,412,439,474` | 三个查询方法都接收 `amountSortOrder ...string` 变参并传入 SQL 查询 |
| `pkg/services/amount_sort_test.go` | 新增单元测试，覆盖 `asc`、`desc`、空值、非法值、无参 5 种情况 |

**关键设计**：金额排序时，数据库层面用 `amount {asc|desc}, transaction_time desc` 作为二级排序，保证相同金额的记录按时间倒序排列，分页结果稳定。

---

#### 前端 — 公共层（TypeScript）

| 文件 | 改动 |
|------|------|
| `src/core/numeral.ts:539` | 新增 `AmountSortOrderType` 类，定义 `AmountAscending`（`'asc'`）和 `AmountDescending`（`'desc'`），提供 `values()` / `valueOf()` / `isValidSortOrder()` 静态方法 |
| `src/models/transaction.ts:616,630` | 请求接口 `TransactionListByMaxTimeRequest` 和 `TransactionListInMonthByPageRequest` 新增 `amountSortOrder: string` 字段 |
| `src/lib/services.ts:529,532` | API 请求 URL 增加 `amount_sort_order=${req.amountSortOrder}` 参数 |
| `src/stores/transaction.ts:71,86` | `TransactionListPartialFilter` 和 `TransactionListFilter` 新增 `amountSortOrder` |
| `src/stores/transaction.ts:128` | `transactionsFilter` 默认值增加 `amountSortOrder: ''` |
| `src/stores/transaction.ts:721-726,785-792` | `setTransactionListFilter` 和 `updateTransactionListFilter` 中处理 `amountSortOrder` 的赋值与变更检测 |
| `src/stores/transaction.ts:827-831` | `getTransactionListQueryString` 中将 `amountSortOrder` 序列化到 URL |
| `src/stores/transaction.ts:852-890` | `loadTransactions()` 中**核心分页逻辑调整**：按金额排序时，改用**基于页码的分页**（page-based），而非默认的基于时间游标的分页（cursor-based）。新增 `transactionsLoadPageNumber` 状态跟踪加载的页码 |
| `src/stores/transaction.ts:928` | 每次成功加载后递增页码 |
| `src/router/desktop.ts:118` | 路由中将 `amountSortOrder` 查询参数传递给桌面列表组件 |

---

#### 前端 — 桌面端（Vuetify）

| 文件 | 改动 |
|------|------|
| `src/views/desktop/transactions/ListPage.vue` | 过滤菜单中增加 **Sort** 分组，含"金额升序"和"金额降序"两个选项；选中状态高亮 + check 图标；点击已选中的排序项可取消排序；高亮金额按钮指示排序状态 |
| `src/views/desktop/transactions/ListPage.vue:1239` | 初始化时读取 `initAmountSortOrder` |
| `src/views/desktop/transactions/ListPage.vue:1633` | `changeAmountSortOrder()` 方法：点击切换/取消排序，调用 `updateTransactionListFilter`，然后更新 URL |
| `src/views/desktop/transactions/ListPage.vue:756` | 导入 `AmountSortOrderType` |

---

#### 前端 — 移动端（Framework7）

| 文件 | 改动 |
|------|------|
| `src/views/mobile/transactions/ListPage.vue` | 更多菜单（popover）中增加 **Sort** 分组，含"金额升序"和"金额降序"选项 |
| `src/views/mobile/transactions/ListPage.vue:978-992` | 新增 `flatTransactionItems` 计算属性：将按月份分组的交易数据展平为一维数组，供排序模式下的列表渲染 |
| `src/views/mobile/transactions/ListPage.vue:981-990` | 新增 `getDisplayYear()` / `getDisplayMonthDay()` 辅助方法 |
| `src/views/mobile/transactions/ListPage.vue:175-274` | 排序模式下使用**单独的列表模板**（`transaction-info-list`），按平铺方式展示交易，包含日期、分类图标、金额、备注、标签、账户等信息 |
| `src/views/mobile/transactions/ListPage.vue:280-308` | 原按月分组的列表模板增加 `v-if="!query.amountSortOrder"`，排序模式下隐藏 |
| `src/views/mobile/transactions/ListPage.vue:1564` | `changeAmountSortOrder()` 方法：变更后调用 `reload()` |
| `src/views/mobile/transactions/ListPage.vue:1756-1765` | 排序模式下的日期样式调整 |

**关键设计**：移动端按月分组展示（折叠卡片）与按金额排序不兼容，所以排序模式下使用平铺列表替代分组列表，每条记录独立展示完整日期。

---

#### 国际化

| 文件 | 新增键值 |
|------|---------|
| `src/locales/en.json` | `"Amount (Ascending)"` / `"Amount (Descending)"` |
| `src/locales/zh_Hans.json` | `"金额升序"` / `"金额降序"` |

---

### 分页逻辑变更说明

这是改动中最需要注意的部分。

**默认行为（不排序）**：使用 `max_time` 游标分页。每次加载时传当前最后一条交易的时间作为 `max_time`，后端返回比这个时间更早的记录。这是原项目的标准做法。

**按金额排序时**：由于数据库的 ORDER BY 改成了 `amount`，时间不再单调，不能再用 `max_time` 游标来做分页。所以改为：

1. `transactionsLoadPageNumber` 跟踪当前加载到了第几页（从 1 开始）
2. `max_time` 固定为过滤条件中的值（不随已加载数据变化）
3. 每次加载时传递 `page` 参数（`actualPage`）到后端
4. 加载成功后页码递增

相关代码位置：`src/stores/transaction.ts:852-930`

---

### CI/CD 变更

| 文件 | 说明 |
|------|------|
| `.github/workflows/docker-publish.yml` | **新增文件**。在 `main-with-amount-order` / 现在为 `v1.5.1-feat` 分支 push 时自动构建 Docker 镜像并推送到 GHCR。`latest` 标签也绑定到此分支 |
| `.gitea/workflows/docker-release.yml` | Docker 仓库地址改为 `ldx-007/ezbookkeeping`（个人 fork 配置） |
| `.gitea/workflows/docker-snapshot.yml` | 同上 |

**更新上游后注意**：如果上游版本变化，需要重新 cherry-pick 这些 CI 文件的更改，因为它们引用的不是原仓库路径。

---

### 已修复的 Bug

1. **按金额排序时多个月份不显示**（`1d54f397`）：排序模式下改用平铺列表，不按月份分组
2. **移动端按金额排序时列表不 load more**（`a66fa700`）：修复了分页逻辑，使用正确的页码增量

---

### 后续开发注意事项

1. **合入上游更新时**：最有可能冲突的部分是 `pkg/services/transactions.go` 中的查询方法签名和 `src/stores/transaction.ts` 中的分页逻辑。其次是桌面端和移动端的 `ListPage.vue` 中过滤菜单部分
2. **如果有其他新增的 API 查询接口**（如导出功能），也需要传递 `AmountSortOrder` 参数
3. **如果想加更多排序字段**（如按类别排序等），可以在 `getAmountSortOrder()` 方法中添加新的 case
4. **Docker 构建**：当前配置只构建 linux/amd64 和 linux/arm64，如需其他平台在 `docker-publish.yml` 的 `platforms` 中添加

---

### 项目启动

```bash
# 前端开发服务器
npm run serve

# 构建
npm run build

# 生产环境构建的产出在 ../dist/
```
