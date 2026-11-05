# Prints the screen rectangles of elements anywhere on the desktop whose
# name contains the text: rect|<name>|<type>|x|y|w|h
import sys
sys.path.insert(0, sys.argv[1])
from uialib import *
want = sys.argv[2]
cond = uia.CreateTrueCondition()
els = uia.GetRootElement().FindAll(7, cond)
for i in range(els.Length):
    el = els.GetElement(i)
    try:
        n = el.CurrentName or ''
    except Exception:
        continue
    if want in n:
        r = el.CurrentBoundingRectangle
        print('rect|%s|%s|%d|%d|%d|%d' % (n, CT.get(el.CurrentControlType, el.CurrentControlType), r.left, r.top, r.right - r.left, r.bottom - r.top))
