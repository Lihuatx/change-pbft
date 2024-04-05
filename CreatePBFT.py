import subprocess
import time
import sys

command_template = './app.exe'
groups = ['N']
nodes_per_group = 28

if len(sys.argv) < 2:
    print("Usage: python script.py <start_node_id>")
    sys.exit(1)

# 从命令行读取 start_node_id 的值
start_node_id = int(sys.argv[1])

# 生成命令列表，从 ('app.exe', 'N<start_node_id>') 开始
commands = [(command_template, f'{group}{i}') for group in groups for i in range(start_node_id, start_node_id + nodes_per_group)]

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

        tmux_command = f"tmux send-keys -t myPBFT:{index + 1} '{exe} {arg1}' C-m"
        subprocess.run(['bash', '-c', tmux_command])

    time.sleep(2)

if __name__ == "__main__":
    run_commands()
