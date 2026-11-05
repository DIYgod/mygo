$ErrorActionPreference = 'Continue'
# Helpers the checks share: screenshots, input, UI Automation.
Add-Type -AssemblyName System.Windows.Forms, System.Drawing
Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class W {
  [DllImport("user32.dll")] public static extern void keybd_event(byte vk, byte scan, uint flags, UIntPtr extra);
  [DllImport("user32.dll")] public static extern void mouse_event(uint flags, int dx, int dy, uint data, UIntPtr extra);
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
  [DllImport("user32.dll")] public static extern bool SetProcessDPIAware();
  [DllImport("user32.dll")] public static extern short GetAsyncKeyState(int vk);
  [StructLayout(LayoutKind.Sequential)] public struct CURSORINFO { public int cbSize; public int flags; public IntPtr hCursor; public int x; public int y; }
  [DllImport("user32.dll")] public static extern bool GetCursorInfo(ref CURSORINFO ci);
  [DllImport("user32.dll")] public static extern bool DrawIconEx(IntPtr hdc, int x, int y, IntPtr icon, int w, int h, int step, IntPtr brush, int flags);
  [DllImport("user32.dll")] public static extern int GetSystemMetrics(int i);
}
'@
[void][W]::SetProcessDPIAware()

function Shot($name, [switch]$cursor) {
  $b = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds
  $bmp = New-Object System.Drawing.Bitmap $b.Width, $b.Height
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.CopyFromScreen($b.Location, [System.Drawing.Point]::Empty, $b.Size)
  if ($cursor) {
    $ci = New-Object W+CURSORINFO; $ci.cbSize = [Runtime.InteropServices.Marshal]::SizeOf($ci)
    if ([W]::GetCursorInfo([ref]$ci)) { $h = $g.GetHdc(); [void][W]::DrawIconEx($h, $ci.x, $ci.y, $ci.hCursor, 0, 0, 0, [IntPtr]::Zero, 3); $g.ReleaseHdc($h) }
  }
  $g.Dispose()
  $bmp.Save("$env:OUT\$name.png", [System.Drawing.Imaging.ImageFormat]::Png); $bmp.Dispose()
  "shot $name"
}

# Crop: a screenshot of a rectangle, scaled up by $zoom.
function ShotRect($name, $x, $y, $w, $h, $zoom = 1) {
  $bmp = New-Object System.Drawing.Bitmap $w, $h
  $g = [System.Drawing.Graphics]::FromImage($bmp)
  $g.CopyFromScreen($x, $y, 0, 0, (New-Object System.Drawing.Size $w, $h)); $g.Dispose()
  if ($zoom -ne 1) {
    $z = New-Object System.Drawing.Bitmap ($w*$zoom), ($h*$zoom)
    $g = [System.Drawing.Graphics]::FromImage($z); $g.InterpolationMode = 'NearestNeighbor'; $g.PixelOffsetMode = 'Half'
    $g.DrawImage($bmp, 0, 0, $w*$zoom, $h*$zoom); $g.Dispose(); $bmp.Dispose(); $bmp = $z
  }
  $bmp.Save("$env:OUT\$name.png", [System.Drawing.Imaging.ImageFormat]::Png); $bmp.Dispose()
  "shot $name"
}

$VK = @{ ALT=0x12; CTRL=0x11; SHIFT=0x10; ESC=0x1B; F10=0x79; SPACE=0x20; TAB=0x09; ENTER=0x0D; DOWN=0x28; UP=0x26; LEFT=0x25; RIGHT=0x27; APPS=0x5D; LWIN=0x5B }
function KeyDown($vk) { [W]::keybd_event([byte]$vk, 0, 0, [UIntPtr]::Zero); Start-Sleep -Milliseconds 60 }
function KeyUp($vk) { [W]::keybd_event([byte]$vk, 0, 2, [UIntPtr]::Zero); Start-Sleep -Milliseconds 60 }
function Key($vk) { KeyDown $vk; KeyUp $vk }
function Click($x, $y, [switch]$right) {
  [void][W]::SetCursorPos($x, $y); Start-Sleep -Milliseconds 100
  if ($right) { [W]::mouse_event(8, 0, 0, 0, [UIntPtr]::Zero); [W]::mouse_event(16, 0, 0, 0, [UIntPtr]::Zero) }
  else { [W]::mouse_event(2, 0, 0, 0, [UIntPtr]::Zero); [W]::mouse_event(4, 0, 0, 0, [UIntPtr]::Zero) }
  Start-Sleep -Milliseconds 300
}

