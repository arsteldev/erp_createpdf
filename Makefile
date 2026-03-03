generate:
	@echo "Генерация proto файлов"
	protoc --go_out=proto --go-grpc_out=proto -I=proto proto/createpdffile.proto
	@echo "Генерация завершенa"


build:
	@echo "Билд проекта"
	cd /home/baga/createPDF/cmd/app && go build -o ./createpdf-service
	@echo "Билд завершен"

start:
	./createpdf-service --config=config/local.yaml

stop_prod:
	systemctl stop createpdf-service.service