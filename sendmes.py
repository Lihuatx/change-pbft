import subprocess
import time
import datetime

# 在新的PowerShell窗口中执行命令
print(datetime.datetime.now())
for i in range(5):
    # 动态构建带有当前循环i值的PowerShell命令
#     ps_command = f"""
#         $headers = @{{ "Content-Type" = "application/json" }}
#         $body = '{{"clientID":"ahnhwi","operation":"SendMes1 - {i}","timestamp":{i}}}'
#         $response = Invoke-WebRequest -Uri "47.107.59.211:1110/req" -Method POST -Headers $headers -Body $body
#         """
    ps_command = f"""
            $headers = @{{ "Content-Type" = "application/json" }}
            $body = '{{"clientID":"ahnhwi","operation":"SendMes1 - {i}","timestamp":{i}}}'
            $response = Invoke-WebRequest -Uri "127.0.0.1:1110/req" -Method POST -Headers $headers -Body $body
            """
    subprocess.Popen(['powershell', '-Command', ps_command])
    #time.sleep(0.05)

