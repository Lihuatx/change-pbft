import subprocess
import sys
import time

command_template = 'app.exe'
groups = ['N']

if len(sys.argv) < 4:
    print("Usage: python script.py <start_node_id> <nodes_per_group> <z>")
    sys.exit(1)

start_node_id = int(sys.argv[1])
nodes_per_group = int(sys.argv[2])
z = int(sys.argv[3])

# 从 nodetable.txt 读取节点信息
commands = []
with open('nodetable.txt', 'r') as file:
    for line in file:
        parts = line.strip().split()  # 分割行为节点 ID 和 IP地址:端口
        if len(parts) == 2:
            node_id, ip_port = parts
            ip_address, _ = ip_port.split(':')  # 分割 IP地址:端口 为 IP 地址和端口号
            if ip_address == '0.0.0.0':
                # 当 IP 地址为 0.0.0.0 时，将节点加入命令列表
                commands.append((command_template, node_id))

def run_commands():
    print("Starting commands...")

    for index, (exe, arg1) in enumerate(commands):
        # 使用 'start' 命令在新窗口中运行程序
        cmd_command = f'start cmd /c {exe} {arg1} {nodes_per_group} {z}'
        subprocess.run(cmd_command, shell=True)




if __name__ == "__main__":
    run_commands()
    cmd_command = f'start cmd /c {command_template} "client"'
    subprocess.run(cmd_command, shell=True)
