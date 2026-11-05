. "$env:VM\lib.ps1"
git clone -q --depth 1 https://github.com/microsoft/Selawik "$env:RUNNER_TEMP\selawik"
$ttf = Get-ChildItem "$env:RUNNER_TEMP\selawik" -Recurse -Filter selawksl.ttf | Select-Object -First 1
$dir = $ttf.DirectoryName; "fonts in $dir"; Get-ChildItem $dir -Filter *.ttf | ForEach-Object { $_.Name }
BuildCheck font "$env:RUNNER_TEMP\font.exe"
"== Semilight registered first"; & "$env:RUNNER_TEMP\font.exe" $dir semi
"== Light registered first"; & "$env:RUNNER_TEMP\font.exe" $dir light
"== window"
$p = StartApp "$env:RUNNER_TEMP\font.exe" @($dir, 'semi', 'window') 5
Shot font-window
Get-Content "$env:OUT\font-stdout.txt"
Stop-Process $p -Force
"== tests"
go test ./internal/text ./ui 2>&1 | Select-Object -Last 30
