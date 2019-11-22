#Requires -RunAsAdministrator
[CmdletBinding()]
Param(
    [Parameter(Mandatory=$true, ParameterSetName='Server')]
    [Switch] $Server,
    [Parameter(Mandatory=$true, ParameterSetName='Sensor')]
    [Switch] $Sensor
)

# Pull down the website content for the latest release
try { $webreq = Invoke-WebRequest -Uri "https://github.com/kushtaka/kushtakad/releases/latest" -UseBasicParsing } catch [System.Net.WebException] { 
    Write-Verbose "An exception was caught: $($_.Exception.Message)"
    $_.Exception.Response
}

# Parse content for the latest Windows version and get the link
[string]$url = "https://github.com/" + ($webreq.links.href | Select-String -Pattern "_windows_amd64.zip")

# Get file name and version number
$filename = $url.Split('/')[-1]
$latestVersion = ($url.Split('/')[8]).Trim('v')

# If kushtaka is installed and latest version is newer, stop the process and remove the version file; otherwise, exit the script
if (Test-Path -Path 'C:\Program Files\kushtaka\version') {
    if ($latestVersion -gt (Get-Content -Path 'C:\Program Files\kushtaka\version')) {
        Stop-Process -Name 'kushtakad' -Force
        Remove-Item -Path 'C:\Program Files\kushtaka\version' -Force
     } else { exit }
} 

# Create kushtaka folder if it doesn't exist, then download latest zip to AppData\Local\Temp
if (!(Test-Path -Path "C:\Program Files\kushtaka")) { New-Item -Path "C:\Program Files\kushtaka" -ItemType Directory -Force }
try { Invoke-WebRequest -Uri $url -OutFile "C:\Program Files\kushtaka\$filename" } catch [System.Net.WebException] { 
    Write-Verbose "An exception was caught: $($_.Exception.Message)"
    $_.Exception.Response
}

# Extract kusthakad.exe and clean up zip file
if (Test-Path -Path "C:\Program Files\kushtaka\$filename") { Expand-Archive "C:\Program Files\kushtaka\$filename" -DestinationPath "C:\Program Files\kushtaka" -Force } else { throw "Download failed" }
if (Test-Path -Path "C:\Program Files\kushtaka\kushtakad.exe") { Remove-Item -Path "C:\Program Files\kushtaka\$filename" } else { throw "Extraction failed" }

# Add version file (metadata not included in binary yet)
if (!(Test-Path 'C:\Program Files\kushtaka\version')) { Add-Content -Path 'C:\Program Files\kushtaka\version' -Value $latestVersion }

# Download latest script and use to update (uncomment for automatic updates, but be aware of potential supply chain attacks)
#Invoke-WebRequest -Uri "https://github.com/nathanmcnulty/kushtakad/blob/master/Update-Kushtaka.ps1" -OutFile "C:\Program Files\kushtaka\Update-Kushtaka.ps1"

# Add scheduled task if it does not exist
if (!(Get-ScheduledTask | Where-Object { $_.TaskName -eq 'kushtaka'})) { 
    # Set up task to start application at startup (will move to service in the future)
    if ($server) { $actions = (New-ScheduledTaskAction -Execute 'powershell.exe' -Argument '-ExecutionPolicy Bypass -File "C:\Program Files\kushtaka\Update-Kushtaka.ps1"'),(New-ScheduledTaskAction -Execute 'C:\Program Files\kushtaka\kushtakad.exe') }
    if ($sensor) { $actions = (New-ScheduledTaskAction -Execute 'powershell.exe' -Argument '-ExecutionPolicy Bypass -File "C:\Program Files\kushtaka\Update-Kushtaka.ps1"'),(New-ScheduledTaskAction -Execute 'C:\Program Files\kushtaka\kushtakad.exe' -Argument '-sensor') }
    $trigger = New-ScheduledTaskTrigger -AtStartup
    Register-ScheduledTask -Action $actions -Trigger $trigger -TaskName "kushtaka" -User "NT AUTHORITY\SYSTEM"
}

#Start kushtaka back up
if ($server) { Start-Process -FilePath 'C:\Program Files\kushtaka\kushtakad.exe' }
if ($sensor) { 
    if (Test-Path "C:\Program Files\kushtaka\sensor.json") { 
        Start-Process -FilePath 'C:\Program Files\kushtaka\kushtakad.exe' -ArgumentList '-sensor' 
    } else { 
        Write-Warning "Please copy your sensor.json file to C:\Program Files\kushtaka, then run the kushtaka scheduled task"
    }
}