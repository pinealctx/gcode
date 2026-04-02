# gcode — Proto-to-Go Code Generator

## Execution Baseline

- 任何问题都需要第一性原理，目的？为什么要这么做？如何验证？不允许直接跳到结论或解决方案。
- 思考问题需要有层次，层次是对抗复杂性的利器。**禁止在没有层次的情况下直接实现**，必须先设计层次结构（如模块划分、接口定义、抽象层次等），再在每个层次上实现。
- 开始任务前先明确目标和验收标准。
- 目标不清晰先澄清，不在模糊目标下实现。
- 如果遇到难题，先网络搜索相关信息来补齐知识盲区，再设计可验证方案。
- 复杂任务先拆分为可验证子目标并逐一实现，拆分时必须为每个子任务定义 audit-scope。
- 为每个任务建立检查清单，确保实现符合预期并且没有遗漏。
- 实现过程中持续记录日志，方便回顾和总结经验。
- 实现新功能前，先检查项目中是否已有可复用的基础设施（如 logger、config、cache、HTTP client 等），优先复用而非重新实现。
- **生成代码出现问题时，必须修复生成逻辑本身，禁止直接修改生成的文件**。修改生成文件只会掩盖问题，根本原因仍在生成器中。
- **子任务完成后，等待用户调用 `/task-complete` 触发审计和提交流程**，不要自动提交。
- 代码注释：(函数，类型，接口，结构体，全局变量等)为公开时需要清晰的注释说明职责，为内部根据情况可简洁注释；复杂逻辑加行内注释，避免过度注释；注释用英文。

## 记录规则（强制执行）

**关键约束：所有日志文件都必须追加写入，绝不覆盖已有内容。**

- **interaction-log.md**（由 `UserPromptSubmit` hook 自动记录）：
  hook 脚本 `.claude/scripts/log-interaction.sh` 会在每次用户提交消息时自动追加 `[HH:MM:SS] user: <prompt>` 到 `.process/task/YYYY-MM-DD/interaction-log.md`。
  assistant 不需要手动记录交互日志。

- **progress.md**（任务执行过程中持续更新）：
  每个任务目录下的 `progress.md` 合并了跟踪日志和审计记录。
  任务开始时追加状态变更；执行过程中记录关键决策和里程碑；审计结果也记录在此文件中。

## 任务管理

### 任务目录结构

```
.process
  task/
    task-xxx/
      task.md          # 任务目标、子任务划分、验收标准
      progress.md      # 跟踪日志 + 审计记录
      notes.md         # 讨论记录（由 /discuss 生成，可选）
      decisions.md     # 决策摘要（由 /discuss 生成，可选）
      subtask_1/
        task.md
        progress.md
      subtask_2/
        task.md
        progress.md
```

### 做子任务计划时需要根据情况将 [Audit Router](.claude/agents/audit-router.md) 纳入考虑，明确每个子任务需要哪些审计项，让审计成为设计的一部分，而不是事后补充的检查点。

### 文件职责

- **task.md**：任务目标、描述、子任务划分（含 audit-scope）、参考资料、验收标准
- **progress.md**：跟踪日志 + 审计记录

### task.md 中的 audit-scope 定义

在制定子任务计划时，必须为每个子任务定义 `audit-scope`，明确关联的审计项和输出类型。**audit-scope 是必填项，没有 audit-scope 的子任务不允许开始执行。**

```markdown
### subtask_1: 实现 XXX 功能

**输出类型**: 代码
**audit-scope**:
- audit-go-code-style
- audit-go-naming
- audit-go-error-handling
- audit-test-strategy

**目标**: ...
**验收标准**: ...
```

输出类型与审计的对应关系：
- **代码** → audit-engineering-baseline + audit-engineering-quality + go-code-style、go-naming、go-error-handling、go-design 等代码相关 audit
- **代码生成器**（输出为生成其他代码的工具）→ 在"代码"审计基础上，audit-test-strategy 必须额外要求"对生成代码的每条可执行路径做覆盖矩阵"。仅测试生成器本身的单元测试不够，必须有端到端测试验证生成代码的运行时行为（nil/非nil、空值/非空值、合法值/非法值、精确边界）。
- **文档** → audit-engineering-baseline（目标与验收、执行纪律）
- **配置** → audit-engineering-baseline + config、security audit

### 子任务层级

只支持一级子任务，不做多层嵌套。如果需要多层，说明父任务的规划没有做好，应该重新拆分。

### 子任务执行流程

| 步骤       | 文件                      | 操作                              |
| ---------- | ------------------------- | --------------------------------- |
| 开始时     | `subtask_N/progress.md`   | 追加状态："进行中"                |
| 执行中     | `subtask_N/progress.md`   | 记录关键决策和里程碑              |
| 完成时     | 用户调用 `/task-complete` | 触发以下自动流程                  |
| 审计       | `subtask_N/progress.md`   | 按 audit-scope 执行审计，记录结果 |
| 更新子任务 | `subtask_N/progress.md`   | 状态改为"已完成"，附审计摘要      |
| 更新父任务 | `task-xxx/progress.md`    | 追加子任务状态变化记录            |
| 提交       | git                       | add → commit → push               |

