# solidity-Excavator


一个工具： 可以自定义编写规则扫描特定合约,包括未开源合约。未开源的反编译可以利用已有开源工具,但是时间有点久。这个后续在写








------------------------note--装修中------------------------------------




go run main.go -d last  下载目前区块到最新区块
go run main.go -d -d-range 10000000-10005000  下载1000到2000区块
go run main.go -d -file eoferror.txt

//暂时先不加合约hash计算,后续反编译的时候遇到余额为0或者hash一样的就不需要反编译了。可以省非常多功夫,百分之99.9的合约没有钱

//或者后续下载完给hash一样,只保留有钱的,其他删掉,然后做个库查询那些一样就可以


## 命令

### 基本用法

#### 下载模式
```bash
# 从上次位置继续下载
go run src/main.go -d

# 下载指定区块范围
go run src/main.go -d -d-range 1000-2000

# 只下载文件中的合约地址（独立模式）
go run src/main.go -d -file contracts.txt

# 使用代理下载
go run src/main.go -d -file contracts.txt -proxy http://127.0.0.1:7897
```

#### 扫描模式
```bash
# 扫描单个合约（使用默认提示词模板）
go run src/main.go -ai deepseek -m mode1 -i hourglassvul.toml -t contract -t-address 0x123... -c eth

# 扫描单个合约（指定报告输出目录）
go run src/main.go -ai deepseek -m mode1 -i hourglassvul.toml -t contract -t-address 0x123... -c eth -r ./

# 扫描数据库中的合约
go run src/main.go -ai deepseek -m mode1 -i hourglassvul.toml -t db -t-block 1-1000 -c eth

# 扫描文件中的合约地址
go run src/main.go -ai deepseek -m mode1 -i hourglassvul.toml -t file -t-file contracts.txt -c eth

# 使用代理进行扫描
go run src/main.go -ai deepseek -m mode1 -i hourglassvul.toml -t contract -t-address 0x123... -c eth -proxy http://127.0.0.1:7897
```

### 帮助系统

获取详细帮助信息：

```bash
# 通用帮助
go run main.go --help

# 特定命令帮助
go run main.go -d --help      # 下载模式帮助
go run main.go -ai --help     # AI提供商帮助
go run main.go -m --help       # 扫描模式帮助
go run main.go -s --help       # 扫描策略帮助
go run main.go -t --help       # 扫描目标帮助
go run main.go -c --help       # 区块链网络帮助
```

### 参数说明

-ai 这里是使用的ai模型 (chatgpt5, deepseek, local-llm等)
-m  扫描模式(比如 mode1:特定类别扫描 (mode1_targeted)：)
-s  提示词策略（默认为all，使用default.tmpl模板）
-i  输入文件（如复现代码文件，支持TOML和SOL格式）
-t (contract/db/file)
    -t contract -t-address 0x000 (先判断合约地址在不在数据库中,如果不在,调用download下载这个合约到数据库中,在进行扫描)
    -t file -t-file 1.txt (扫描1.txt文件中合约(先判断合约地址在不在数据库中,如果不在,调用download下载这个合约到数据库中,在进行扫描。))
    -t db -t-block 1-1000 (扫描数据库中区块为1-1000的合约)
-c (目标链：比如eth,bsc。) (目前只先测试eth)
-r 报告输出目录（默认为reports，支持自定义目录如-r ./）
-proxy HTTP代理（如http://127.0.0.1:7897）

-i参数支持简化路径，如-i hourglassvul.toml，系统会自动在src/strategy/exp_libs/mode1/目录查找






## 任务
已经完成： 
- ✅ CLI 命令行接口
- ✅ 下载模块 (download)
- ✅ AI 模块集成
- ✅ Mode1 定向扫描
- ✅ 报告生成系统
- ✅ 代理支持
- ✅ TOML 文件支持

Todo：
- 🔄 Mode2 模糊扫描
- 🔄 Mode3 通用扫描
- 🔄 反编译模块



## 架构

### llm

AI暂时先试用API,因为本地llm效果不太好,不过不排除针对性训练的,这个后话了



### 模式
采用3种模式

1:特定类别扫描 (mode1_targeted)： 根据历史漏洞复现出来,然后给他说漏洞的关键部分,比如复现了town,多层推荐人返佣机制漏洞，就提前写好提示词和参考漏洞代码。然后后面直接给他地址和合约代码去扫

2:模糊类别扫描 (mode2_fuzzy)：    没有复现出来,给他一个大概的详情,让他去扫描有没有类似的漏洞。  (和1有点差不多)  偏向模糊一点

3：泛漏洞扫描 (mode3_general)：   [打算用来审计，投喂大量SWC和漏洞报告。只适合本地模型]。

