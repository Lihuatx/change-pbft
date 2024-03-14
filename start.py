import subprocess
import time

# 定义要在新终端中执行的命令及其参数
commands = [
    ('app.exe', 'N0'),
    ('app.exe', 'N1'),
    ('app.exe', 'N2'),
    ('app.exe', 'N3'),

]

# 遍历命令和参数，然后在新的命令提示符窗口中执行
for exe, arg1 in commands:
    # 如果 app.exe 路径中包含空格，确保使用引号括起来
    command = f'start cmd /k "{exe}" {arg1}'
    subprocess.Popen(command, shell=True)

time.sleep(2)

# 定义第五个终端要执行的PowerShell命令
ps_command = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"GetMyName","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1111/req" -Method POST -Headers $headers -Body $body
"""


# 在新的PowerShell窗口中执行第五个命令
subprocess.Popen(['powershell', '-Command', ps_command])