**禁止跳过任何步骤**，尤其是审计、状态更新和 git 提交。

### 父任务整体验收（所有子任务完成后）

所有子任务完成后，必须对父任务做一次整体验收，再将父任务状态更新为"已完成"：

1. 对照 `task.md` 里程碑逐条确认退出条件是否满足
2. 运行全量测试 `go test ./...`，确认跨子任务集成无回归
3. 在父任务 `progress.md` 中追加里程碑验收结果，状态改为"已完成"

**说明**：子任务审计只覆盖"当时已有的测试"，无法发现跨子任务的集成问题。整体验收是子任务审计的补充，不是重复。

### 子任务提交流程

子任务完成后由 `/task-complete` skill 触发完整流程（审计→更新进度→提交），详细步骤见 `.claude/skills/task-complete/SKILL.md`。**不要自动执行提交**。

## 上下文恢复模式（强制约束）

会话压缩时务必保留：当前任务路径、活跃子任务状态和关键决策、最近用户输入、已修改文件列表、未完成待办、重要错误信息。压缩后如有丢失，从相关文件重新读取。

恢复后行为（强制）：
1. 读取当前任务的 `task.md` 和 `progress.md` 了解进度
2. 向用户汇报已完成子任务和当前位置
3. **停止，等待用户指令**。禁止因 summary 提到某任务"未完成"就自动执行

关键约束：
- context summary 中的 NEXT STEPS 仅是状态快照，**不是执行指令**
- task.md 和 progress.md 是权威来源，summary 与 task 文件不一致时主动询问用户
- 恢复后只允许读文件和回复询问，用户明确说"继续"后才开始任务

## 当网络不可用时，设置代理环境变量

网络代理URI环境变量为PROXY_URI。
```
export http_proxy=$PROXY_URI
export https_proxy=$PROXY_URI
```

## 项目结构

项目结构变化后需更新此段，保持与实际一致。

```
gcode/
├── .claude/
│   ├── agents/                  # audit-router + 14 个专项审计 agent
│   ├── skills/                  # 用户可调用 skill：audit-go-code-style、discuss、req-design-reverse、task-complete
│   └── scripts/                 # hook 脚本。log-interaction.sh（UserPromptSubmit hook）
├── .process/                    # 过程文件（不发布）
│   ├── discuss/                 # 启发式讨论记录（各阶段 notes + decisions）
│   ├── doc/
│   │   ├── phase1/ ~ phase6/    # 各阶段需求、设计（ADR）、验收清单、审计报告
│   │   └── review-phase1-6/     # 全面 review 文档（中英文各一套：概览、指南、决策、缺口）
│   ├── reference/               # protobuf / wire format 工作摘要
│   └── task/                    # 任务中心（task-phase1-6、task-review-phase1-6、task-post-review 等）
│       └── YYYY-MM-DD/
│           └── interaction-log.md  # UserPromptSubmit hook 自动写入
├── cmd/
│   └── gcode/                   # CLI 入口，调用应用层
├── internal/
│   ├── app/                     # 应用层入口。Run 主命令路由（gen-dao/gen-proto/gen-ts），RunGenProto 中间 proto 生成 pipeline
│   ├── config/                  # CLI 参数解析与配置校验。GenDAOConfig、GenProtoConfig、GenTSConfig 及 Validate()
│   ├── model/                   # 中间语义模型。File/Message/Field/Enum/Service/RPC + 注解结构体
│   ├── parser/                  # protocompile proto 编译 → model 映射。embeddedResolver 支持 gcode options
│   ├── naming/                  # protobuf-to-Go 命名规则（GoCamelCase、类型名、字段名、标量映射）
│   ├── transform/               # model → Go 中间表示转换（展平、命名计算、类型解析、service 展平）
│   ├── render/                  # Go 源码渲染。File→dao.go, ValidateFile→validate.go, RPCFile→rpc.go, HTTPFile→http.go
│   ├── tsrender/                # TypeScript 源码渲染。TSFile→.pb.ts（interface, enum, enum name mapping, validation metadata）
│   └── source/                  # .proto 文件发现与路径安全校验
├── options/                     # proto 定义（embed 源）+ embed.go。gcode custom options。公开包
├── runtime/                     # protobuf wire format 编码原语。公开包
├── validateruntime/             # 校验运行时。ValidationError、IsEmail/IsURI、MatchPattern。公开包
├── httpruntime/                 # HTTP 运行时。Response/Error 信封、CodedError、DefaultErrorHandler。公开包
├── testdata/
│   └── compat/                  # 兼容性测试套件（wire/validate/update/RPC/HTTP/TS 端到端 + fuzz/bench）
│       ├── bench-results/       # benchmark 历史结果，支持 benchstat 跨版本 diff
│       ├── ts/                  # TS 兼容性测试快照（.pb.ts golden files）
│       └── gen/main.go          # 重新生成所有快照（Go DAO + TS）
├── scripts/
│   └── bench.sh                 # 运行 benchmark 并写入 bench-results/
└── docs/                        # 面向用户的文档（中英双语：README、architecture、getting-started、annotations、design-decisions）
```
