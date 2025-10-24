package download

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/admi-n/solidity-Excavator/src/config"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

// ContractInfo 合约信息结构体
type ContractInfo struct {
	Address      string
	Contract     string
	Balance      string
	IsOpenSource int
	CreateTime   time.Time
	CreateBlock  uint64
	TxLast       time.Time
	IsDecompiled int
	DedCode      string
}

// Downloader 下载器
type Downloader struct {
	Client          *ethclient.Client
	db              *sql.DB
	etherscanConfig EtherscanConfig
	rateLimiter     *RateLimiter
}

// NewDownloader 创建下载器（使用配置文件中的 RPC URL）
// 新增 proxy 参数，若 proxy 非空，会设置全局 HTTP Transport 的代理并传入 etherscan 配置
func NewDownloader(db *sql.DB, proxy string) (*Downloader, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库连接不能为 nil")
	}

	// 如果传入 proxy，则设置全局默认 transport 的代理（影响 HTTP 客户端以及 ethclient 使用的默认 transport）
	if strings.TrimSpace(proxy) != "" {
		u, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("解析 proxy URL 失败: %w", err)
		}
		// 设置全局 DefaultTransport 为带 proxy 的 Transport（保留默认超时等其他字段可以按需调整）
		http.DefaultTransport = &http.Transport{
			Proxy: http.ProxyURL(u),
			// 其它字段采用零值或按需设置（可根据需要添加 TLSHandshakeTimeout 等）
		}
	}

	// 从配置文件获取 RPC URL
	rpcURL, err := config.GetRPCURL()
	if err != nil {
		return nil, fmt.Errorf("获取 RPC URL 失败: %w", err)
	}

	// 连接以太坊节点（使用默认 transport，若上面设置了 proxy，则会生效）
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("连接以太坊节点失败: %w", err)
	}

	log.Printf("✅ 成功连接到以太坊节点: %s\n", rpcURL)

	// 初始化 etherscan 配置（从 config 常量读取），并注入 proxy
	ethersCfg := EtherscanConfig{
		APIKey:  config.EtherscanAPIKey,
		BaseURL: config.EtherscanBaseURL,
		Proxy:   strings.TrimSpace(proxy),
	}

	return &Downloader{
		Client:          client,
		db:              db,
		etherscanConfig: ethersCfg,
		rateLimiter:     NewRateLimiter(5), // 可调整速率
	}, nil
}

// GetCurrentBlock 获取当前最新区块号
func (d *Downloader) GetCurrentBlock(ctx context.Context) (uint64, error) {
	return d.Client.BlockNumber(ctx)
}

// GetLastDownloadedBlock 获取数据库中最后下载的区块号
func (d *Downloader) GetLastDownloadedBlock(ctx context.Context) (uint64, error) {
	var maxBlock sql.NullInt64
	err := d.db.QueryRowContext(ctx, "SELECT MAX(createblock) FROM contracts").Scan(&maxBlock)
	if err != nil {
		return 0, fmt.Errorf("查询最后下载区块失败: %w", err)
	}
	if !maxBlock.Valid {
		return 0, nil
	}
	return uint64(maxBlock.Int64), nil
}

