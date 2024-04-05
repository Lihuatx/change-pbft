clusters = ['N', 'M', 'P']  # 集群的标识符
nodes_per_cluster = 26  # 每个集群的节点数
base_port = 1110  # 基础端口号

# 读取 network.txt 中的所有行作为 IP 地址列表
with open('network.txt', 'r') as network_file:
    ip_addresses = [line.strip() for line in network_file if line.strip()]

# 初始化 NodeTable
node_table = {}

# 对于每个 IP 地址，创建一组节点
for ip_index, ip_address in enumerate(ip_addresses):
    # 计算当前 IP 地址对应的集群标识符
    cluster = clusters[0]

    # 确保 node_table 中有对应的集群键
    if cluster not in node_table:
        node_table[cluster] = {}

    # 为当前 IP 创建节点
    for i in range(nodes_per_cluster):
        node_id = f"{cluster}{i + (ip_index // len(clusters)) * nodes_per_cluster}"
        address = f"{ip_address}:{base_port + i}"
        node_table[cluster][node_id] = address

# 将 NodeTable 保存到 nodetable.txt 文件中
with open('nodetable.txt', 'w') as file:
    for cluster, nodes in node_table.items():
        for node_id, address in nodes.items():
            file.write(f"{cluster} {node_id} {address}\n")
