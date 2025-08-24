# windows build lambda zip

Write-Host "Building Lambda function..." -ForegroundColor Green

# 設定環境變數
$env:GOOS="linux"
$env:GOARCH="amd64"
$env:CGO_ENABLED="0"

# 編譯程式
Write-Host "Compiling Go binary..." -ForegroundColor Yellow
go build -o bootstrap main.go

if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}

# 建立打包目錄
if (Test-Path "lambda-package") {
    Remove-Item -Recurse -Force "lambda-package"
}
New-Item -ItemType Directory -Name "lambda-package"

# 複製檔案
Write-Host "Copying files..." -ForegroundColor Yellow
Copy-Item "bootstrap" "lambda-package/"
Copy-Item -Recurse "settings" "lambda-package/"

# 打包成 ZIP
Write-Host "Creating ZIP package..." -ForegroundColor Yellow
Compress-Archive -Path "lambda-package/*" -DestinationPath "lambda-function.zip" -Force

# 清理
Remove-Item -Recurse -Force "lambda-package"
Remove-Item "bootstrap"

Write-Host "Lambda package created: lambda-function.zip" -ForegroundColor Green
