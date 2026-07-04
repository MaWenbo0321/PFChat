# PFChat 精简重构计划

## 完成状态

全部计划任务已完成并验证。当前实现已移除发送前检查、实时检测、右侧 AI 顾问面板和相关冗余代码；保留两种会话模式与语用错误记录展示。

## 目标

- [x] 删除 `pre_send_check` 发送前检查相关代码。
- [x] 删除 Grammarly 风格实时检测相关代码。
- [x] 删除右侧 AI 顾问面板。
- [x] 保留两种会话模式：
  - `complete`：完整对话，用户手动结束或 10 分钟超时后生成汇总。
  - `rounds_5`：五轮对话，5 轮后自动结束并生成汇总。
- [x] 保留语用错误记录展示，沿用原有记录格式。
- [x] 删除冗余代码和未使用文案。
- [x] 完成后端检查与前端构建验证。

## Phase 1: 后端修改

- [x] 删除 `backend/pre_send_check.go`。
- [x] 从 `backend/main.go` 删除 `/messages/check`、`/messages/realtime-check`、`/ai/chat` 路由。
- [x] 在 `backend/model.go` 中使用 `FeedbackComplete = "complete"` 和 `FeedbackRounds5 = "rounds_5"`。
- [x] 删除未使用的 `AIChatHistory` 模型。
- [x] 从 `backend/grammar_handler.go` 删除实时检测和 AI chat 处理逻辑，仅保留 `getMessageErrorsByIds()`。
- [x] 在 `backend/session_handler.go` 中只接受 `complete` 和 `rounds_5`。
- [x] 在 `backend/llm_chat_handler.go` 返回 `session_ended` 与 `session_summary`。
- [x] 移除阶段性反馈逻辑，`rounds_5` 达到 5 轮后自动结束会话并生成汇总。

## Phase 2: 前端修改

- [x] 从 `frontend/src/api/index.js` 删除 `checkMessageBeforeSend`、`realtimeCheck`、`aiChat`。
- [x] 将 `frontend/src/views/SessionSetup.vue` 的反馈模式卡片改为会话模式卡片。
- [x] 将默认 `feedback_mode` 设置为 `complete`。
- [x] 从 `frontend/src/views/Chat.vue` 删除 AI 顾问面板、实时检测、输入下划线渲染、错误卡片和发送前确认弹窗。
- [x] 简化 `sendMessage()`，直接调用 LLM 会话接口。
- [x] 增加完整对话 10 分钟倒计时自动结束逻辑。
- [x] 处理 `result.session_ended`，自动展示汇总并禁止继续发送。
- [x] 更新聊天头部和侧栏中的 `complete` / `rounds_5` 模式展示。
- [x] 保留会话汇总对话框、错误标识和会话信息侧栏。
- [x] 更新 `frontend/src/i18n/index.js`，删除旧实时检测、发送前确认和阶段性反馈文案。

## Phase 3: 冗余代码清理

- [x] 检查 `backend/llm.go`，保留当前语用检测所需函数。
- [x] 检查 `backend/handler.go`，确认旧用户间发送前检查路径不再参与当前 LLM 会话流程。
- [x] 确认不存在 `frontend/src/stores/chat.js` 残留。
- [x] 检查前后端关键字，确认旧接口和旧 UI 逻辑无残留引用。

## Phase 4: 验证

- [x] `go vet ./...` 通过。
- [x] `go test ./...` 通过；当前后端无测试文件。
- [x] `npm run build` 通过。

## 验证备注

- `go vet` 和 `go test` 使用工作区内临时 `GOCACHE` 执行，以避开默认用户目录缓存权限问题。
- 前端构建存在 Vite 大 chunk 警告和 npm PowerShell 用户目录权限提示，但构建成功。
- `frontend/package-lock.json` 在本次收尾前已处于修改状态，未作为计划任务处理。