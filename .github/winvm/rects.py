# Prints the screen rectangles of the named elements: rect <name> x y w h
import sys
sys.path.insert(0, sys.argv[1])
from uialib import *
root, hwnd = window(sys.argv[2])
for el in all_elements(root):
    n = el.CurrentName
    if n:
        r = el.CurrentBoundingRectangle
        print('rect|%s|%d|%d|%d|%d' % (n, r.left, r.top, r.right - r.left, r.bottom - r.top))
