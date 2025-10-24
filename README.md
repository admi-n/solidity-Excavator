# solidity-Excavator


一个工具： 可以自定义编写规则扫描特定合约,包括未开源合约。未开源的反编译可以利用已有开源工具,但是时间有点久。这个后续在写








------------------------note--装修中------------------------------------




go run main.go -d last  下载目前区块到最新区块
go run main.go -d -d-range 10000000-10005000  下载1000到2000区块
go run main.go -d -file eoferror.txt

//暂时先不加合约hash计算,后续反编译的时候遇到余额为0或者hash一样的就不需要反编译了。可以省非常多功夫,百分之99.9的合约没有钱

//或者后续下载完给hash一样,只保留有钱的,其他删掉,然后做个库查询那些一样就可以


## 命令

`go run main.go -ai chatgpt5 -m mode1 -s hourglass-vul -t contract -t-address 0x000 -c eth`
0x000指的某个合约
使用chagpt5模型通过模式1去扫描合约，先判断是否在数据库中,如果在直接使用数据库中的代码,如果不在,下载到数据库中,再去使用.

`go run main.go -ai chatgpt5 -m mode1 -s hourglass-vul -t db -t-block 1-1000 -c eth`

使用chagpt5模型通过模式1去扫描区块1-1000部署的合约关于hourglass-vul的漏洞。(如果不开源,判断是否反编译,如果未反编译,不进行操作扫描并记录)

`go run main.go -ai chatgpt5 -m mode1 -s hourglass-vul -t file -t-file 1.txt -c eth`

使用chagpt5模型通过模式1去扫描1.txt文件中的合约地址合约，先判断这些地址是否在数据库中,如果在：直接使用数据库中的代码。如果不在,将这些合约下载到数据库中,再去进行后续扫描。

-s是指src/strategy/prompts/mode1/hourglass-vul.tmpl这个提示词

-ai 这里是使用的ai模型
-m  扫描模式(比如 mode1:特定类别扫描 (mode1_targeted)：)
-s  提示词
-t (contract/db/file)
    -t contract -t-address 0x000 (先判断合约地址在不在数据库中,如果不在,调用download下载这个合约到数据库中,在进行扫描)
    -t file -t-file 1.txt (扫描1.txt文件中合约(先判断合约地址在不在数据库中,如果不在,调用download下载这个合约到数据库中,在进行扫描。))
    -t db -t-block 1-1000 (扫描数据库中区块为1-1000的合约)
-c (目标链：比如eth,bsc。) (目前只先测试eth)






## 任务
已经完成： cli   download  
Todo：调用ai模块



## 架构

### llm

AI暂时先试用API,因为本地llm效果不太好,不过不排除针对性训练的,这个后话了



### 模式
采用3种模式

1:特定类别扫描 (mode1_targeted)： 根据历史漏洞复现出来,然后给他说漏洞的关键部分,比如复现了town,多层推荐人返佣机制漏洞，就提前写好提示词和参考漏洞代码。然后后面直接给他地址和合约代码去扫

2:模糊类别扫描 (mode2_fuzzy)：    没有复现出来,给他一个大概的详情,让他去扫描有没有类似的漏洞。  (和1有点差不多)  偏向模糊一点

3：泛漏洞扫描 (mode3_general)     没有漏洞名称,什么也没有。 投喂swc和一些严重/高危漏洞。让他自己去扫

### 目录结构

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
│   └── SWC/                               # SWC
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
│   ├── reporter/
│   │   ├── reporter.go         # Reporter 管理器与公共接口
│   │   ├── generator.go        # 高级生成器：把 ScanResult 组装成 ReportData
│   │   ├── renderers/
│   │   │   ├── markdown.go     # 生成 Markdown 报告
│   │   │   ├── html.go         # 生成 HTML 报告
│   │   │   ├── json.go         # 直接输出结构化 JSON
│   │   │   └── pdf.go          # 可选：把 HTML -> PDF（依赖外部工具）
│   │   └──  storage.go          # 将报告写入文件系统/对象存储/DB 的逻辑
│   │   
│   ├── decompiler               #反编译：有点难先不管。。。。
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
    │       └── hourglass-vul.t.sol        # 漏洞复现的exp.
    │
    └── prompts/                           # 💬 提示词模板系统 (AI Prompt Templates)
        ├── builder.go                     # 构建器：根据 mode/策略 动态拼接 Prompt（支持变量替换）
        ├── mode1/
        │   └── hourglass-vul.tmpl         # Mode1 对应漏洞（hourglass）的提示词模板
        ├── mode2/                         # Mode2 模糊模式模板目录
        ├── mode3/                         # Mode3 泛化扫描模板目录
        └── template_loader.go             # 模板加载器：负责读取 .tmpl 文件并注入动态上下文

```

```
层级   	目录	           职责	                                类比
命令层	cmd/	      CLI 参数与入口控制	                    用户界面
配置层	config/	      全局配置与环境变量	                    设置中心
数据层	data/	      静态输入与样本集	                        数据仓库
核心层	internal/	  主逻辑（AI、解析、核心扫描、handler）	    大脑
策略层	strategy/	  Prompt 模板与 EXP 脚本	                智能指导与验证器
主入口	main.go	      程序启动、配置加载	                      启动器
```


