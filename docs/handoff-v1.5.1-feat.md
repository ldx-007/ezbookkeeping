# Handoff Document - `v1.5.1-feat`

> 本文档用于说明 `v1.5.1-feat` 分支目前已经完成的功能改动、关键实现思路、涉及文件以及后续接手时需要注意的点。

---

## 1. 分支目标概览

这个分支目前主要包含两类改动：

1. 交易列表新增“按金额排序”能力
2. 交易列表总收入 / 总支出展示增强

第二部分是在第一部分基础上的后续补充，重点解决了：

- 自定义日期范围下没有显示总收入 / 总支出的问题
- 多币种金额被错误合并到一起的问题
- 桌面端自定义日期总额会受分页条数影响的问题
- 移动端多币种总额过长时显示不完整的问题

---

## 2. 项目技术栈

- 后端：Go + xorm
- 前端桌面端：Vue 3 + Vite + Vuetify
- 前端移动端：Vue 3 + Framework7
- 构建：
  - 前端：`npm run serve` / `npm run build`
  - 后端：`go build ./...`

---

## 3. 已完成功能

### 3.1 交易列表按金额排序

在原有按时间排序的基础上，新增了：

- 金额升序 `asc`
- 金额降序 `desc`

桌面端和移动端都支持。

#### 后端实现

- `pkg/models/transaction.go`
  - `TransactionListByMaxTimeRequest`
  - `TransactionListInMonthByPageRequest`
  - 新增 `AmountSortOrder`

- `pkg/api/transactions.go`
  - 将 `amountSortOrder` 透传到 service 层
  - 月度查询和普通列表查询都支持金额排序

- `pkg/services/transactions.go`
  - 增加金额排序 SQL 逻辑
  - 排序规则：
    - `asc` -> `amount asc, transaction_time desc`
    - `desc` -> `amount desc, transaction_time desc`
    - 空值或非法值 -> `transaction_time desc`

- `pkg/services/amount_sort_test.go`
  - 补充金额排序相关单测

#### 前端公共层实现

- `src/core/numeral.ts`
  - 新增 `AmountSortOrderType`

- `src/models/transaction.ts`
  - `TransactionListByMaxTimeRequest`
  - `TransactionListInMonthByPageRequest`
  - 新增 `amountSortOrder`

- `src/lib/services.ts`
  - 列表请求追加 `amount_sort_order`

- `src/stores/transaction.ts`
  - 列表筛选状态新增 `amountSortOrder`
  - URL 序列化与反序列化支持该字段
  - 金额排序时改为页码分页，而不是时间游标分页

#### 桌面端

- `src/views/desktop/transactions/ListPage.vue`
  - 筛选菜单新增排序项
  - 支持切换 / 取消金额排序

#### 移动端

- `src/views/mobile/transactions/ListPage.vue`
  - 更多筛选菜单新增排序项
  - 排序模式下不再按月份折叠展示，而是改为平铺列表

---

### 3.2 月份 / 自定义日期总收入总支出增强

这是本轮新增的重点。

#### 目标

统一桌面端和移动端的行为：

- 月份视图显示总收入 / 总支出
- 自定义日期范围也显示总收入 / 总支出
- 总额按币种分别展示，而不是把不同货币直接累加
- 示例：
  - `总收入 ¥ 2000 / $ 2000`
  - `总支出 ¥ 1500 / $ 500`

#### 已解决的问题

1. 自定义日期原本不显示总收入 / 总支出
2. 不同货币原本会被直接加总，导致结果不准确
3. 桌面端自定义日期总额原本只统计“当前页数据”，分页条数变化时总额也会变化
4. 移动端多币种金额过长时无法完整显示

---

## 4. 本轮关键实现思路

### 4.1 统一月度汇总的数据结构

文件：`src/stores/transaction.ts`

新增：

- `TransactionCurrencyAmount`
- `TransactionTotalAmountByCurrency`
- `TransactionMonthList.totalAmountByCurrency`

作用：

- 月份数据除了保留原有“默认货币汇总值”，还额外保留“按币种拆分后的收入 / 支出总额”

