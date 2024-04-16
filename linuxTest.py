import subprocess
import time
import datetime
import sys

arg = sys.argv[1]

if arg == "N":

    cnt = 0
    for i in range(300):
        cnt += 1
        curl_command = f"""
            curl -X POST "http://127.0.0.1:1110/req" -H "Content-Type: application/json" -d '{{"clientID":"ahnhwi-{i}","operation":"SendMes1 - {i}","timestamp":{i}}}'
            """

        subprocess.Popen(['bash', '-c', curl_command])
