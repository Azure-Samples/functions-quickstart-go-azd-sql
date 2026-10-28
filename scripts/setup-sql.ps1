$ErrorActionPreference = 'Stop'
$values = azd env get-values --output json | ConvertFrom-Json
if ($LASTEXITCODE -ne 0) { throw 'Could not load azd environment.' }
foreach ($key in @('SQL_SERVER_FQDN', 'SQL_DATABASE', 'SQL_IDENTITY_NAME', 'SQL_IDENTITY_CLIENT_ID')) {
    Set-Item "env:$key" $values.$key
}
go run ./cmd/setup
if ($LASTEXITCODE -ne 0) { throw 'SQL initialization failed.' }
