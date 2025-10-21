package cmd

import "fmt"

func Print() {
	logo := `
▗▄▄▄▖▄   ▄ ▗▞▀▘▗▞▀▜▌▄   ▄ ▗▞▀▜▌   ■   ▄▄▄   ▄▄▄ 
▐▌    ▀▄▀  ▝▚▄▖▝▚▄▟▌█   █ ▝▚▄▟▌▗▄▟▙▄▖█   █ █    
▐▛▀▀▘▄▀ ▀▄           ▀▄▀         ▐▌  ▀▄▄▄▀ █    
▐▙▄▄▖                            ▐▌             
                                 ▐▌             

   Solidity Excavator - AI Based Vulnerability Scanner
`
	fmt.Println(logo)
}
