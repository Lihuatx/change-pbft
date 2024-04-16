import subprocess
import time
import datetime

cnt = 0
for i in range(120):
    cnt += 1
    curl_command = f"""
        curl -X POST "http://47.107.59.211:1110/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi","operation":"SendMes1 - {i}","timestamp":{i}}}'
        """

    subprocess.Popen(['bash', '-c', curl_command])

    if cnt % 20 == 0:
        # time.sleep(1)  # 如果需要，取消注释此行以使用 sleep
        pass
