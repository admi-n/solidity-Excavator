# ==== 配置 ====
# ETHERSCAN_API_KEY = "S287KCHRVPZ7439JNJYREKNU1Y135U2F35"
# RPC_URL = "https://rpc.ankr.com/eth/f6d5d2fe5359af3a7d15801f0ec73d5d0d997cadfb50ff072f6e18d5bbfe0103"
import csv
from web3 import Web3
from datetime import datetime
import time
import os

# 配置节点RPC URL
RPC_URL = "https://rpc.ankr.com/eth/f6d5d2fe5359af3a7d15801f0ec73d5d0d997cadfb50ff072f6e18d5bbfe0103"  # 例如: "https://mainnet.infura.io/v3/YOUR_API_KEY"
w3 = Web3(Web3.HTTPProvider(RPC_URL))

# 检查连接
if not w3.is_connected():
    print("❌ 无法连接到区块链节点")
    exit()

print(f"✅ 已连接到区块链，当前区块高度: {w3.eth.block_number}")


def get_contract_info(contract_address):
    """获取合约详细信息"""
    try:
        # 转换地址格式
        address = w3.to_checksum_address(contract_address)

        # 获取合约代码
        code = w3.eth.get_code(address).hex()
        if code == '0x':
            print(f"⚠️  {address} 不是合约地址")
            return None

        # 获取合约余额
        balance = w3.eth.get_balance(address)
        balance_eth = w3.from_wei(balance, 'ether')

        # 这里返回基本信息，创建时间等需要额外API
        return {
            '合约地址': address,
            '合约代码': code,
            '合约余额': f"{balance_eth:.6f}",
            '创建时间': 'N/A',
            '创建区块': 'N/A',
            '最后一次交互时间': 'N/A'
        }

    except Exception as e:
        print(f"❌ 获取合约信息失败 {contract_address}: {str(e)}")
        return None


def scan_blocks_for_contracts(start_block, end_block):
    """扫描区块范围内的合约创建交易"""
    print(f"\n🔍 开始扫描区块 {start_block} 到 {end_block}...")
    contracts = []

    for block_num in range(start_block, end_block + 1):
        try:
            block = w3.eth.get_block(block_num, full_transactions=True)
            print(f"📦 扫描区块 {block_num} (共 {len(block['transactions'])} 笔交易)...")

            block_timestamp = datetime.fromtimestamp(block['timestamp']).strftime('%Y-%m-%d %H:%M:%S')

            for tx in block['transactions']:
                # 如果 to 地址为空，说明是合约创建交易
                if tx['to'] is None:
                    receipt = w3.eth.get_transaction_receipt(tx['hash'])
                    contract_address = receipt['contractAddress']

                    if contract_address:
                        # 获取合约代码
                        code = w3.eth.get_code(contract_address).hex()

                        # 获取合约余额
                        balance = w3.eth.get_balance(contract_address)
                        balance_eth = w3.from_wei(balance, 'ether')

                        contract_info = {
                            '合约地址': contract_address,
                            '合约代码': code,
                            '合约余额': f"{balance_eth:.6f}",
                            '创建时间': block_timestamp,
                            '创建区块': block_num,
                            '最后一次交互时间': block_timestamp
                        }

                        contracts.append(contract_info)
                        print(f"✅ 发现合约: {contract_address} (区块 {block_num})")

            time.sleep(0.1)  # 避免请求过快

        except Exception as e:
            print(f"❌ 扫描区块 {block_num} 失败: {str(e)}")
            continue

    print(f"\n📊 扫描完成，共发现 {len(contracts)} 个合约\n")
    return contracts


def read_addresses_from_txt(filename):
    """从txt文件读取合约地址列表"""
    if not os.path.exists(filename):
        print(f"❌ 文件不存在: {filename}")
        return []

    addresses = []
    with open(filename, 'r', encoding='utf-8') as f:
        for line in f:
            address = line.strip()
            # 跳过空行和注释
            if address and not address.startswith('#'):
                addresses.append(address)

    print(f"📄 从 {filename} 读取到 {len(addresses)} 个地址")
    return addresses


def scan_contract_list(contract_addresses):
    """扫描指定的合约地址列表"""
    print(f"\n🔍 开始扫描 {len(contract_addresses)} 个合约地址...\n")
    contracts = []

    for i, address in enumerate(contract_addresses, 1):
        print(f"[{i}/{len(contract_addresses)}] 扫描合约 {address}...")
        info = get_contract_info(address)
        if info:
            contracts.append(info)
        time.sleep(0.1)  # 避免请求过快

    print(f"\n📊 扫描完成，成功获取 {len(contracts)} 个合约信息\n")
    return contracts


def save_to_csv(contracts, filename='contracts.csv'):
    """保存到CSV文件"""
    if not contracts:
        print("⚠️  没有合约数据可保存")
        return

    fieldnames = ['合约地址', '合约代码', '合约余额', '创建时间', '创建区块', '最后一次交互时间']

    with open(filename, 'w', newline='', encoding='utf-8-sig') as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(contracts)

    print(f"✅ 已保存 {len(contracts)} 个合约信息到 {filename}")


# ========== 使用示例 ==========

if __name__ == "__main__":
    print("="*60)
    print("🚀 区块链合约扫描工具")
    print("="*60)

    # 选择扫描模式
    print("\n请选择扫描模式:")
    print("1. 扫描指定区块范围")
    print("2. 扫描txt文件中的合约地址列表")

    choice = input("\n请输入选项 (1 或 2): ").strip()

    contracts = []

    if choice == "1":
        # 方式1: 扫描指定区块范围
        start = int(input("请输入起始区块号: "))
        end = int(input("请输入结束区块号: "))
        contracts = scan_blocks_for_contracts(start, end)
        output_file = f'contracts_blocks_{start}_{end}.csv'

    elif choice == "2":
        # 方式2: 从txt文件读取地址列表
        txt_file = input("请输入txt文件名 (默认: addresses.txt): ").strip()
        if not txt_file:
            txt_file = "addresses.txt"

        addresses = read_addresses_from_txt(txt_file)
        if addresses:
            contracts = scan_contract_list(addresses)
            output_file = 'contracts_from_list.csv'
        else:
            print("❌ 没有找到有效的地址")

    else:
        print("❌ 无效的选项")
        exit()

    # 保存到CSV
    if contracts:
        output_name = input(f"\n保存文件名 (默认: {output_file}): ").strip()
        if not output_name:
            output_name = output_file
        save_to_csv(contracts, output_name)

    print("\n" + "="*60)
    print("✅ 任务完成!")
    print("="*60)