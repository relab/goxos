@ECHO OFF

go build

if %errorlevel% == 0 (
	goto buildSuccess
) ELSE (
	echo go build failed. Press any key to exit
	pause>nul
	exit
)

:buildSuccess
echo go build successful.


for /l %%i in (0,1,2) do (
	echo starting %%i...
	start /B kvsd.exe -v=3 -log_dir=./log/ -id %%i -all-cores -config-file conf/config.ini
)



echo.
echo Running. Press any key to stop.

pause>nul

taskkill /F /IM kvsd.exe

echo Processes stopped. Press any key to exit.

pause>nul

