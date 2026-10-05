@echo OFF
:: This batch file simply launches win_CLICK_ME_TO_START.ps1 (kept ASCII-only on purpose:
:: cmd.exe decodes .bat files with the active code page, so non-ASCII here would break on some systems).
pushd %~dp0
set starter_script="%~dp0\win_CLICK_ME_TO_START.ps1"
powershell -noprofile -nologo -executionpolicy bypass -File %starter_script%
popd
