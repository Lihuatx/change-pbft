import sys

arg = sys.argv[1]
nodes_per_cluster = int(arg)  # 节点总数
client = sys.argv[2]
server = sys.argv[3]
z = int(sys.argv[4])
nodes_per_cluster = z * nodes_per_cluster
base_port = 1110  # 基础端口号

# 计算前三分之一节点的数量（向上取整）
first_third = (nodes_per_cluster + 2) // 3  # 使用整数除法并向上取整

# 初始化 NodeTable
node_table = {}
for i in range(nodes_per_cluster):
    node_id = f"N{i}"
    if i < first_third:
        # 前三分之一的节点使用 client IP 地址
        address = f"{client}:{base_port + i}"
    else:
        # 后三分之二的节点使用 server IP 地址
        address = f"{server}:{base_port + i}"
    node_table[node_id] = address

# 将 NodeTable 保存到 nodetable.txt 文件中
with open('nodetable.txt', 'w') as file:
    for node_id, address in node_table.items():
        file.write(f"{node_id} {address}\n")