// ContractExists 检查合约是否已存在
func (d *Downloader) ContractExists(ctx context.Context, address string) (bool, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM contracts WHERE address = ?", address).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SaveContract 保存合约信息到数据库
func (d *Downloader) SaveContract(ctx context.Context, info *ContractInfo) error {
	query := `
	INSERT INTO contracts (address, contract, balance, isopensource, createtime, createblock, txlast, isdecompiled, dedcode)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE 
		contract = VALUES(contract),
		balance = VALUES(balance),
		isopensource = VALUES(isopensource),
		txlast = VALUES(txlast),
		isdecompiled = VALUES(isdecompiled),
		dedcode = VALUES(dedcode)
	`

	_, err := d.db.ExecContext(ctx, query,
		info.Address,
		info.Contract,
		info.Balance,
		info.IsOpenSource,
		info.CreateTime,
		int64(info.CreateBlock),
		info.TxLast,
		info.IsDecompiled,
		info.DedCode,
	)

	return err
}

// IsBlockDownloaded 检查区块是否已下载
func (d *Downloader) IsBlockDownloaded(ctx context.Context, blockNum uint64) (bool, error) {
	var count int
	err := d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM contracts WHERE createblock = ?", int64(blockNum)).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// helper: 将失败地址追加到文件（每行一个），忽略写入错误但记录日志
func appendFailAddress(failFile, addr string) {
	if strings.TrimSpace(failFile) == "" || strings.TrimSpace(addr) == "" {
		return
	}
	f, err := os.OpenFile(failFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Printf("⚠️  无法打开失败记录文件 %s: %v\n", failFile, err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(strings.TrimSpace(addr) + "\n"); err != nil {
		log.Printf("⚠️  无法写入失败记录文件 %s: %v\n", failFile, err)
	}
}

// 新增：记录已下载区块区间的结构与文件路径
type BlockRangeRecord struct {
	Start uint64 `json:"start"`
	End   uint64 `json:"end"`
}

const blockedFile = "blocked.json" // 可修改为其他路径

// loadBlockedRanges 从 blocked.json 读取已下载区间（若文件不存在，返回空切片）
func loadBlockedRanges() ([]BlockRangeRecord, error) {
	bs, err := os.ReadFile(blockedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取 %s 失败: %w", blockedFile, err)
	}
	var recs []BlockRangeRecord
	if err := json.Unmarshal(bs, &recs); err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", blockedFile, err)
	}
	return recs, nil
}

// saveBlockedRanges 将合并后的区间写回 blocked.json
func saveBlockedRanges(recs []BlockRangeRecord) error {
	bs, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 blocked ranges 失败: %w", err)
	}
	if err := os.WriteFile(blockedFile, bs, 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", blockedFile, err)
	}
	return nil
}

// mergeAndInsertRange 将 newRange 并入 existing 并返回合并后的区间列表（按 start 排序且不重叠）
func mergeAndInsertRange(existing []BlockRangeRecord, newRange BlockRangeRecord) []BlockRangeRecord {
	existing = append(existing, newRange)
	if len(existing) == 0 {
		return existing
	}
	// 按 Start 排序
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].Start < existing[j].Start
	})
	merged := make([]BlockRangeRecord, 0, len(existing))
	curr := existing[0]
	for i := 1; i < len(existing); i++ {
		r := existing[i]
		if r.Start <= curr.End+1 { // 重叠或相邻 -> 合并
			if r.End > curr.End {
				curr.End = r.End
			}
		} else {
			merged = append(merged, curr)
			curr = r
		}
	}
	merged = append(merged, curr)
	return merged
}

// getUncoveredRanges 返回请求区间 requestRange 在 existingRanges 中未覆盖的子区间列表（按升序）
func getUncoveredRanges(existingRanges []BlockRangeRecord, requestStart, requestEnd uint64) []BlockRangeRecord {
	if requestStart > requestEnd {
		return nil
	}
	// 无已存在区间，直接返回请求区间
	if len(existingRanges) == 0 {
		return []BlockRangeRecord{{Start: requestStart, End: requestEnd}}
	}
	// 先合并 existingRanges 以简化计算
	sort.Slice(existingRanges, func(i, j int) bool { return existingRanges[i].Start < existingRanges[j].Start })
	merged := []BlockRangeRecord{}
	curr := existingRanges[0]
	for i := 1; i < len(existingRanges); i++ {
		r := existingRanges[i]
		if r.Start <= curr.End+1 {
			if r.End > curr.End {
				curr.End = r.End
			}
		} else {
			merged = append(merged, curr)
			curr = r
		}
	}
	merged = append(merged, curr)

	var out []BlockRangeRecord
	cursor := requestStart
	for _, r := range merged {
		// 如果当前已合并区间在请求区间左侧且不重叠，跳过
		if r.End < cursor {
			continue
		}
		// 如果合并区间开始在请求结束之后，剩余整个区间都是未覆盖
		if r.Start > requestEnd {
			break
		}
		// 有未覆盖段
		if r.Start > cursor {
			end := minUint64(r.Start-1, requestEnd)
			if cursor <= end {
				out = append(out, BlockRangeRecord{Start: cursor, End: end})
			}
		}
		// 推进 cursor 到合并区间之后
		if r.End+1 > cursor {
			cursor = r.End + 1
		}
		if cursor > requestEnd {
			break
		}
	}
	// 剩下的尾部
	if cursor <= requestEnd {
		out = append(out, BlockRangeRecord{Start: cursor, End: requestEnd})
	}
	return out
}

