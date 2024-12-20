import subprocess
import time

node = 56
# 定义要在新终端中执行的命令及其参数
# 使用列表推导式生成命令列表
commands = [('app.exe', f'N{i}') for i in range(node)]

# 遍历命令和参数，然后在新的命令提示符窗口中执行
for exe, arg1 in commands:
    # 如果 app.exe 路径中包含空格，确保使用引号括起来
    command = f'start cmd /k "{exe}" {arg1}'
    subprocess.Popen(command, shell=True)

time.sleep(6)

# 定义第五个终端要执行的PowerShell命令
ps_command = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"GetMyName","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1110/req" -Method POST -Headers $headers -Body $body
"""
# 定义第五个终端要执行的PowerShell命令
start_command = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"start","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1110/req" -Method POST -Headers $headers -Body $body
"""

print("Start testing")

# for i in range(30):
#     # 动态构建带有当前循环i值的PowerShell命令
#     ps_command = f"""
#     $headers = @{{ "Content-Type" = "application/json" }}
#     $body = '{{"clientID":"ahnhwi","operation":"GetMyName","timestamp":{i}}}'
#     $response = Invoke-WebRequest -Uri "http://localhost:1111/req" -Method POST -Headers $headers -Body $body
#     """
#
#     subprocess.Popen(['powershell', '-Command', ps_command])
# # 在新的PowerShell窗口中执行第五个命令
# # subprocess.Popen(['powershell', '-Command', ps_command])
# time.sleep(5)
end_time = time.time() + 5
begin = False
cnt = 0
while cnt < 1:
    cnt+=1
    ps_command = f"""
    $headers = @{{ "Content-Type" = "application/json" }}
    $body = '{{"clientID":"ahnhwi","operation":"GetMyName","timestamp":{cnt}}}'
    $response = Invoke-WebRequest -Uri "http://localhost:1110/req" -Method POST -Headers $headers -Body $body
    """
    if cnt == 1:
        break
    if begin == False :
        subprocess.Popen(['powershell', '-Command', start_command])
        begin = True
    else:
        subprocess.Popen(['powershell', '-Command', ps_command])
print(cnt)
# 定义第五个终端要执行的PowerShell命令
end_command = """
$headers = @{ "Content-Type" = "application/json" }
$body = '{"clientID":"ahnhwi","operation":"end","timestamp":859381532}'
$response = Invoke-WebRequest -Uri "http://localhost:1110/req" -Method POST -Headers $headers -Body $body
"""

subprocess.Popen(['powershell', '-Command', end_command])
