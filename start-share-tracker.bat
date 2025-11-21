@echo off
echo Starting Share Tracker Service...
set KAFKA_BROKERS=localhost:9092
set KAFKA_TOPIC=file-events
set KAFKA_GROUP_ID=share-tracker-group
set LOG_FILE_PATH=./SharedFiles/shared_files.json

cd /d "%~dp0services\share-tracker"
go run main.go
pause