func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

// DownloadBlockRange 下载指定区块范围的合约（改为按未覆盖子区间下载并记录 blocked.json）
func (d *Downloader) DownloadBlockRange(ctx context.Context, startBlock, endBlock uint64) error {
	log.Printf("🔍 开始下载区块 %d 到 %d...\n", startBlock, endBlock)

	// 读取已下载区间记录
	existing, err := loadBlockedRanges()
	if err != nil {
		log.Printf("⚠️  读取已下载区间失败: %v（继续，但可能重复下载）\n", err)
		// 继续使用 empty existing
		existing = nil
	}

	uncovered := getUncoveredRanges(existing, startBlock, endBlock)
	if len(uncovered) == 0 {
		log.Printf("✅ 请求区间 [%d-%d] 已全部下载，跳过\n", startBlock, endBlock)
		return nil
	}

	totalContracts := 0
	skippedBlocks := 0

	// 对每个未覆盖的子区间依次下载，子区间完成后合并写入 blocked.json
	for _, sub := range uncovered {
		log.Printf("🔁 处理未覆盖子区间: %d - %d\n", sub.Start, sub.End)
		for blockNum := sub.Start; blockNum <= sub.End; blockNum++ {
			// 检查区块是否已在数据库（谨慎双重判断）
			downloaded, err := d.IsBlockDownloaded(ctx, blockNum)
			if err != nil {
				log.Printf("⚠️  检查区块 %d 状态失败: %v\n", blockNum, err)
			} else if downloaded {
				skippedBlocks++
				if skippedBlocks%100 == 0 {
					log.Printf("⏭️  已跳过 %d 个已下载的区块...\n", skippedBlocks)
				}
				continue
			}

			// 获取区块数据
			block, err := d.Client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
			if err != nil {
				log.Printf("❌ 获取区块 %d 失败: %v\n", blockNum, err)
				continue
			}

			txCount := len(block.Transactions())
			if txCount > 0 {
				log.Printf("📦 处理区块 %d (共 %d 笔交易)...\n", blockNum, txCount)
			}

			blockTime := time.Unix(int64(block.Time()), 0)
			contractCount := 0

			// 遍历交易查找合约创建
			for _, tx := range block.Transactions() {
				// 合约创建交易的 To 地址为 nil
				if tx.To() == nil {
					receipt, err := d.Client.TransactionReceipt(ctx, tx.Hash())
					if err != nil {
						log.Printf("⚠️  获取交易收据失败: %v\n", err)
						continue
					}

					if receipt.ContractAddress != (common.Address{}) {
						contractAddr := receipt.ContractAddress.Hex()

						// 再次检查合约是否已存在
						exists, err := d.ContractExists(ctx, contractAddr)
						if err != nil {
							log.Printf("❌ 检查合约存在失败: %v\n", err)
							continue
						}

						if exists {
							continue
						}

						// 获取合约字节码
						code, err := d.Client.CodeAt(ctx, receipt.ContractAddress, nil)
						if err != nil {
							log.Printf("⚠️  获取合约代码失败: %v\n", err)
							continue
						}

						// 声明用于保存的变量，确保在所有分支都有初始值
						var contractCode string
						var isOpenSource int

						// 清理地址，去除可能的空格/换行/不可见字符，避免在 URL 拼接时出现问题
						contractAddr = strings.TrimSpace(contractAddr)

						// 检查 Etherscan 验证状态（若未配置 APIKey 则直接回退为字节码）
						if d.etherscanConfig.APIKey != "" {
							sourceCode, isVerified, err := GetContractSource(contractAddr, d.etherscanConfig)
							if err != nil {
								// 查询失败时回退为字节码并记录日志
								log.Printf("⚠️  查询 Etherscan 失败: %v，回退保存字节码\n", err)
								contractCode = fmt.Sprintf("0x%x", code)
								isOpenSource = 0
								// 记录到失败文件
								appendFailAddress("eoferror.txt", contractAddr)
							} else if isVerified {
								contractCode = sourceCode // 保存源代码
								isOpenSource = 1          // 标记为已开源
							} else {
								contractCode = fmt.Sprintf("0x%x", code) // 保存字节码
								isOpenSource = 0                         // 标记为未开源
							}
						} else {
							// 未配置 Etherscan API key，直接保存字节码
							contractCode = fmt.Sprintf("0x%x", code)
							isOpenSource = 0
						}

						// 获取合约余额
						balance, err := d.Client.BalanceAt(ctx, receipt.ContractAddress, nil)
						if err != nil {
							log.Printf("⚠️  获取余额失败: %v\n", err)
							balance = big.NewInt(0)
						}

						balanceEth := new(big.Float).Quo(
							new(big.Float).SetInt(balance),
							big.NewFloat(1e18),
						)

						// 构造合约信息（使用上面确定的 contractCode 与 isOpenSource）
						info := &ContractInfo{
							Address:      contractAddr,
							Contract:     contractCode,
							Balance:      balanceEth.Text('f', 6),
							IsOpenSource: isOpenSource,
							CreateTime:   blockTime,
							CreateBlock:  blockNum,
							TxLast:       blockTime,
							IsDecompiled: 0,  // 默认未反编译
							DedCode:      "", // 默认空
						}

						// 保存到数据库
						if err := d.SaveContract(ctx, info); err != nil {
							log.Printf("❌ 保存合约失败: %v\n", err)
							continue
						}

						contractCount++
						totalContracts++
						log.Printf("✅ 发现合约: %s (区块 %d)\n", contractAddr, blockNum)
					}
				}
			}

			// 避免请求过快
			time.Sleep(50 * time.Millisecond)
		} // end for blockNum in subrange

		// 子区间完成后，将其合并写入 blocked.json
		merged := mergeAndInsertRange(existing, BlockRangeRecord{Start: sub.Start, End: sub.End})
		if err := saveBlockedRanges(merged); err != nil {
			log.Printf("⚠️  保存已下载区间到 %s 失败: %v\n", blockedFile, err)
		} else {
			// 更新内存 existing 为最新（避免下一个子区间重复计算）
			existing = merged
		}
	} // end for each sub

	log.Printf("\n✅ 下载完成!\n")
	log.Printf("   - 区块范围: %d - %d\n", startBlock, endBlock)
	log.Printf("   - 新增合约: %d\n", totalContracts)
	log.Printf("   - 跳过区块: %d\n", skippedBlocks)

	return nil
}