这样桌面端和移动端都可以直接复用月度数据，而不用重复写月度汇总逻辑。

#### 汇总规则

- 普通收入 / 支出：按原账户币种累计
- 转账：
  - 仅在存在账户筛选时参与“流入 / 流出”统计
  - 若源账户命中筛选、目标账户未命中，则记为支出
  - 若目标账户命中筛选、源账户未命中，则记为收入
  - 若两边都命中或都不命中，则不计入流入流出

这部分规则与原有总额统计逻辑保持一致。

---

### 4.2 桌面端自定义日期总额改为“全量汇总”

文件：

- `src/views/desktop/transactions/ListPage.vue`
- `src/lib/services.ts`
- `src/models/transaction.ts`

#### 实现原则

用户要求“参考月份的实现方式”，所以最终没有新增专门的 summary API，而是复用现有能力：

1. 自定义日期时，先请求该筛选条件下的完整交易列表
2. 前端把完整结果转成 `Transaction[]`
3. 前端按和月份汇总一致的规则计算收入 / 支出
4. 最终按币种格式化后显示到右上角

#### 关键点

- `src/lib/services.ts`
  - `getAllTransactions()` 现在支持更多筛选参数：
    - `type`
    - `categoryIds`
    - `accountIds`
    - `tagFilter`
    - `amountFilter`
    - `keyword`
    - `mustHavePictures`
    - `startTime`
    - `endTime`

- `src/models/transaction.ts`
  - `TransactionAllListRequest` 扩展为可携带上述过滤条件

- `src/views/desktop/transactions/ListPage.vue`
  - 月份模式：
    - 直接使用 `currentMonthTransactionData.totalAmountByCurrency`
  - 自定义日期 / 非月份列表模式：
    - 调用 `loadFullRangeTotalAmount()`
    - 基于完整结果计算 `fullRangeTotalAmount`

#### 结果

桌面端现在满足：

- 自定义日期显示总收入 / 总支出
- 总额不受“每页显示多少条”影响
- 多币种分别显示
- 若没有任何币种数据，则显示默认货币 `0`
- 若只有一种币种，则不会再出现多余的 `+`

---

### 4.3 移动端总额展示增强

文件：`src/views/mobile/transactions/ListPage.vue`

移动端新增了列表页顶部的总收入 / 总支出卡片，并做了以下处理：

1. 支持按币种分别展示
2. 多币种时改为逐行展示，而不是单行拼接
3. 左右两列标题顶部对齐，避免一侧金额换行后另一侧上移
4. 放开 `f7-list-item` 默认单行截断限制，允许长金额自动换行

#### 当前展示方式

- `income` / `expense` 在前端先被格式化成字符串数组
- 模板中通过 `v-for` 逐行渲染

这样即使出现：

- `¥ 123456789.00`
- `$ 987654321.00`
- `€ 456789.00`

也能完整展示，不会被父容器裁掉。

---

## 5. 涉及文件汇总

### 5.1 本轮核心文件

- `src/stores/transaction.ts`
  - 月度总额新增按币种汇总结构和计算逻辑
  - 新增 `calculateCurrencyTotalAmounts()` 导出函数，供桌面端和后续功能共用按币种汇总逻辑

- `src/views/desktop/transactions/ListPage.vue`
  - 桌面端月份 / 自定义日期总额展示
  - 自定义日期全量汇总逻辑
  - 改用共享的 `calculateCurrencyTotalAmounts()`，移除本地重复实现
  - 修复：`loadFullRangeTotalAmount` 失败不影响主列表、翻页不重复拉取总额、竞态保护

- `src/views/mobile/transactions/ListPage.vue`
  - 移动端总额展示卡片
  - 多币种换行展示和样式修正

- `src/lib/services.ts`
  - `getAllTransactions()` 支持完整筛选参数

- `src/models/transaction.ts`
  - `TransactionAllListRequest` 扩展

### 5.2 与按金额排序相关的核心文件

