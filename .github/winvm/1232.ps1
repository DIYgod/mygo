. "$env:VM\lib.ps1"
$sd = "$env:RUNNER_TEMP\selawik"; New-Item -ItemType Directory -Force $sd | Out-Null
gh release list --repo microsoft/Selawik --limit 5
gh release download 1.01 --repo microsoft/Selawik -D $sd --pattern '*' --skip-existing
Get-ChildItem $sd -Filter *.zip | ForEach-Object { Expand-Archive $_.FullName -DestinationPath "$sd\$($_.BaseName)" -Force }
Get-ChildItem "$env:RUNNER_TEMP\selawik" -Recurse -Include *.ttf | ForEach-Object { $_.FullName }
$ttf = Get-ChildItem "$env:RUNNER_TEMP\selawik" -Recurse -Include selawksl.ttf | Select-Object -First 1
$dir = $ttf.DirectoryName; "fonts in $dir"
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
