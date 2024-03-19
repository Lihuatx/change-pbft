from flask import Flask, request, jsonify
import requests
import time

app = Flask(__name__)

def send_message_and_receive_response():
    # 目标服务器地址
    url = 'http://localhost:1111/req'

    # 请求头部，指定内容类型为JSON
    headers = {'Content-Type': 'application/json'}

    # 循环发送5条消息
    for i in range(1):
        # 请求体，包括客户端ID，操作和时间戳
        body = {
            "clientID": "ahnhwi",
            "operation": f"SendMsg - {i}",
            "timestamp": i
        }

        # 发送POST请求到目标服务器
        requests.post(url, json=body, headers=headers)

        # 打印服务器的回复
        # print(f"服务器回复（消息 {i}）:", response.text)
        # 等待一段时间再发送下一条消息（如果需要）
        # time.sleep(0.1)

@app.route('/reply', methods=['GET', 'POST'])
def get_reply():
    # 尝试获取JSON数据
    data = request.get_json()

    # 检查是否成功获取到数据
    if data is None:
        print("没有接收到JSON数据或数据格式不正确")
        return jsonify({"error": "没有接收到JSON数据或数据格式不正确"}), 400

    # 成功获取JSON数据
    print("接收到的数据:", data)
    return jsonify({"message": "数据已接收并发送"}), 200


@app.route('/send_messages')
def send_messages():
    send_message_and_receive_response()
    return "消息已发送"

if __name__ == '__main__':
    send_message_and_receive_response()
    app.run(port=5000,debug=True)