- `pkg/models/transaction.go`
- `pkg/api/transactions.go`
- `pkg/services/transactions.go`
- `pkg/services/amount_sort_test.go`
- `src/core/numeral.ts`
- `src/stores/transaction.ts`
- `src/views/desktop/transactions/ListPage.vue`
- `src/views/mobile/transactions/ListPage.vue`

---

## 6. 关键行为说明

### 6.1 为什么桌面端自定义日期不能直接使用当前页数据？

因为当前页只是分页结果的一部分。

如果总额直接基于当前页计算，会出现：

- 每页 20 条时一个总额
- 每页 100 条时另一个总额

这不符合“该日期范围内全部数据总和”的业务含义。

因此当前实现改为：

- 列表继续分页显示
- 总额单独基于该筛选范围内的完整数据计算

---

### 6.2 为什么没有新增专门的 summary 接口？

中间一度尝试过新增后端 summary API，但最终撤回了。

最终选择复用现有 `list/all` 的原因：

1. 更符合“参考月份实现方式”的要求
2. 逻辑更统一，前端可以复用同一套汇总规则
3. 减少额外后端接口和维护成本

---

### 6.3 多币种为什么不直接折算成默认货币？

当前这个需求关注的是“原币种分别展示”，不是“统一折算后展示”。

因此展示策略是：

- 保留原币种
- 分币种显示
- 默认货币只用于兜底显示 `0`

例如：

- 没有收入时显示 `¥ 0`
- 有人民币和美元收入时显示 `¥ 100 / $ 20`

---

## 7. 校验结果

本轮改动已经通过：

```bash
npx vue-tsc --noEmit
go build ./...
```

---

## 8. 后续接手注意事项

### 8.1 `list/all` 的”全量”语义

桌面端自定义日期总额依赖 `v1/transactions/list/all.json`。

经检查，后端 `GetAllSpecifiedTransactions()` 使用游标分页循环拉取全部数据（每页 1000 条），**没有硬性数量上限**，可以放心使用。

但如果某用户单次筛选范围内的交易量极大（数万笔以上），循环分页会导致多次 SQL 查询，响应时间会明显增加。如果后续出现此场景，可考虑为 `list/all` 加一个专门的聚合接口。

### 8.2 月度汇总和自定义汇总的规则要保持一致

后续如果再改”账户筛选下转账如何计入流入 / 流出”，现在只需要改一处：

1. `src/stores/transaction.ts`
   - `calculateCurrencyTotalAmounts()` 函数（模块级导出）
   - 桌面端自定义日期全量汇总也调用此函数

月度汇总的 `calculateMonthTotalAmount()` 内联了独立的按币种累加逻辑，规则和 `calculateCurrencyTotalAmounts()` 一致但未经共享抽象。如后续规则变更，建议将月度汇总中的按币种部分也迁至 `calculateCurrencyTotalAmounts()`。

### 8.3 移动端总额区域后续可继续优化

当前已经解决：

- 长金额换行
- 顶部对齐
- 父容器截断

如果以后还想进一步提升移动端可读性，可以考虑：

- 超窄屏时改成上下堆叠布局
- 为每种币种加更紧凑的间距
- 收入 / 支出卡片拆成两行独立块

---

## 9. 启动方式

```bash
# 前端开发服务器
npm run serve

# 前端构建
npm run build

# 后端构建
go build ./...
```

如果本地默认端口已被占用，Vite 可能自动切换到其他端口，例如 `8081`。

---

## 10. 当前结论

这个分支目前已经完成：

- 交易列表按金额排序
- 桌面端自定义日期总额展示
- 月份 / 自定义日期多币种总额展示
- 移动端总额展示与换行优化
- 代码审查与修复（详见第 11 节）

如果后续继续开发，最值得优先关注的是：

1. `list/all` 在超大数据量下的响应性能
2. 月度 `calculateMonthTotalAmount()` 中的按币种逻辑是否可迁至 `calculateCurrencyTotalAmounts()` 共享
3. 移动端总额区域是否还需要进一步压缩布局

---

## 11. 代码审查与修复

本轮代码审查针对第二部分（总收入/总支出增强）发现了以下问题并已修复。

