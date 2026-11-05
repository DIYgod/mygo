import sys, time, ctypes
sys.path.insert(0, sys.argv[1])
from uialib import *
title = sys.argv[2]
root, hwnd = window(title)
ctypes.windll.user32.SetForegroundWindow(hwnd)
time.sleep(1)
print('== tree'); dump(root)
log = listen(root)
for i, b in enumerate(sys.argv[3:]):
    print('== invoke %s' % b)
    el = find(root, b, ct='Button')
    if el is None:
        print('  no button', b); continue
    invoke(el)
    pump(2.0)
print('== tree after'); dump(root)
print('events:', [(e[1], e[2], e[3]) for e in log.events])
