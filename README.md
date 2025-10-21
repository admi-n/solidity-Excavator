# solidity-Excavator


数据集的格式应该要哪一种,是丢数据库吗还是yaml。eth_0x0000.yaml

格式包含信息：  创建时间/区块   合约地址   合约代码  合约余额(这个不好判断,因为有的是eth有的是有其他币的)  交互数量  快照最后一次交易截止日期



## 架构

### llm
AI暂时先试用API,因为本地llm效果不太好,不过不排除针对性训练的,这个后话了


### 模式
采用3种模式
1:特定类别扫描 (mode1_targeted)： 根据历史漏洞复现出来,然后给他说漏洞的关键部分,比如复现了town,多层推荐人返佣机制漏洞，就提前写好提示词和参考漏洞代码。然后后面直接给他地址和合约代码去扫

2:模糊类别扫描 (mode2_fuzzy)：    没有复现出来,给他一个大概的详情,让他去扫描有没有类似的漏洞。  (和1有点差不多)  偏向模糊一点

3：泛漏洞扫描 (mode3_general)     没有漏洞名称,什么也没有。 投喂swc和一些严重/高危漏洞。让他自己去扫

```
src/
├── cmd/                                   # 🧠 CLI 层：命令行入口与参数解析
│   ├── cli.go                             # 定义 excavator CLI 的 flag 参数、子命令逻辑与帮助信息
│   └── scanner.go                         # CLI 调用的主执行逻辑，解析参数后分派到相应 Handler（mode1/2/3）
│
├── config/                                # ⚙️ 配置层：运行配置与外部依赖
│   └── settings.yaml                      # 项目主配置文件（数据库、AI Key、网络节点信息等）
│
├── data/                                  # 🧾 数据层：静态输入和基准数据集
│   ├── benchmarks/                        # 性能与正确性基准数据（用于测试扫描准确率）
│   └── source_contracts/                  # 待扫描合约源文件，可按项目/来源分子目录
│
├── internal/                              # 🔍 核心逻辑层：扫描、AI、解析、处理的内部模块
│   ├── ai/                                # 🤖 AI 模块：统一 AI 客户端和解析逻辑
│   │   ├── ai_manager.go                  # 管理 AI 调用流程，分派至不同 Client 并执行 Prompt 构建与结果解析
│   │   ├── client/                        # 各种 AI 引擎的客户端适配器
│   │   │   ├── chatgpt5_client.go         # ChatGPT-5 模型的具体实现（API 调用/格式化请求）
│   │   │   └── local_llm_client.go        # 本地 LLM（如 Ollama、Claude、Mistral）客户端实现
│   │   ├── client_factory.go              # 根据配置或参数动态创建对应 AI Client 实例
│   │   └── parser/                        # AI 输出解析与结构化模块
│   │       ├── parser.go                  # 实现 AI 输出到结构体的解析逻辑（自然语言 → JSON/结构体）
│   │       └── schema.go                  # 定义统一的漏洞扫描结果结构体（ScanResult、IssueDetail 等）
│   │
│   ├── core/                              # ⚡ 扫描核心逻辑：不同模式的引擎实现
│   │   ├── mode1.go                       # mode1_targeted 精确扫描逻辑（复现类漏洞）
│   │   ├── mode2.go                       # mode2_fuzzy 模糊扫描逻辑（相似漏洞检索）
│   │   └── mode3.go                       # mode3_general 泛化扫描逻辑（全局漏洞模式）
│   │
│   └── handler/                           # 🧩 Handler 层：不同扫描模式的工作流组织
│       ├── mode1_targeted.go              # Mode1 的完整执行流程：构建 prompt → 调用 AI → 解析 → 输出
│       ├── mode2_fuzzy.go                 # Mode2 的完整执行流程：模糊匹配漏洞特征
│       └── mode3_general.go               # Mode3 的完整执行流程：基于 SWC 与高危模式的全面扫描
│
├── main.go                                # 🚀 项目启动入口，初始化 CLI / Logger / 配置加载
│
└── strategy/                              # 🧱 策略层：提示词与复现脚本
    ├── exp_libs/                          # 🧪 漏洞复现代码库 (Exploit Libraries)
    │   └── mode1/
    │       └── hourglass-vul.t.sol        # hourglass 漏洞复现 Solidity 脚本，可用于验证 AI 检测结果
    │
    └── prompts/                           # 💬 提示词模板系统 (AI Prompt Templates)
        ├── builder.go                     # 构建器：根据 mode/策略 动态拼接 Prompt（支持变量替换）
        ├── mode1/
        │   └── hourglass-vul.tmpl         # Mode1 对应漏洞（hourglass）的提示词模板
        ├── mode2/                         # Mode2 模糊模式模板目录
        ├── mode3/                         # Mode3 泛化扫描模板目录
        └── template_loader.go             # 模板加载器：负责读取 .tmpl 文件并注入动态上下文

```

层级   	目录	           职责	                                类比
命令层	cmd/	      CLI 参数与入口控制	                    用户界面
配置层	config/	      全局配置与环境变量	                    设置中心
数据层	data/	      静态输入与样本集	                        数据仓库
核心层	internal/	  主逻辑（AI、解析、核心扫描、handler）	    大脑
策略层	strategy/	  Prompt 模板与 EXP 脚本	                智能指导与验证器
主入口	main.go	      程序启动、配置加载	                      启动器


