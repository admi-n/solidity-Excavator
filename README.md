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


solidity-ai-scanner/src
    ├── cmd/                      # 1. 执行入口
    │   └── scanner.go
    │
    ├── internal/                 # 2. 核心业务逻辑 (Handler 模式实现)
    │   ├── handler/              # 模式 1, 2, 3的 Handler 实现
    │   ├── core/                 # 通用扫描引擎和分析器
    │   └── ai/                   # AI 客户端和适配器
    │
    ├── config/                   # 3. 配置文件和环境设置
    │   ├── settings.yaml
    │   └── api_keys.env.example
    │
    ├── data/                     # 4. 静态数据集 (待扫描的输入源)
    │   ├── source_contracts/     # 可扫描的子目录结构 (project_a/, project_b/)
    │   └── benchmarks/             测试和性能基准（Benchmark）的样本数据
    │
    ├── strategy/                 # 5. 扫描策略和指导文件 (Prompt/EXP)
    │   ├── prompts/              # AI Prompt 模板
    │   │         └── mode1             
    │   │               └── hourglass-vul.tmpl      hourglass-vul漏洞提示词
    │   │  
    │   └── exp_libs/             # 漏洞复现代码/自定义函数     这个用yaml还是什么格式的
    │         └── mode1             
    │                └── hourglass-vul.t.sol     hourglass-vul漏洞复现exp
    │  
    └── go.mod



