package download

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/big"
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
	client *ethclient.Client
	db     *sql.DB
}

// NewDownloader 创建下载器（使用配置文件中的 RPC URL）
func NewDownloader(db *sql.DB) (*Downloader, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库连接不能为 nil")
	}

	// 从配置文件获取 RPC URL
	rpcURL, err := config.GetRPCURL()
	if err != nil {
		return nil, fmt.Errorf("获取 RPC URL 失败: %w", err)
	}

	// 连接以太坊节点
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("连接以太坊节点失败: %w", err)
	}

	log.Printf("✅ 成功连接到以太坊节点: %s\n", rpcURL)

	return &Downloader{
		client: client,
		db:     db,
	}, nil
}

// GetCurrentBlock 获取当前最新区块号
func (d *Downloader) GetCurrentBlock(ctx context.Context) (uint64, error) {
	return d.client.BlockNumber(ctx)
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

// DownloadBlockRange 下载指定区块范围的合约
func (d *Downloader) DownloadBlockRange(ctx context.Context, startBlock, endBlock uint64) error {
	log.Printf("🔍 开始下载区块 %d 到 %d...\n", startBlock, endBlock)

	totalContracts := 0
	skippedBlocks := 0

	for blockNum := startBlock; blockNum <= endBlock; blockNum++ {
		// 检查区块是否已下载
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
		block, err := d.client.BlockByNumber(ctx, big.NewInt(int64(blockNum)))
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
				receipt, err := d.client.TransactionReceipt(ctx, tx.Hash())
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

					// 获取合约代码
					code, err := d.client.CodeAt(ctx, receipt.ContractAddress, nil)
					if err != nil {
						log.Printf("⚠️  获取合约代码失败: %v\n", err)
						continue
					}

					// 获取合约余额
					balance, err := d.client.BalanceAt(ctx, receipt.ContractAddress, nil)
					if err != nil {
						log.Printf("⚠️  获取余额失败: %v\n", err)
						balance = big.NewInt(0)
					}

					balanceEth := new(big.Float).Quo(
						new(big.Float).SetInt(balance),
						big.NewFloat(1e18),
					)

					// 构造合约信息
					info := &ContractInfo{
						Address:      contractAddr,
						Contract:     fmt.Sprintf("0x%x", code),
						Balance:      balanceEth.Text('f', 6),
						IsOpenSource: 0, // 默认未开源
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

		// 进度报告
		if blockNum%100 == 0 || contractCount > 0 {
			progress := float64(blockNum-startBlock+1) / float64(endBlock-startBlock+1) * 100
			log.Printf("📊 进度: %.2f%% (区块 %d/%d, 累计合约: %d)\n",
				progress, blockNum, endBlock, totalContracts)
		}

		// 避免请求过快
		time.Sleep(50 * time.Millisecond)
	}

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
		startBlock = 10000000 //默认10000000   一千万区块开始
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
	if d.client != nil {
		d.client.Close()
	}
}
