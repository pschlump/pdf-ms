
all:
	go build

linux:
	GOOS=linux go build -o pdf-micro-service.linux .

deploy_74:
	GOOS=linux go build -o pdf-micro-service.linux .
	echo ssh pschlump@192.154.97.74 "mkdir -p ./tools/pdf-micro-service"
	echo ssh pschlump@192.154.97.74 "mkdir -p ./tools/pdf-micro-service/www"
	echo ssh pschlump@192.154.97.74 "mkdir -p ./tools/pdf-micro-service/www/out"
	-echo ssh pschlump@192.154.97.74 "mv ./tools/pdf-micro-service/pdf-micro-service.linux ,aaaaa"
	echo scp *.linux pschlump@192.154.97.74:/home/pschlump/tools/pdf-micro-service
	scp prod-cfg.json pschlump@192.154.97.74:/home/pschlump/tools/pdf-micro-service
	scp run-prod.sh pschlump@192.154.97.74:/home/pschlump/tools/pdf-micro-service
	echo rsync -r -a -v -e "ssh -l pschlump"    ./www            			pschlump@192.154.97.74:/home/pschlump/tools/pdf-micro-service

