package config

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/admi-n/solidity-Excavator/src/internal"
	_ "github.com/go-sql-driver/mysql"
)

// 数据库配置信息 - 直接在这里修改
const (
	DBHost     = "localhost"
	DBPort     = "3306"
	DBUser     = "root"
	DBPassword = "123456"
	DBName     = "solidity_excavator"
)

// RPC 配置信息 - 直接在这里修改
const (
	RPCURL = "https://rpc.ankr.com/eth/f6d5d2fe5359af3a7d15801f0ec73d5d0d997cadfb50ff072f6e18d5bbfe0103"
)

const (
	EtherscanAPIKey  = "S287KCHRVPZ7439JNJYREKNU1Y135U2F35"
	EtherscanBaseURL = "https://api.etherscan.io/v2"
)

// 全局连接池
var DBPool *sql.DB

// InitDB 初始化 MySQL 连接池并 ping 验证
func InitDB() (*sql.DB, error) {
	// 构建 DSN: username:password@tcp(host:port)/dbname?parseTime=true
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		DBUser, DBPassword, DBHost, DBPort, DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("InitDB: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// 验证连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("InitDB ping failed: %w", err)
	}

	DBPool = db
	return db, nil
}

// GetContracts 从 contracts 表读取记录，limit<=0 表示不限制
func GetContracts(ctx context.Context, db *sql.DB, limit int) ([]internal.Contract, error) {
	if db == nil {
		return nil, fmt.Errorf("GetContracts: db is nil")
	}

	query := "SELECT address, contract, balance, isopensource, createtime, createblock, txlast, isdecompiled, dedcode FROM contracts"
	var rows *sql.Rows
	var err error

	//if limit > 0 {
	//	query += " LIMIT ?"
	//	rows, err = db.QueryContext(ctx, query, limit)
	//} else {
	//	rows, err = db.QueryContext(ctx, query)
	//}
	if limit > 0 {
		query = fmt.Sprintf("%s LIMIT %d", query, limit)
		rows, err = db.QueryContext(ctx, query)
	} else {
		rows, err = db.QueryContext(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []internal.Contract
	for rows.Next() {
		var c internal.Contract
		var isOpenInt int
		var isDecompiledInt int
		var createBlock int64
		var createTime time.Time
		var txLast time.Time
		var balance string
		var dedCode sql.NullString

		if err := rows.Scan(&c.Address, &c.Code, &balance, &isOpenInt, &createTime, &createBlock, &txLast, &isDecompiledInt, &dedCode); err != nil {
			return nil, err
		}

		c.Balance = balance
		c.IsOpenSource = isOpenInt != 0
		c.CreateTime = createTime
		c.CreateBlock = uint64(createBlock)
		c.TxLast = txLast
		c.IsDecompiled = isDecompiledInt != 0
		if dedCode.Valid {
			c.DedCode = dedCode.String
		}

		out = append(out, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// GetContractsByAddresses 根据地址数组批量查询
func GetContractsByAddresses(ctx context.Context, db *sql.DB, addresses []string) ([]internal.Contract, error) {
	if db == nil {
		return nil, fmt.Errorf("GetContractsByAddresses: db is nil")
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("GetContractsByAddresses: addresses empty")
	}

	// 构建 IN 查询的占位符
	placeholders := make([]string, len(addresses))
	args := make([]interface{}, len(addresses))
	for i, addr := range addresses {
		placeholders[i] = "?"
		args[i] = addr
	}

	query := fmt.Sprintf("SELECT address, contract, balance, isopensource, createtime, createblock, txlast, isdecompiled, dedcode FROM contracts WHERE address IN (%s)",
		joinStrings(placeholders, ","))

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []internal.Contract
	for rows.Next() {
		var c internal.Contract
		var isOpenInt int
		var isDecompiledInt int
		var createBlock int64
		var createTime time.Time
		var txLast time.Time
		var balance string
		var dedCode sql.NullString

		if err := rows.Scan(&c.Address, &c.Code, &balance, &isOpenInt, &createTime, &createBlock, &txLast, &isDecompiledInt, &dedCode); err != nil {
			return nil, err
		}

		c.Balance = balance
		c.IsOpenSource = isOpenInt != 0
		c.CreateTime = createTime
		c.CreateBlock = uint64(createBlock)
		c.TxLast = txLast
		c.IsDecompiled = isDecompiledInt != 0
		if dedCode.Valid {
			c.DedCode = dedCode.String
		}

		out = append(out, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

// GetRPCURL 返回配置的 RPC URL
func GetRPCURL() (string, error) {
	if RPCURL == "" {
		return "", fmt.Errorf("RPC_URL 未配置：请在 config.go 中设置 RPCURL 常量")
	}
	return RPCURL, nil
}

// joinStrings 辅助函数：连接字符串数组
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
