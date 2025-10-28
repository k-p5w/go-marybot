# 元フォルダ
$srcFolder = "E:\_home\GitHub\go-marybot\output"

# コピー先フォルダ
$destFolder = "E:\_home\GitHub\go-marybot\markdown_output"

# コピー先フォルダが無ければ作成
if (-not (Test-Path $destFolder)) {
    New-Item -ItemType Directory -Path $destFolder | Out-Null
}

# フォルダ内のCSVをすべて処理
Get-ChildItem -Path $srcFolder -Filter *.csv | ForEach-Object {
    $csvPath = $_.FullName
    $mdPath = Join-Path $destFolder ($_.BaseName + ".md")

    # CSVを読み込み
    $csv = Import-Csv $csvPath

    # Markdown表形式に変換
    $headers = ($csv | Get-Member -MemberType NoteProperty | Select-Object -ExpandProperty Name)
    $headerLine = "| " + ($headers -join " | ") + " |"
    $separatorLine = "| " + (($headers | ForEach-Object { "---" }) -join " | ") + " |"

    $rows = @()
    foreach ($row in $csv) {
        $line = "| " + (($headers | ForEach-Object { $row.$_ }) -join " | ") + " |"
        $rows += $line
    }

    # Markdownファイルとして保存
    $content = @($headerLine, $separatorLine) + $rows
    Set-Content -Path $mdPath -Value $content -Encoding UTF8

    Write-Host "変換完了: $mdPath"
}
