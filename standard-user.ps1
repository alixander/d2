param([string]$Action,[string]$Expected)
$ErrorActionPreference='Stop'
$log = Join-Path $env:LOCALAPPDATA 'd2-acceptance.txt'
Start-Transcript -Path $log -Force
try {
    $me = [Security.Principal.WindowsPrincipal]::new([Security.Principal.WindowsIdentity]::GetCurrent())
    if ($me.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw "Expected a non-admin test user" }
    $userD2 = Join-Path $env:LOCALAPPDATA 'Programs\D2'
    $machineD2 = Join-Path $env:ProgramFiles 'D2'
    if ($Action -eq 'install' -or $Action -eq 'remove') {
        $operation = if ($Action -eq 'install') {'/i'} else {'/x'}
        $msi = Join-Path $PSScriptRoot 'user.msi'
        $p=Start-Process msiexec.exe -ArgumentList @($operation,('"{0}"' -f $msi),'/qn','/norestart') -Wait -PassThru
        if ($p.ExitCode -ne 0) { throw "user MSI $operation returned $($p.ExitCode)" }
    } else {
        Write-Host "Standard identity: $($me.Identity.Name); LocalApplicationData=$env:LOCALAPPDATA"
        $mp = [Environment]::GetEnvironmentVariable('Path','Machine')
        $up = [Environment]::GetEnvironmentVariable('Path','User')
        $pathPattern = '(?i)\\D2\\?$'
        Write-Host "User PATH D2 entries: $(($up -split ';' | Where-Object {$_ -match $pathPattern}) -join ';')"
        Write-Host "Machine PATH D2 entries: $(($mp -split ';' | Where-Object {$_ -match $pathPattern}) -join ';')"
        $nativeErr = $ErrorActionPreference
        $ErrorActionPreference='Continue'
        $resolved = @(where.exe d2 2>$null)
        $whereStatus = $LASTEXITCODE
        $ErrorActionPreference=$nativeErr
        $expectedPaths = @()
        foreach ($scope in $Action.Split(',')) {
            if ($scope -eq 'machine') { $expectedPaths+=(Join-Path $machineD2 'd2.exe') }
            if ($scope -eq 'user') { $expectedPaths+=(Join-Path $userD2 'd2.exe') }
        }
        $hasMachine = @($mp -split ';' | ForEach-Object {$_.TrimEnd('\')}) -contains $machineD2
        $hasUser = @($up -split ';' | ForEach-Object {$_.TrimEnd('\')}) -contains $userD2
        if ($hasMachine -ne ($Action.Split(',') -contains 'machine') -or $hasUser -ne ($Action.Split(',') -contains 'user')) { throw "Registry PATH membership differs for $Action" }
        Write-Host "where.exe d2 = $($resolved -join ' | ')"
        if (($resolved -join ';') -ine ($expectedPaths -join ';')) { throw "Expected where.exe order: $($expectedPaths -join ' | ')" }
        if ($expectedPaths.Count) {
            $version = (& d2 --version | Out-String).Trim()
            if ($whereStatus -ne 0 -or $LASTEXITCODE -ne 0 -or $version -cne $Expected) { throw "Expected $Expected for d2 on PATH; received $version" }
            Write-Host "PATH CLI version: $version"
            foreach ($exe in $resolved) { Write-Host "$exe reports $(& $exe --version)" }
        } elseif ($whereStatus -eq 0) { throw 'where.exe unexpectedly succeeded after complete removal' }
    }
} catch { Write-Host $_; exit 1 } finally { Stop-Transcript }
