import sys, time, ctypes
sys.path.insert(0, sys.argv[1])
from uialib import *
out = sys.argv[2]
root, hwnd = window('Listbox check')
ctypes.windll.user32.SetForegroundWindow(hwnd)
time.sleep(1)
print('== closed'); dump(root)
log = listen(root)
sel = find(root, 'Plan')
print('select:', line(sel))
sel.SetFocus(); pump(0.5)
# Open with the keyboard, as a user would.
VK_SPACE, VK_DOWN, VK_UP, VK_ESC = 0x20, 0x28, 0x26, 0x1B
def key(vk):
    ctypes.windll.user32.keybd_event(vk, 0, 0, 0); time.sleep(0.05); ctypes.windll.user32.keybd_event(vk, 0, 2, 0); pump(0.6)
key(VK_SPACE)
print('== open'); dump(root)
lst = find(root, ct='List', timeout=3)
if lst is None:
    print('FAIL no List in the open popup')
else:
    print('list:', line(lst), 'selection:', selection(lst) if 'Selection' in patterns(lst) else 'NO SELECTION PATTERN')
print('focused:', line(focused()))
print('== arrow down twice')
key(VK_DOWN); print('focused:', line(focused()))
key(VK_DOWN); print('focused:', line(focused()))
if lst: 
    for el in all_elements(lst)[1:]:
        print('  ', line(el))
print('== SetFocus on Free (should only highlight)')
free = find(root, 'Free', ct='ListItem', timeout=2)
if free:
    free.SetFocus(); pump(0.8)
    print('focused:', line(focused()))
    for el in all_elements(lst)[1:]:
        print('  ', line(el))
    print('popup still open:', find(root, ct='List', timeout=1) is not None)
print('== SelectionItem.Select on Team')
team = find(root, 'Team', ct='ListItem', timeout=2)
if team:
    select(team); pump(1.0)
    print('popup open after Select:', find(root, ct='List', timeout=1) is not None)
    print('text:', [e.CurrentName for e in all_elements(root) if e.CurrentName.startswith('Plan:')])
    print('select now:', line(find(root, 'Plan')))
print('focused:', line(focused()))
