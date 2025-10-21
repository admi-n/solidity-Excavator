// cli
package cmd

// -t 扫描的时候是在数据库或者还是在本地yaml获取,可以每一个yaml固定1000个合约那样

//excavator -h
//excavator -ai(选择的ai) -m(模式) -s（策略）(prompts/mode1下的提示词,可选all) -t(目标)[-block(是区间,默认1-xx)/-file(yaml格式) ]   -c(可选,默认eth网络) (eth/bsc/arb)
//excavator -ai chatgpt5 -m mode1 -s hourglass-vul -t-block 1-220234   -c eth
