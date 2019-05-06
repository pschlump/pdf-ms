#!/bin/bash

# Check that wkhtmltopdf works on system

#if [ "$(hostname)" == "prod0001" ] ; then
#	cp prod-cfg.js ./www/js/cfg.js
#else
#	cp dev-cfg.js ./www/js/cfg.js
#fi

xx=$( ps -ef | grep pdf-micro-service.linux | grep -v grep | awk '{print $2}' )
if [ "X$xx" == "X" ] ; then	
	:
else
	kill $xx
fi

(
while true ; do 
	if [ -f ./set-env.sh ] ; then
		. ./set-env.sh
	fi
	./pdf-micro-service.linux -cfg ./prod-cfg.json -hostport 192.154.97.75:9021 2>&1  >/tmp/pdf-micro-service.out 
	sleep 1 
done
) 2>&1 > /tmp/pdf-micro-service.2.out &
