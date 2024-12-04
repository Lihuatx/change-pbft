#!/bin/bash

# $1: 节点数量
# $2: 拜占庭节点数量
# $3: 起始节点ID（通常是N0）

n=$1
z=$2
startNode=$3
localhost="0.0.0.0"

# 创建节点表
python3 CreateNodeTable2.py "$n" "$localhost"

# 关闭已存在的tmux会话
tmux kill-session -t myPBFT

# 创建并启动节点
python3 CreatePBFT.py "$startNode" "$n" "$z"

# 启动客户端
python3 linuxTest.py "$startNode"