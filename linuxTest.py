import subprocess
import sys

arg = sys.argv[1]

if arg == "0":

    subprocess.run(['tmux', 'kill-session', '-t', 'myClient'])
    subprocess.run(['tmux', 'new-session', '-d', '-s', 'myClient'])

    command = f"./app client N"
    subprocess.run(['tmux', 'new-window', '-t', f'myClient:{1}', '-n', "Client-1"])

    tmux_command = f"tmux send-keys -t myClient:{1} './app client N' C-m"

    subprocess.Popen(['bash', '-c', tmux_command])