### 目录结构

```
src/
├── cmd/                                   # 🧠 CLI 层：命令行入口与参数解析
│   ├── cli.go                             # 定义 excavator CLI 的 flag 参数、子命令逻辑与帮助信息
│   ├── command.go                         # CLI 调用的主执行逻辑，解析参数后分派到相应 Handler
│   └── banner.go                          # 程序启动横幅显示
│
├── config/                                # ⚙️ 配置层：运行配置与外部依赖
│   ├── settings.yaml                      # 项目主配置文件（数据库、AI Key、网络节点信息等）
│   └── api_keys.go                        # API密钥管理
│
├── internal/                              # 🔍 核心逻辑层：扫描、AI、解析、处理的内部模块
│   ├── ai/                                # 🤖 AI 模块：统一 AI 客户端和解析逻辑
│   │   ├── ai_manager.go                  # 管理 AI 调用流程，分派至不同 Client 并执行 Prompt 构建与结果解析
│   │   ├── client/                        # 各种 AI 引擎的客户端适配器
│   │   │   ├── chatgpt5_client.go         # ChatGPT-5 模型的具体实现（API 调用/格式化请求）
│   │   │   ├── deepseek_client.go         # DeepSeek AI 模型的具体实现
│   │   │   └── local_llm_client.go       # 本地 LLM（如 Ollama、Claude、Mistral）客户端实现
│   │   ├── client_factory.go              # 根据配置或参数动态创建对应 AI Client 实例
│   │   └── parser/                        # AI 输出解析与结构化模块
│   │       ├── parser.go                  # 实现 AI 输出到结构体的解析逻辑（自然语言 → JSON/结构体）
│   │       └── schema.go                  # 定义统一的漏洞扫描结果结构体（ScanResult、IssueDetail 等）
│   │
│   ├── download/                          # 📥 下载模块：合约代码下载和数据库管理
│   │   ├── download.go                    # 下载器主逻辑，管理区块和合约下载流程
│   │   └── etherscan_helper.go            # Etherscan API 调用和合约源码获取
│   │
│   ├── handler/                           # 🧩 Handler 层：不同扫描模式的工作流组织
│   │   ├── mode1_targeted.go             # Mode1 的完整执行流程：构建 prompt → 调用 AI → 解析 → 输出
│   │   ├── mode2_fuzzy.go                # Mode2 的完整执行流程：模糊匹配漏洞特征（待实现）
│   │   └── mode3_general.go              # Mode3 的完整执行流程：基于 SWC 与高危模式的全面扫描（待实现）
│   │
│   ├── report/                            # 📊 报告模块：扫描结果生成和输出
│   │   ├── generator.go                   # 报告生成器：将扫描结果转换为不同格式
│   │   ├── reporter.go                    # 报告器：整合生成器和存储功能
│   │   ├── storage.go                     # 报告存储：文件系统存储实现
│   │   └── renderers/                     # 报告渲染器
│   │       └── markdown.go               # Markdown 格式报告渲染器
│   │
│   ├── proxy.go                           # 🌐 代理管理：统一 HTTP 代理配置
│   └── types.go                           # 📋 类型定义：核心数据结构定义
│
├── strategy/                              # 🧱 策略层：提示词与复现脚本
│   ├── exp_libs/                          # 🧪 漏洞复现代码库 (Exploit Libraries)
│   │   └── mode1/
│   │       └── hourglassvul.toml         # 漏洞复现的 TOML 格式文件（包含漏洞描述、源码、复现代码）
│   │
│   └── prompts/                           # 💬 提示词模板系统 (AI Prompt Templates)
│       ├── builder.go                     # 构建器：根据 mode/策略 动态拼接 Prompt（支持变量替换）
│       ├── template_loader.go             # 模板加载器：负责读取 .tmpl 文件并注入动态上下文
│       └── mode1/
│           └── default.tmpl               # Mode1 默认提示词模板
│
├── main.go                                # 🚀 项目启动入口，初始化 CLI / Logger / 配置加载
│
└── reports/                               # 📁 报告输出目录（默认）
    └── scan_report_*.md                   # 生成的 Markdown 格式扫描报告

```

```
层级   	目录	           职责	                                类比
命令层	cmd/	      CLI 参数与入口控制	                    用户界面
配置层	config/	      全局配置与环境变量	                    设置中心
核心层	internal/	  主逻辑（AI、解析、下载、报告、handler）	    大脑
策略层	strategy/	  Prompt 模板与 EXP 脚本	                智能指导与验证器
报告层	reports/	      扫描结果输出	                        输出仓库
主入口	main.go	      程序启动、配置加载	                      启动器
```


