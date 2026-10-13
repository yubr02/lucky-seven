$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
$localRoot = Join-Path $projectRoot '.local'
$pgBin = Join-Path $localRoot 'pgsql/bin'
$dataDir = Join-Path $localRoot 'pgdata'
$envFile = Join-Path $projectRoot '.env'
if (!(Test-Path "$pgBin/pg_ctl.exe")) { throw 'PostgreSQL binaries must be extracted into .local/pgsql first.' }
if (!(Test-Path $envFile)) {
    $bytes = New-Object byte[] 32
    $rng = [Security.Cryptography.RandomNumberGenerator]::Create()
    try { $rng.GetBytes($bytes) } finally { $rng.Dispose() }
    $password = [BitConverter]::ToString($bytes).Replace('-', '').ToLowerInvariant()
    $template = Get-Content (Join-Path $projectRoot '.env.example') -Raw
    $template = $template.Replace('POSTGRES_PASSWORD=', "POSTGRES_PASSWORD=$password").Replace('CHANGE_ME', $password)
    [IO.File]::WriteAllText($envFile, $template)
}
$settings = @{}
foreach ($line in Get-Content $envFile) {
    if ($line -match '^([A-Z_]+)=(.*)$') { $settings[$matches[1]] = $matches[2] }
}
if (!$settings.POSTGRES_PASSWORD) { throw 'Set POSTGRES_PASSWORD in .env.' }
$dbUser = $settings.POSTGRES_USER
$dbName = $settings.POSTGRES_DB
$dbPort = $settings.POSTGRES_PORT
if (!$dbUser -or !$dbName -or !$dbPort) { throw 'Set POSTGRES_USER, POSTGRES_DB and POSTGRES_PORT in .env.' }
if (!(Test-Path "$dataDir/PG_VERSION")) {
    $pwFile = Join-Path $localRoot 'init-password.tmp'
    try {
        [IO.File]::WriteAllText($pwFile, $settings.POSTGRES_PASSWORD)
        & "$pgBin/initdb.exe" -D $dataDir -U $dbUser -A scram-sha-256 --pwfile=$pwFile --encoding=UTF8 --locale=C
        if ($LASTEXITCODE -ne 0) { throw 'initdb failed.' }
    } finally { if (Test-Path $pwFile) { Remove-Item -LiteralPath $pwFile } }
}
& "$pgBin/pg_ctl.exe" -D $dataDir status
if ($LASTEXITCODE -ne 0) {
    $startArgs = @('-D', $dataDir, '-l', (Join-Path $localRoot 'postgres.log'), '-o', "`"-h 127.0.0.1 -p $dbPort`"", '-w', 'start')
    $process = Start-Process -FilePath "$pgBin/pg_ctl.exe" -ArgumentList $startArgs -WindowStyle Hidden -PassThru
    $process.WaitForExit()
    if ($process.ExitCode -ne 0) { throw 'PostgreSQL failed to start; inspect .local/postgres.log.' }
}
$oldPassword = $env:PGPASSWORD
try {
    $env:PGPASSWORD = $settings.POSTGRES_PASSWORD
    $exists = & "$pgBin/psql.exe" -h 127.0.0.1 -p $dbPort -U $dbUser -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$($dbName.Replace("'", "''"))'"
    if ($LASTEXITCODE -ne 0) { throw 'Database connection failed.' }
    if ($exists -ne '1') {
        & "$pgBin/createdb.exe" -h 127.0.0.1 -p $dbPort -U $dbUser $dbName
        if ($LASTEXITCODE -ne 0) { throw 'createdb failed.' }
    }
    $table = & "$pgBin/psql.exe" -h 127.0.0.1 -p $dbPort -U $dbUser -d $dbName -tAc "SELECT to_regclass('public.wallets')"
    if ($LASTEXITCODE -ne 0) { throw 'Schema check failed.' }
    if ([string]::IsNullOrWhiteSpace($table)) {
        & "$pgBin/psql.exe" -h 127.0.0.1 -p $dbPort -U $dbUser -d $dbName -v ON_ERROR_STOP=1 -f (Join-Path $projectRoot 'backend/migrations/001_game.sql')
        if ($LASTEXITCODE -ne 0) { throw 'Migration failed.' }
    }
} finally { $env:PGPASSWORD = $oldPassword }
Write-Output "PostgreSQL ready at 127.0.0.1:$dbPort ($dbName)."
