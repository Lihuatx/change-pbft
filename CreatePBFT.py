import subprocess
import time
import sys

command_template = './app.exe'
groups = ['N']
nodes_per_group = int(sys.argv[2])
z = int(sys.argv[3])

if len(sys.argv) < 2:
    print("Usage: python script.py <start_node_id>")
    sys.exit(1)

# 从命令行读取 start_node_id 的值
start_node_id = int(sys.argv[1])

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

    # 由于是exe文件，可能不需要这个步骤，除非app.exe需要从源代码构建
    print("Building Go application...")
    subprocess.run(['go', 'build', '-o', 'app.exe'])

    print("Waiting for build to finish...")
    time.sleep(1)

    subprocess.run(['tmux', 'new-session', '-d', '-s', 'myPBFT'])

    # 遍历命令列表
    for index, (exe, arg1) in enumerate(commands):
        window_name = f"app-{arg1}"
        subprocess.run(['tmux', 'new-window', '-t', f'myPBFT:{index + 1}', '-n', window_name])
        time.sleep(0.1)

        tmux_command = f"tmux send-keys -t myPBFT:{index + 1} '{exe} {arg1} {nodes_per_group} {z}' C-m"
        subprocess.run(['bash', '-c', tmux_command])

    time.sleep(1)

if __name__ == "__main__":
    run_commands()
