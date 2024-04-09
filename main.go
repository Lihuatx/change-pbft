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
	nodeNumStr := os.Args[2]
	nodeZStr := os.Args[3]
	nodeNumN, _ := strconv.Atoi(nodeNumStr)
	nodeNumZ, _ := strconv.Atoi(nodeZStr)
	consensus.F = nodeNumN * nodeNumZ / 3
	server := network.NewServer(nodeID)

	server.Start()
}
