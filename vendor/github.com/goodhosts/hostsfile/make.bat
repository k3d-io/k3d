@echo off
SETLOCAL ENABLEDELAYEDEXPANSION
set LF=^

echo %1%
if "%1%"=="" (
    set cmd=ci
) else (
    set cmd=%1%
)

if "%cmd%" == "ci" (
  for /F %%i in ('goimports -l .') do (
    set "line=%%i"
    set goimports=%goimports%!line!!LF!
  )

  if not "!goimports!" == "" (
    goimports -d .
    goto :eof
  )

  golangci-lint run || goto :eof
  go test -v . || goto :eof

  goto :eof
)
