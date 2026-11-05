# Synthetic pen and touch input (InjectSyntheticPointerInput), as a
# tablet's: what WM_POINTER and the mouse messages made from it carry.
Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class Ptr {
  [StructLayout(LayoutKind.Sequential)] public struct PT { public int x, y; }
  [StructLayout(LayoutKind.Sequential)] public struct RC { public int l, t, r, b; }
  [StructLayout(LayoutKind.Sequential)] public struct INFO {
    public int type; public uint id; public uint frame; public int flags;
    public IntPtr source; public IntPtr hwnd; public PT px; public PT him; public PT pxRaw; public PT himRaw;
    public uint time; public uint history; public int inputData; public uint keyStates; public ulong perf; public int buttonChange;
  }
  [StructLayout(LayoutKind.Sequential)] public struct PEN { public INFO info; public uint penFlags; public uint penMask; public uint pressure; public uint rotation; public int tiltX; public int tiltY; }
  [StructLayout(LayoutKind.Sequential)] public struct TOUCH { public INFO info; public uint touchFlags; public uint touchMask; public RC contact; public RC contactRaw; public uint orientation; public uint pressure; }
  [StructLayout(LayoutKind.Explicit)] public struct TYPEINFO { [FieldOffset(0)] public int type; [FieldOffset(8)] public PEN pen; [FieldOffset(8)] public TOUCH touch; }
  [DllImport("user32.dll", SetLastError=true)] public static extern IntPtr CreateSyntheticPointerDevice(int type, uint max, int mode);
  [DllImport("user32.dll", SetLastError=true)] public static extern bool InjectSyntheticPointerInput(IntPtr dev, TYPEINFO[] info, uint n);
  static IntPtr pen, touch;
  public const int INRANGE = 0x2, INCONTACT = 0x4, PRIMARY = 0x2000, DOWN = 0x10000, UPDATE = 0x20000, UP = 0x40000;
  public static string Pen(int x, int y, int flags, uint pressure) {
    if (pen == IntPtr.Zero) pen = CreateSyntheticPointerDevice(3, 1, 1);
    if (pen == IntPtr.Zero) return "no pen device: " + Marshal.GetLastWin32Error();
    var t = new TYPEINFO(); t.type = 3;
    t.pen.info.type = 3; t.pen.info.flags = flags; t.pen.info.px.x = x; t.pen.info.px.y = y;
    t.pen.penMask = 1; t.pen.pressure = pressure;
    bool ok = InjectSyntheticPointerInput(pen, new[] { t }, 1);
    return ok ? "ok" : "inject failed: " + Marshal.GetLastWin32Error();
  }
  public static string Touch(int x, int y, int flags) {
    if (touch == IntPtr.Zero) touch = CreateSyntheticPointerDevice(2, 1, 1);
    if (touch == IntPtr.Zero) return "no touch device: " + Marshal.GetLastWin32Error();
    var t = new TYPEINFO(); t.type = 2;
    t.touch.info.type = 2; t.touch.info.flags = flags | PRIMARY; t.touch.info.px.x = x; t.touch.info.px.y = y;
    t.touch.touchMask = 1; t.touch.contact.l = x - 2; t.touch.contact.t = y - 2; t.touch.contact.r = x + 2; t.touch.contact.b = y + 2;
    bool ok = InjectSyntheticPointerInput(touch, new[] { t }, 1);
    return ok ? "ok" : "inject failed: " + Marshal.GetLastWin32Error();
  }
}
'@
function PenStroke($x, $y) {
  "pen down: " + [Ptr]::Pen($x, $y, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::DOWN, 256); Start-Sleep -Milliseconds 50
  foreach ($i in 1..6) { "pen move: " + [Ptr]::Pen($x + 15 * $i, $y + 3 * $i, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::UPDATE, 256 + 100 * $i); Start-Sleep -Milliseconds 30 }
  # Harder, without moving.
  "pen press harder: " + [Ptr]::Pen($x + 90, $y + 18, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::UPDATE, 1000); Start-Sleep -Milliseconds 50
  "pen up: " + [Ptr]::Pen($x + 90, $y + 18, [Ptr]::INRANGE -bor [Ptr]::UP, 0); Start-Sleep -Milliseconds 50
  "pen hover: " + [Ptr]::Pen($x + 100, $y + 20, [Ptr]::INRANGE -bor [Ptr]::UPDATE, 0); Start-Sleep -Milliseconds 50
  "pen away: " + [Ptr]::Pen($x + 100, $y + 20, [Ptr]::UPDATE, 0); Start-Sleep -Milliseconds 200
}
function PenTap($x, $y) {
  "pen tap: " + [Ptr]::Pen($x, $y, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::DOWN, 512) + " " + $(Start-Sleep -Milliseconds 60) + [Ptr]::Pen($x, $y, [Ptr]::INRANGE -bor [Ptr]::UP, 0)
  [void][Ptr]::Pen($x, $y, [Ptr]::UPDATE, 0); Start-Sleep -Milliseconds 300
}
function TouchDrag($x, $y) {
  "touch down: " + [Ptr]::Touch($x, $y, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::DOWN); Start-Sleep -Milliseconds 50
  foreach ($i in 1..6) { "touch move: " + [Ptr]::Touch($x + 15 * $i, $y, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::UPDATE); Start-Sleep -Milliseconds 30 }
  "touch up: " + [Ptr]::Touch($x + 90, $y, [Ptr]::UP); Start-Sleep -Milliseconds 300
}
function TouchTap($x, $y) {
  "touch tap: " + [Ptr]::Touch($x, $y, [Ptr]::INRANGE -bor [Ptr]::INCONTACT -bor [Ptr]::DOWN) + " " + $(Start-Sleep -Milliseconds 60) + [Ptr]::Touch($x, $y, [Ptr]::UP); Start-Sleep -Milliseconds 300
}