# Builds a check program, from .github/winvm/<dir>, inside the PR's module.
function BuildCheck($dir, $exe) {
  $dst = "cmd\winvm-$dir"
  New-Item -ItemType Directory -Force $dst | Out-Null
  Copy-Item "$env:VM\$dir\*" $dst -Recurse -Force
  go build -o $exe "./$dst"
  if ($LASTEXITCODE -ne 0) { throw "build $dir failed" }
}

# Starts an app and waits for its window to be up; returns the process.
function StartApp($exe, $arguments = @(), $wait = 4) {
  $p = if ($arguments.Count) { Start-Process $exe -ArgumentList $arguments -PassThru -RedirectStandardOutput "$env:OUT\$([IO.Path]::GetFileNameWithoutExtension($exe))-stdout.txt" -RedirectStandardError "$env:OUT\$([IO.Path]::GetFileNameWithoutExtension($exe))-stderr.txt" } else { Start-Process $exe -PassThru -RedirectStandardOutput "$env:OUT\$([IO.Path]::GetFileNameWithoutExtension($exe))-stdout.txt" -RedirectStandardError "$env:OUT\$([IO.Path]::GetFileNameWithoutExtension($exe))-stderr.txt" }
  for ($i = 0; $i -lt 40 -and $p.MainWindowHandle -eq 0; $i++) { Start-Sleep -Milliseconds 250; $p.Refresh() }
  Start-Sleep -Seconds $wait
  if ($p.MainWindowHandle -ne 0) { [void][W]::SetForegroundWindow($p.MainWindowHandle) }
  $p
}

# UI Automation.
Add-Type -AssemblyName UIAutomationClient, UIAutomationTypes
function UiaRoot($p) { [System.Windows.Automation.AutomationElement]::FromHandle($p.MainWindowHandle) }
function UiaDump($el, $depth = 0, $max = 12) {
  if ($null -eq $el -or $depth -gt $max) { return }
  $c = $el.Current
  $extra = @()
  foreach ($pat in $el.GetSupportedPatterns()) { $extra += $pat.ProgrammaticName -replace 'PatternIdentifiers.Pattern','' }
  $lt = $el.GetCurrentPropertyValue([System.Windows.Automation.AutomationElement]::LocalizedControlTypeProperty)
  "{0}{1} name='{2}' loc='{3}' aid='{4}' cls='{5}' focus={6} [{7}]" -f ('  ' * $depth), $c.ControlType.ProgrammaticName.Replace('ControlType.',''), $c.Name, $lt, $c.AutomationId, $c.ClassName, $c.HasKeyboardFocus, ($extra -join ',')
  $w = [System.Windows.Automation.TreeWalker]::RawViewWalker
  $ch = $w.GetFirstChild($el)
  while ($null -ne $ch) { UiaDump $ch ($depth + 1) $max; $ch = $w.GetNextSibling($ch) }
}

# The Windows 11 runner starts at a privacy settings screen of the
# out-of-box experience, above every window: close it.
function Prep {
  Get-Process | Where-Object { $_.MainWindowTitle } | ForEach-Object { "window: $($_.ProcessName) '$($_.MainWindowTitle)'" }
  foreach ($n in 'WWAHost', 'CloudExperienceHost', 'CloudExperienceHostBroker', 'UserOOBEBroker') { Get-Process $n -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue }
  Start-Sleep 2
  Get-Process | Where-Object { $_.MainWindowTitle } | ForEach-Object { "window after: $($_.ProcessName) '$($_.MainWindowTitle)'" }
  Shot prep | Out-Null
}
Prep