// DownloadFromLast 从最后下载的区块继续下载到最新区块
func (d *Downloader) DownloadFromLast(ctx context.Context) error {
	// 获取最后下载的区块
	lastBlock, err := d.GetLastDownloadedBlock(ctx)
	if err != nil {
		return fmt.Errorf("获取最后下载区块失败: %w", err)
	}

	// 获取当前最新区块
	currentBlock, err := d.GetCurrentBlock(ctx)
	if err != nil {
		return fmt.Errorf("获取当前区块失败: %w", err)
	}

	// 如果没有下载记录,从创世区块开始
	startBlock := lastBlock + 1
	if lastBlock == 0 {
		startBlock = 0
		log.Println("📌 数据库为空,从创世区块开始下载")
	} else {
		log.Printf("📌 从区块 %d 继续下载 (上次: %d)\n", startBlock, lastBlock)
	}

	log.Printf("🎯 目标区块: %d (当前最新)\n", currentBlock)

	if startBlock > currentBlock {
		log.Println("✅ 已经是最新,无需下载")
		return nil
	}

	// 开始下载
	return d.DownloadBlockRange(ctx, startBlock, currentBlock)
}

// Close 关闭连接
func (d *Downloader) Close() {
	if d.Client != nil {
		d.Client.Close()
	}
}

