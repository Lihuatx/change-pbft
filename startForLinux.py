import subprocess
import time

# 开始执行命令，类似于Bash脚本中的 set -x
print("Starting commands...")

# 执行 go build 命令
print("Building Go application...")
subprocess.run(['go', 'build', '-o', 'app'])

# 等待一段时间以确保编译完成
print("Waiting for build to finish...")
time.sleep(1)

# 创建新的 Tmux 会话
subprocess.run(['tmux', 'new-session', '-d', '-s', 'mySession'])

# 定义要在新窗口中执行的命令及其参数
commands = [
    ('./app', 'N0'),
    ('./app', 'N1'),
    ('./app', 'N2'),
    ('./app', 'N3'),
]

# 为每个命令创建新的 Tmux 窗口，并在该窗口中执行命令
for index, (exe, arg1) in enumerate(commands):
    # 为每个进程创建新窗口，窗口名为 "app-Nx"
    window_name = f"app-N{index}"
    subprocess.run(['tmux', 'new-window', '-t', f'mySession:{index + 1}', '-n', window_name])

    # 构建在新窗口中执行的命令
    tmux_command = f"tmux send-keys -t mySession:{index + 1} '{exe} {arg1}' C-m"
    subprocess.run(['bash', '-c', tmux_command])

time.sleep(2)

# 定义要执行的 curl 命令来代替 PowerShell 命令
curl_commands = [
    "curl -X POST -H 'Content-Type: application/json' -d '{\"clientID\":\"ahnhwi\",\"operation\":\"SendMes1\",\"timestamp\":859381532}' http://localhost:1111/req",
    "curl -X POST -H 'Content-Type: application/json' -d '{\"clientID\":\"ahnhwi\",\"operation\":\"SendMes2\",\"timestamp\":859381532}' http://localhost:1116/req",
    "curl -X POST -H 'Content-Type: application/json' -d '{\"clientID\":\"ahnhwi\",\"operation\":\"GetMyName\",\"timestamp\":859381532}' http://localhost:1121/req"
]

# 在 Tmux 会话的第一个窗口中执行 curl 命令
for curl_command in curl_commands:
    tmux_command = f"tmux send-keys -t mySession:1 '{curl_command}' C-m"
    subprocess.run(['bash', '-c', tmux_command])


