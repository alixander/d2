$ErrorActionPreference='Stop'
$stage=New-Item -ItemType Directory -Path "$env:ProgramData\D2PublishedAcceptance"
Copy-Item ./standard-user.ps1 $stage
$release='https://github.com/d2lang/d2/releases/download'
$assets=@(
    @('old.msi',"v0.9.0/d2-v0.9.0-windows-amd64.msi",'dbac13edaec26878d36c79c2bcd91d57927cae8878204e3b9816e6c0f32aae1b'),
    @('machine.msi',"v0.10.0/d2-v0.10.0-windows-amd64.msi",'5904a600801847f1e61c5bff6d0b9b5999786b9686de9516f2b5ede95287339f'),
    @('user.msi',"v0.10.0/d2-v0.10.0-windows-amd64-user.msi",'31b9690bede9a8a2408bfb6f37567540c476694cba6c5ae1056de5aa41f0efc8')
)
foreach ($asset in $assets) {
    $file = Join-Path $stage $asset[0]
    Invoke-WebRequest "$release/$($asset[1])" -OutFile $file
    if ((Get-FileHash $file -Algorithm SHA256).Hash -ine $asset[2]) { throw "Digest mismatch for $($asset[1])" }
    Write-Host "Published asset $($asset[1]) SHA-256 $($asset[2])"
}
$policy='HKLM:\SOFTWARE\Policies\Microsoft\Windows\Installer'
if (-not (Test-Path $policy)) { New-Item $policy -Force | Out-Null }
New-ItemProperty $policy -Name DisableMSI -PropertyType DWord -Value 0 -Force | Out-Null
$password=ConvertTo-SecureString ('T1!'+[guid]::NewGuid().ToString('N')) -AsPlainText -Force
$user=New-LocalUser -Name 'D2UpgradeTest' -Password $password
$group=Get-LocalGroup -SID 'S-1-5-32-545'
if ($user.SID -notin (Get-LocalGroupMember -Group $group).SID) { Add-LocalGroupMember -Group $group -Member $user }
$cred=[pscredential]::new("$env:COMPUTERNAME\$($user.Name)",$password)
function Check-Standard([string]$Action,[string]$Expected='unused') {
    Write-Host "== $Action ($Expected) =="
    $params="-NoProfile -NonInteractive -ExecutionPolicy RemoteSigned -File `"$stage\standard-user.ps1`" -Action `"$Action`" -Expected $Expected"
    $p=Start-Process powershell.exe -Credential $cred -LoadUserProfile -WorkingDirectory $stage -ArgumentList $params -Wait -PassThru
    $log=Join-Path $env:SystemDrive "Users\$($user.Name)\AppData\Local\d2-acceptance.txt"
    if (Test-Path $log) { Get-Content $log }
    if ($p.ExitCode -ne 0) { throw "Standard user step $Action returned $($p.ExitCode)" }
}
function Invoke-Machine([string]$Operation,[string]$File) {
    $path=Join-Path $stage $File
    $p=Start-Process msiexec.exe -ArgumentList @($Operation,('"{0}"' -f $path),'/qn','/norestart') -Wait -PassThru
    if ($p.ExitCode -ne 0) { throw "Machine $File $Operation returned $($p.ExitCode)" }
}
function Check-MachineProduct([string]$Expected) {
    $installer=New-Object -ComObject WindowsInstaller.Installer
    $rawProducts=$installer.GetType().InvokeMember('RelatedProducts','GetProperty',$null,$installer,@('{AC84FEE7-EB67-4F5D-A08D-ADEF69538690}'))
    $products=@(foreach ($code in $rawProducts) { $code })
    Write-Host "Machine MSI ProductCodes: $($products -join ',')"
    if ($Expected -eq 'none') {
        if ($products.Count -gt 0) { throw 'Machine uninstall left an installed product' }
    } elseif ($products.Count -ne 1 -or $products[0] -ine $Expected) { throw "Expected one machine product $Expected" }
}
Check-Standard 'none'
Invoke-Machine /i 'old.msi'
Check-MachineProduct '{91D48A67-5FAB-47CF-AB12-A2E1F1399B4C}'
Check-Standard 'machine' 'v0.9.0'
Check-Standard 'install'
Check-Standard 'machine,user' 'v0.9.0'
Invoke-Machine /i 'machine.msi'
Check-MachineProduct '{3832C347-0CC9-4EBB-AA76-B3E836AE3CBC}'
Check-Standard 'machine,user' 'v0.10.0'
Check-Standard 'remove'
Check-Standard 'machine' 'v0.10.0'
Check-Standard 'install'
Invoke-Machine /x 'machine.msi'
Check-MachineProduct 'none'
Check-Standard 'user' 'v0.10.0'
Check-Standard 'remove'
Check-Standard 'none'
