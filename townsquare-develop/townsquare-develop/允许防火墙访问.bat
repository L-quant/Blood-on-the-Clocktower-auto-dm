@echo off
echo 正在添加防火墙规则，允许端口8081访问...
echo.

netsh advfirewall firewall add rule name="Vue Dev Server 8081" dir=in action=allow protocol=TCP localport=8081

echo.
echo 防火墙规则已添加！
echo 现在可以在手机上访问：
echo   http://10.110.79.250:8081/
echo   或 http://172.22.224.1:8081/
echo.
pause
