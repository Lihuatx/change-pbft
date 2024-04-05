import subprocess
import time
import datetime

# 在新的PowerShell窗口中执行命令
print(datetime.datetime.now())
for i in range(1):
    # 动态构建带有当前循环i值的PowerShell命令
    ps_command = f"""
    $headers = @{{ "Content-Type" = "application/json" }}
    $body = '{{"clientID":"ahnhwi","operation":"GetMyName","timestamp":{i}}}'
    $response = Invoke-WebRequest -Uri "http://114.55.130.178:1110/req" -Method POST -Headers $headers -Body $body
    """

    subprocess.Popen(['powershell', '-Command', ps_command])

