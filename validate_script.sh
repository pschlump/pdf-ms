#!/bin/bash

cd ./gen/validate_script
rm validate_script
go build
mkdir -p ./out ./log
./validate_script "t_email_q" 
goimports -w out/x.go
cp out/x.go ../../ValidateTables.go


