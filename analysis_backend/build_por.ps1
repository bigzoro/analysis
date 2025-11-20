# cmd/por 编译脚本（禁用 CGO）
# 使用方法: .\build_por.ps1

$env:CGO_ENABLED=0
go build -o por.exe ./cmd/por

if ($LASTEXITCODE -eq 0) {
    Write-Host "编译成功: por.exe" -ForegroundColor Green
} else {
    Write-Host "编译失败" -ForegroundColor Red
    exit 1
}

