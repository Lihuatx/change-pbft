import sys

# 从命令行参数读取配置
nodes_per_cluster = int(sys.argv[1])
server1 = sys.argv[2]
server2 = sys.argv[3]
server3 = sys.argv[4]
server4 = sys.argv[5]
server5 = sys.argv[6]

base_port = 2222  # 基础端口号

# 为了方便追踪每个服务器的端口号分配，我们使用一个变量来记录下一个可用的端口号
next_port = base_port

# 打开文件以写入节点信息
with open('nodetable.txt', 'w') as file:
    # 处理每个服务器
    for index, server in enumerate([server1, server2, server3, server4, server5]):
        # 对每个服务器生成指定数量的节点
        for i in range(nodes_per_cluster):
            node_id = index * nodes_per_cluster + i  # 计算节点ID，确保不重复
            # 写入节点信息到文件
            file.write(f"N{node_id} {server}:{next_port}\n")
            next_port += 1  # 更新端口号为下一个

#print("Node table has been created successfully.")
