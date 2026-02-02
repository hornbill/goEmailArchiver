# Package Variables
$package = "goEmailArchiver"
$platforms = @( "windows/386", "windows/amd64")

# Code Sign Variables
$sharedPath = "D:\dev\build\_shared"
$signConfigPath = "D:\dev\build\_shared\supplier\certs\config.json"

$version = go run . -version
Write-Output "`n🔶🔶🔶🔶 Building Hornbill Email Archiver v$($version) 🔶🔶🔶🔶"

Write-Output "`n* Creating folder: release"
Remove-Item -r -force .\release -ErrorAction silent -WarningAction silent
mkdir release | Out-Null

Write-Output "* Creating folder: builds"
Remove-Item -r -force .\builds -ErrorAction silent -WarningAction silent
mkdir builds | Out-Null

$env:CGO_ENABLED = 1
foreach ($platform in $platforms) {
    $osPlatform = $platform -split "/" 
    $env:GOOS = $osPlatform[0]
    $env:GOARCH = $osPlatform[1]
    if ($osPlatform[0] -ne "windows") {
        $output = $package
    }
    else {
        $output = "$($package).exe"
    }

    Write-Output "`n🟢 Building for $($osPlatform[0])/$($osPlatform[1]) 🟢"
    $buildFolder = "builds\$($osPlatform[0])-$($osPlatform[1])"
    $built = "$buildFolder\$output"

    Write-Output "`n* Compiling code..."
    go build -trimpath -o $built

#    Write-Output "* Signing code..."
#    hsign -config $signConfigPath -shared_path $sharedPath -filename $built

#    Write-Output "`n* Validating code sign status..."
#    $codeSigned = Get-AuthenticodeSignature $built

#    if ($codeSigned.Status -eq "Valid") {
#        Write-Output "✅ $($codeSigned.Status) - $($codeSigned.StatusMessage)"
#    } else {
#        Write-Output "❌ $($codeSigned.Status) - $($codeSigned.StatusMessage)"
#    }


    Write-Output "`n* Copying Source Files..."
    Copy-Item -Path *.md -Destination $buildFolder
    Copy-Item -Path conf*.json -Destination $buildFolder

    Write-Output "* Creating build archive..."
    $releasePath = "release\$($package)-$($osPlatform[0])-$($osPlatform[1]).zip"
    Compress-Archive -Path "$($buildFolder)\*.*" -DestinationPath $releasePath -CompressionLevel Optimal

    Write-Output "* $($osPlatform[0]))/$($osPlatform[1]) PACKAGE BUILD COMPLETE"
}
Write-Output "`n* Performing cleanup..."
Remove-Item -r -force .\builds
Write-Output "`n🔷🔷🔷🔷 BUILD COMPLETE 🔷🔷🔷🔷`n"
