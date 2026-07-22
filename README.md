# CLOUD_CLI

---

## Требования

Для локальной сборки необходимы:

- Go версии, указанной в `go.mod`;
- Git;
- Make — для команд из `Makefile`;
- Docker — только для сборки контейнера.

Проверить установленную версию Go:

```bash
go version
```
---
## Клонирование репозитория

```bash
git clone https://github.com/shuklarituparn/cloud_cli.git
```
---
##  Сборка

Сборка через Makefile
```bash
make build
```
После выполнения команды бинарный файл будет создан здесь: `bin/cloud_cli`

--- 

## Запуск:

```bash
./bin/cloud_cli help
```

## Создание профиля

```bash
./bin/cloud_cli profile create \
  --name=test \
  --user=example \
  --project=new-project
```
После выполнения команды в текущей директории появится файл:   `test.yaml`

Его содержимое:

`
user: "example"
project: "new-project"
`
## Получение профиля

```bash
./bin/cloud_cli profile get --name=test
```

Вывод в YAML:

`
user: "example"
project: "new-project"
`

Вывод в JSON:

```bash
./bin/cloud_cli profile get --name=test --output=json
```

Пример результата:

`
{
"name": "test",
"user": "example",
"project": "new-project"
}
`

Вывод списка профилей
```bash
./bin/cloud_cli profile list

```

Вывод в JSON:
```bash
./bin/cloud_cli profile list --output=json
```

Удаление профиля
```bash
./bin/cloud_cli profile delete --name=test
```

После удаления файл test.yaml исчезнет из текущей директории.

Справка

```bash
./bin/cloud_cli help
```

Также справка выводится при запуске без аргументов:

```bash
./bin/cloud_cli
```
