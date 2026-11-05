"""UI Automation client helpers for the checks, over comtypes."""
import ctypes, sys, time
import comtypes, comtypes.client

UIA = comtypes.client.GetModule('UIAutomationCore.dll')
uia = comtypes.client.CreateObject(UIA.CUIAutomation, interface=UIA.IUIAutomation)

P = dict(ControlType=30003, LocalizedControlType=30004, Name=30005, HasKeyboardFocus=30008,
         AutomationId=30011, ClassName=30012, ItemStatus=30026, Orientation=30023, IsOffscreen=30022,
         LiveSetting=30135, AriaRole=30101, AriaProperties=30102, ControllerFor=30104,
         FullDescription=30159, LandmarkType=30157, LocalizedLandmarkType=30158,
         PositionInSet=30152, SizeOfSet=30153, Level=30154,
         SelectionItemIsSelected=30079, ExpandCollapseState=30070, ToggleState=30086,
         ValueValue=30045, HeadingLevel=30173)
PAT = dict(Invoke=10000, Selection=10001, Value=10002, RangeValue=10003, Scroll=10004,
           ExpandCollapse=10005, Grid=10006, GridItem=10007, SelectionItem=10010, Table=10012,
           TableItem=10013, Toggle=10015, Text=10014, LegacyIAccessible=10018)
CT = {50000+i: n for i, n in enumerate(
    'Button Calendar CheckBox ComboBox Edit Hyperlink Image ListItem List Menu MenuBar MenuItem '
    'ProgressBar RadioButton ScrollBar Slider Spinner StatusBar Tab TabItem Text ToolBar ToolTip '
    'Tree TreeItem Custom Group Thumb DataGrid DataItem Document SplitButton Window Pane Header '
    'HeaderItem Table TitleBar Separator SemanticZoom AppBar'.split())}

def prop(el, name):
    v = el.GetCurrentPropertyValue(P[name])
    return v

def patterns(el):
    out = []
    for n, i in PAT.items():
        try:
            if el.GetCurrentPattern(i):
                out.append(n)
        except Exception:
            pass
    return out

def describe(el):
    d = {}
    for n in ['Name', 'LocalizedControlType', 'AriaRole', 'AriaProperties', 'LandmarkType', 'LocalizedLandmarkType',
              'LiveSetting', 'Orientation', 'ItemStatus', 'PositionInSet', 'SizeOfSet', 'Level', 'HasKeyboardFocus',
              'SelectionItemIsSelected', 'ExpandCollapseState', 'IsOffscreen']:
        try:
            v = prop(el, n)
        except Exception as e:
            v = 'ERR'
        if v not in (None, '', 0, False) or n == 'Name':
            d[n] = v
    try:
        cf = el.GetCurrentPropertyValue(P['ControllerFor'])
        if cf:
            arr = cf.QueryInterface(UIA.IUIAutomationElementArray) if hasattr(cf, 'QueryInterface') else None
            if arr:
                d['ControllerFor'] = [arr.GetElement(i).CurrentName for i in range(arr.Length)]
    except Exception as e:
        pass
    ct = CT.get(el.CurrentControlType, el.CurrentControlType)
    return ct, d, patterns(el)

def line(el):
    ct, d, pats = describe(el)
    return '%s %s [%s]' % (ct, ' '.join('%s=%r' % kv for kv in d.items()), ','.join(pats))

def dump(el, depth=0, maxdepth=30, out=print):
    if el is None or depth > maxdepth:
        return
    out('  ' * depth + line(el))
    w = uia.RawViewWalker
    ch = w.GetFirstChildElement(el)
    while ch:
        dump(ch, depth + 1, maxdepth, out)
        ch = w.GetNextSiblingElement(ch)

def all_elements(root):
    res = []
    def walk(el):
        res.append(el)
        w = uia.RawViewWalker
        ch = w.GetFirstChildElement(el)
        while ch:
            walk(ch)
            ch = w.GetNextSiblingElement(ch)
    walk(root)
    return res

def find(root, name=None, ct=None, timeout=5):
    end = time.time() + timeout
    while True:
        for el in all_elements(root):
            if (name is None or el.CurrentName == name) and (ct is None or CT.get(el.CurrentControlType) == ct):
                return el
        if time.time() > end:
            return None
        time.sleep(0.25)

def window(title, timeout=20):
    user32 = ctypes.windll.user32
    end = time.time() + timeout
    while time.time() < end:
        h = user32.FindWindowW(None, title)
        if h:
            return uia.ElementFromHandle(h), h
        time.sleep(0.25)
    raise SystemExit('no window %r' % title)

def focused():
    return uia.GetFocusedElement()

def invoke(el):
    el.GetCurrentPattern(PAT['Invoke']).QueryInterface(UIA.IUIAutomationInvokePattern).Invoke()

def select(el):
    el.GetCurrentPattern(PAT['SelectionItem']).QueryInterface(UIA.IUIAutomationSelectionItemPattern).Select()

def selection(el):
    p = el.GetCurrentPattern(PAT['Selection']).QueryInterface(UIA.IUIAutomationSelectionPattern)
    arr = p.GetCurrentSelection()
    return [arr.GetElement(i).CurrentName for i in range(arr.Length)]

def pump(seconds):
    try:
        comtypes.client.PumpEvents(seconds)
    except Exception:
        time.sleep(seconds)

class EventLog(comtypes.COMObject):
    """Logs automation events (live regions, notifications) and focus changes."""
    _com_interfaces_ = [UIA.IUIAutomationEventHandler, UIA.IUIAutomationFocusChangedEventHandler]
    def __init__(self):
        super().__init__()
        self.events = []
    def IUIAutomationEventHandler_HandleAutomationEvent(self, sender, eventId):
        try:
            name = sender.CurrentName
            live = sender.GetCurrentPropertyValue(P['LiveSetting'])
        except Exception:
            name, live = '?', '?'
        self.events.append((time.time(), eventId, name, live))
        print('  event %d from %r (LiveSetting %s)' % (eventId, name, live), flush=True)
        return 0
    def IUIAutomationFocusChangedEventHandler_HandleFocusChangedEvent(self, sender):
        try:
            print('  focus -> ' + line(sender), flush=True)
        except Exception as e:
            print('  focus -> ?', e, flush=True)
        return 0

def listen(root, events=(20024, 20035, 20000)):
    h = EventLog()
    for e in events:
        uia.AddAutomationEventHandler(e, root, 7, None, h)
    uia.AddFocusChangedEventHandler(None, h)
    return h
