#!/bin/bash

go build
mkdir -p ./out ./log
./validate_script "t_email_q" 
goimports -w out/x.go

