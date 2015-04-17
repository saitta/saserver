#! /usr/bin/env python3
# -*- coding: utf-8 -*-
import socket
import os
import sys
import time
import select

MYSOCKET="/tmp/TelldusClient"

getNDev = "20:tdGetNumberOfDevices"

print("Connecting...")
if os.path.exists(MYSOCKET):
    client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    client.connect(MYSOCKET)
    print("Ready.")
    print("Ctrl-C to quit.")
    print("Sending 'DONE' shuts down the server and quits.")
    client.setblocking(0)

    try:
        while True:
            try:
                
                # Send data
                message = getNDev
                ready = select.select([], [client], [], 1)
                if ready[1]:
                    print('sending "{0:s}"'.format(message))
                    client.sendall(bytearray(message,'utf8'))

                    amount_received = 0
                    ready = select.select([client], [], [], 1)
                    if ready[0]:
                        data = client.recv(4096)
                        amount_received += len(data)
                        print('received "{0:s}"'.format(data.decode('utf8')))
                time.sleep(3)
            except BrokenPipeError as bp:
                client.close()
                # time.sleep(.1)
                client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
                client.connect(MYSOCKET)
                client.setblocking(0)
                print("bp")
                #ready = select.select([client], [], [], 1)
                #if ready[0]:
                #    data = client.recv(4096)
                #    amount_received += len(data)
                #    print('received "{0:s}"'.format(data.decode('utf8')))

    except KeyboardInterrupt as k:
        print("Shutting down.")
    finally:
        print('closing socket')
        client.close()
else:
    print("Couldn't Connect!")
    print("Done")
