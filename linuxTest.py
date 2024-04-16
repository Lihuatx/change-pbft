import subprocess
import time
import datetime

cnt = 0
for i in range(3000):
    cnt += 1
    curl_command = f"""
        curl -X POST "http://127.0.0.1:1110/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi-{i}","operation":"SendMes1 - {i}","timestamp":{i}}}'
        """

    subprocess.Popen(['bash', '-c', curl_command])
