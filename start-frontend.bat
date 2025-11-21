@echo off
echo Starting Frontend...
set NEXT_PUBLIC_API_URL=http://localhost:8080

cd /d "%~dp0frontend"
echo Installing dependencies (if needed)...
call npm install
echo Starting development server...
call npm run dev
pause
