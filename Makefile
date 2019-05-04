
all:
	../bin/cmp-local.sh

linux:
	GOOS=linux go build -o pdf-micro-service.linux .

deploy_74:
	../bin/cmp-prod.sh pdf-micro-service.linux
	echo ssh pschlump@192.154.97.74 "mkdir -p ./tools/pdf-micro-service"
	echo ssh pschlump@192.154.97.74 "mkdir -p ./tools/pdf-micro-service/www"
	echo ssh pschlump@192.154.97.74 "mkdir -p ./tools/pdf-micro-service/www/out"
	-ssh pschlump@192.154.97.74 "mv ./tools/pdf-micro-service/pdf-micro-service.linux ,aaaaa"
	scp *.linux pschlump@192.154.97.74:/home/pschlump/tools/pdf-micro-service
	scp prod-cfg.json pschlump@192.154.97.74:/home/pschlump/tools/pdf-micro-service
	echo scp run-prod.sh pschlump@192.154.97.74:/home/pschlump/tools/pdf-micro-service
	echo rsync -r -a -v -e "ssh -l pschlump"    ./www            			pschlump@192.154.97.74:/home/pschlump/tools/pdf-micro-service
	echo "deploy-to-prod: " ${GIT_COMMIT} `date` >>build-log.txt 

# From: https://github.com/mileszs/wicked_pdf/issues/723
NoteInstallPatchedVersion:
	wget https://github.com/wkhtmltopdf/wkhtmltopdf/releases/download/0.12.3/wkhtmltox-0.12.3_linux-generic-amd64.tar.xz
	tar vxf wkhtmltox-0.12.3_linux-generic-amd64.tar.xz 