### 11.1 严重：`loadFullRangeTotalAmount` 失败阻塞主列表

**问题：** `reload()` 将 `loadFullRangeTotalAmount()` 放在 `Promise.all` 中，如果全量 API 失败，整个交易列表也加载失败。

**修复：** 加上 `.catch(() => {})`，总额失败不影响主列表。额度是辅助信息，不应影响主功能的使用。

**涉及文件：** `src/views/desktop/transactions/ListPage.vue`

### 11.2 中高：不必要的 API 调用

**问题：** 每次 `reload()` 都无条件调用 `getAllTransactions` 拉取全量数据：
- 用户关闭 “显示总额” 设置时也调用
- 翻页/改每页条数时也重新拉取

**修复：** 
- 加 `showTotalAmountInTransactionListPage`、`!queryMonthlyData`、`pageType === List.type` 三重守卫
- 引入 `lastTotalFilterKey` 缓存筛选条件 key，仅筛选条件变化时才重新拉取

**涉及文件：** `src/views/desktop/transactions/ListPage.vue`

### 11.3 中：并发 reload 竞态

**问题：** 快速切换筛选条件时，多个 `loadFullRangeTotalAmount()` 请求并发，先返回的慢请求可能覆盖后返回的正确结果。

**修复：** 引入 `fullRangeTotalRequestId` 递增计数器，只有最新请求的结果才写入 `fullRangeTotalAmount`，旧请求静默丢弃。

**涉及文件：** `src/views/desktop/transactions/ListPage.vue`

### 11.4 建议：`getDisplayCurrencyTotalAmounts` 类型简化

**问题：** 函数签名包含从未使用的 `incomeAmount`/`expenseAmount` 联合类型分支。

**修复：** 简化为仅接受 `TransactionCurrencyAmount[]`，移除 `type` 参数和对应的 `TransactionListDisplayTotalAmountItem` 接口。

**涉及文件：** `src/views/desktop/transactions/ListPage.vue`

### 11.5 建议：桌面端自定义日期总额计算逻辑去重

**问题：** 桌面端 `calculateDisplayTotalAmountByTransactions` 与 store 中 `calculateMonthTotalAmount` 的按币种汇总逻辑重复。

**修复：** 将公共逻辑提取为 `calculateCurrencyTotalAmounts()` 导出函数，放在 `src/stores/transaction.ts` 模块级，桌面端调用此函数替代本地实现。后续规则变更只需改一处。

**涉及文件：** `src/stores/transaction.ts`（新增函数）、`src/views/desktop/transactions/ListPage.vue`（删除重复代码）

### 11.6 建议：死代码清理

**问题：** `loadFullRangeTotalAmount` 中 `mustHavePictures` 的值恒为 `false`（函数入参已保证 `pageType` 只能是 `List.type`）。

**修复：** 直接写 `false`。

**涉及文件：** `src/views/desktop/transactions/ListPage.vue`

### 11.7 修复：移动端总额展示框限制为仅自定义日期范围

**问题：** 移动端交易列表新增的总收入/总支出展示框在所有日期选项（今天、昨天、本月、上月等）下都显示了。原需求是该展示框仅用于自定义日期范围，其他选项已有各自的总额展示方式。

**修复：** 在 `v-if` 条件中增加 `query.dateType === DateRange.Custom.type`，确保仅自定义日期时才显示该卡片。

**涉及文件：** `src/views/mobile/transactions/ListPage.vue`

### 11.8 后端审查

后端改动来自金额排序功能（commit `f151127b`），经检查：

- `amountSortOrder` 在 API → Service → SQL 三层正确透传
- `disableTimeSort` 在金额排序时正确跳过 post-processing 的时间排序
- SQL 排序 `amount asc/desc, transaction_time desc` 语义正确
- `GetAllSpecifiedTransactions()` 使用游标循环分页，无硬性数量上限
- 单测覆盖 asc/desc/空/非法/无参数 5 种情况

**结论：后端无需修改。**

---

## 12. 校验方式

```bash
# 前端类型检查
npx vue-tsc --noEmit

# 后端编译
go build ./...
```
