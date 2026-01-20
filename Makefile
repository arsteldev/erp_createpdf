generate:
	@echo "Генерация proto файлов"
	protoc --go_out=. --go-grpc_out=. -I=proto proto/createpdffile.proto
	@echo "Генерация завершенa"


build:
	@echo "Билд проекта"
	cd /home/baga/createPDF/cmd/app && go build -o ./createpdf-service
	@echo "Билд завершен"