// 新增：按地址列表下载合约（用于 -d -file <file> 重试）
func (d *Downloader) DownloadContractsByAddresses(ctx context.Context, addresses []string, failLog string) error {
	if len(addresses) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	for _, a := range addresses {
		addr := strings.TrimSpace(a)
		if addr == "" {
			continue
		}
		// 去重
		if _, ok := seen[strings.ToLower(addr)]; ok {
			continue
		}
		seen[strings.ToLower(addr)] = struct{}{}

		// 检查是否已存在
		exists, err := d.ContractExists(ctx, addr)
		if err != nil {
			log.Printf("⚠️  检查合约 %s 是否存在失败: %v\n", addr, err)
			appendFailAddress(failLog, addr)
			continue
		}
		if exists {
			log.Printf("⏭️  合约已存在，跳过: %s\n", addr)
			continue
		}

		// 转换为 common.Address
		caddr := common.HexToAddress(addr)

		// 获取合约字节码
		code, err := d.Client.CodeAt(ctx, caddr, nil)
		if err != nil {
			log.Printf("⚠️  获取合约字节码失败: %s -> %v\n", addr, err)
			appendFailAddress(failLog, addr)
			continue
		}

		// 默认值
		contractCode := fmt.Sprintf("0x%x", code)
		isOpenSource := 0

		// 如果配置了 Etherscan APIKey，尝试获取源码；网络错误时将地址写入失败文件
		if d.etherscanConfig.APIKey != "" {
			sourceCode, isVerified, err := GetContractSource(addr, d.etherscanConfig)
			if err != nil {
				log.Printf("⚠️  查询 Etherscan 失败 for %s: %v，回退保存字节码并记录到失败文件\n", addr, err)
				appendFailAddress(failLog, addr)
				// 回退保存字节码（contractCode 已为字节码）
			} else if isVerified {
				contractCode = sourceCode
				isOpenSource = 1
			} else {
				// 未验证，保持字节码
			}
		}

		// 获取余额（不阻塞主流程，失败则置零）
		balance, err := d.Client.BalanceAt(ctx, caddr, nil)
		if err != nil {
			log.Printf("⚠️  获取余额失败: %s -> %v\n", addr, err)
			balance = big.NewInt(0)
		}
		balanceEth := new(big.Float).Quo(new(big.Float).SetInt(balance), big.NewFloat(1e18))

		info := &ContractInfo{
			Address:      addr,
			Contract:     contractCode,
			Balance:      balanceEth.Text('f', 6),
			IsOpenSource: isOpenSource,
			CreateTime:   time.Now(),
			CreateBlock:  0,
			TxLast:       time.Now(),
			IsDecompiled: 0,
			DedCode:      "",
		}

		// 保存到数据库
		if err := d.SaveContract(ctx, info); err != nil {
			log.Printf("❌ 保存合约失败: %s -> %v\n", addr, err)
			appendFailAddress(failLog, addr)
			continue
		}

		log.Printf("✅ 重试下载合约成功: %s\n", addr)

		// 为了避免速率过快，可在此加入短暂停顿或使用 d.rateLimiter.Wait()
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}
