Для компиляции golang файлов из protocol buffer спецификации использовать следующие команды:

```bash

cd tinkoff/proto

git clone --depth 1 --branch v1.0.7 https://github.com/Tinkoff/investAPI.git

mkdir 1.0.7

protoc \
  --proto_path=./investAPI/src/docs/contracts \
  --go_out=./1.0.7 \
  --go-grpc_out=./1.0.7 \
  ./investAPI/src/docs/contracts/*


rm -r -f investAPI

```