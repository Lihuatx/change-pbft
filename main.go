package main

import (
	"os"
	"simple_pbft/pbft/consensus"
	"strconv"

	"simple_pbft/pbft/network"
)

func main() {
	genRsaKeys()
	nodeID := os.Args[1]
	sendMsgNumber := 500
	network.SendMsgNumber = sendMsgNumber
	if nodeID == "client" {
		client := network.ClientStart("N")

		go client.SendMsg(sendMsgNumber)

		client.Start()
	} else {
		nodeNumStr := os.Args[2]
		nodeZStr := os.Args[3]
		network.StartNodeID = os.Args[4]
		nodeNumN, _ := strconv.Atoi(nodeNumStr)
		nodeNumZ, _ := strconv.Atoi(nodeZStr)
		network.EndNodeNum = nodeNumN * nodeNumZ
		consensus.F = nodeNumN * nodeNumZ / 3
		server := network.NewServer(nodeID)

		// 检查是否提供了第5个参数
		if len(os.Args) > 5 { // 判断节点是正常节点还是恶意节点
			network.IsMaliciousNode = os.Args[5] // 使用提供的第三个参数
		}

		server.Start()
	}

}
