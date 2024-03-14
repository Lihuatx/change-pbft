import subprocess
import time

# 在新的PowerShell窗口中执行命令
for i in range(20):
    # 动态构建带有当前循环i值的PowerShell命令
    ps_command = f"""
    $headers = @{{ "Content-Type" = "application/json" }}
    $body = '{{"clientID":"ahnhwi","operation":"GetMyName","timestamp":{i}}}'
    $response = Invoke-WebRequest -Uri "http://localhost:1111/req" -Method POST -Headers $headers -Body $body
    """

    subprocess.Popen(['powershell', '-Command', ps_command])
    time.sleep(0.1)
