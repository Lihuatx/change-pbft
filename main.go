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

		server.Start()
	}

}
