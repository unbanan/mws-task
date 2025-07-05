# Задание

Анализ зависимостей репозитория
Задача: Для указанного репозитория вывести данные о модуле и список зависимостей, которые можно обновить

Результат: CLI которая на вход получает адрес гит репо, на выходе имя модуля, версия golang и список зависимостей для обновления

Критерии оценки:
Работает на репо без зависимостей
Работает на репо вне github

# Оценка условия

Как я понял от меня требуется сделать программу, которой на вход подается ссылка на git репозиторий и я должен сделать одно из 2 действий.

+ Если репозиторий **валидный** то я должен найти в нем go.mod файлик и вывести имя модуля, версию го, и список зависимостей для обновления. Также версию зависимости нужно выводить **самую новую**. В моем понимании репозиторий валидный если его возможно сбилдить *go build* и в нем находится 1 go.mod файл, также возможны конфликты зависимостей т. к. мы показываем самую новую версию.

+ Если репозиторий **невалидный** или появились какие-то ошибки по ходу пробросить их/сделать логи.

# Варианты решения

## Как достать репозиторий?
1. Первая идея была клонировать репозиторий целиком, искать в нем go.mod и смотреть на версии.
2. Дальше пришла идея использовать Github API, но по условию у нас дан просто гит репозиторий. С другой стороны используя Github API когда репозиторий находится на Github может помочь сэкономить память.
3. После мне пришла идея, что нам не обязательно клонировать репозиторий целиком. Мы можем просто склонировать только последнюю версию кода без всей истории изменений через ```git clone --depth 1```.
4. Использовать [готовые решения](https://github.com/go-git/go-git)(рассмотрю дальше)

Я выбрал 3 вариант здесь, так как он показался мне самым эффективным из тех, что я смог придумать.

## Как проверять версии на наличие обновлений
У меня было 2 варианта как решить эту часть задачи

1. Использовать go list(для получения информации о нашем репозитории) + ls-remote(для получения информации о репозиториях с нужными библиотеками и их версиями. В тегах информация о версии.) и сравнивать эту информацию.
2. Воспольоваться функционалом ```go list -u -m all```, команда по сути предоставляет нам список зависимостей модуля и информацию об обновлениях.
3. Опять же можно использовать [готовые решения](https://pkg.go.dev/golang.org/x/mod).

Тут я остановился на 2 варианте в силу того что я предполагаю что выводить информацию нужно только если репозиторий **валидный**.

# Реализация №1
Тепеь пройдемся по деталям реализации и как это вообще работает.

Для начала нам нужно получить от пользователя ссылку. Я сделал это с помощью ```flag``` следующим образом: 

```go
    var gitURL = flag.String("link", "", "GIT Repository URL")
    flag.Parse()
    if *gitURL == "" {
        log.Fatal("Git repository URL not specified via --link flag.")
    }
```

При запуске моей программы я передаю ей агрумент ```--link``` и ссылку на репозиторий. Если нам не ввели ссылку печатаю сообщение, о неправильном использовании программы и завершаю ее.

Далее нам нужно создать временную директорию в которой будет располагаться наш репозиторий. Делаю это с помощью ```os.MkdirTemp```. Опять же при невозможности создания говорим об этом пользователю и завершаем программу. Также нужно удалить директорию после завершения работы с ней для этого через ```defer``` пропишем удаление директории, чтобы при звершении нашей main функции у нас почистилась директория.

```go
    repositoryDirectory, err := os.MkdirTemp("", "repo-dir-*")
	if err != nil {
		log.Fatalf("An error while creating temp directory: %v.", err)
	}
	defer os.RemoveAll(repositoryDirectory)
```

После создания директории можно перейти к клонированию репозитория.

```go
    func cloneRepository(url, directory string) error {
        command := exec.Command("git", "clone", "--depth", "1", url, directory)
        return command.Run()
    }
```

И опять при неожиданном поведении печатаем ошибку:

```go
    if err := cloneRepository(*gitURL, repositoryDirectory); err != nil {
        log.Fatalf("An error while cloning the repository: %v.", err)
    }
```

Теперь нам нужно рекурсивно обойти директорию и найти в ней go.mod файл, для этого воспользуемся ```filepath.walk``` и внутри будем смотреть. На данный момент реализация предполагает, что мы не работаем с мультимодульными проектами, но если нужно можно расширить ее созданием списка директорий в которых находятся go.mod файлы.


```go
    func findGoModFile(directory string) (string, error) {
        var goModPath string
        err := filepath.Walk(directory, func(path string, fi os.FileInfo, err error) error {
            if strings.HasSuffix(path, "go.mod") {
                goModPath = path
                return filepath.SkipAll
            }
            return nil
        })
        return goModPath, err
    }
```
Также созданим объект содержащий директорию в которой находится наш go.mod файл.

```go
    goModPath, err := findGoModFile(repositoryDirectory)
	if err != nil {
		log.Fatalf("go.mod file is not found: %v.", err)
	}
	goModDirectory := filepath.Dir(goModPath)
```

Далее нам нужно получить информацию о нашем проекте из go.mod. Напишем для этого функцию.

```go
    func getGoModInfo(directory string) ([]byte, error) {
        cmd := exec.Command("go", "mod", "edit", "-json")
        cmd.Dir = directory
        return cmd.Output()
    }
```

Доставать информацию будем с помощью ```go mod edit -json``` и в результате будем получать такого вида json объект.


```json
    {
        "Module": {
                "Path": "test"
        },
        "Go": "1.16",
        "Require": [
                {
                        "Path": "github.com/adshao/go-binance/v2",
                        "Version": "v2.3.5",
                        "Indirect": true
                },
                {
                        "Path": "github.com/bitly/go-simplejson",
                        "Version": "v0.5.0",
                        "Indirect": true
                },
                {
                        "Path": "github.com/gorilla/websocket",
                        "Version": "v1.5.0",
                        "Indirect": true
                }
        ],
        "Exclude": null,
        "Replace": null,
        "Retract": null
}
```

Теперь вопрос как нам эту информацию достать. Для этого воспользуемся ```json.Unmarshal```. Только для его использования нам нужно создать структуру для результата. Нас нужны оттуда только ```Module```, ```Go``` и ```Require```. На самом деле ```Require``` нам не обязательно нужен в текущей реалицации. Я больше испольщовал его для проверки работоспособности.


```go
    var goModData struct {
        Module struct {
            Path string
        }
        Go      string
        Require []struct {
            Path    string
            Version string
        }
    }
```

```go
    goModInfo, err := getGoModInfo(goModDirectory)
	if err != nil {
		log.Fatalf("An error while reading go.mod file: %v.", err)
	}
	json.Unmarshal(goModInfo, &goModData)
```

Теперь нам нужно проверить обновления для наших зависимостей. Напишем для этого функцию. Можно было снова брать json версию и парсить ее, но я решил что в данном случае удобнее будет брать просто вывод команды ```go list -u -m all```, который покажет нам все зависимости с обновлениями, если там не самая новая версия. Если версия не самая новая то в строке будет 3 аргумента а именно имя зависимости, текущая версия и новая версия. Вывод команды будет выглядеть вот так:

```txt
    test
    github.com/adshao/go-binance/v2 v2.3.5 [v2.8.3]
    github.com/bitly/go-simplejson v0.5.0 [v0.5.1]
    github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869
    github.com/davecgh/go-spew v1.1.0 [v1.1.1]
    github.com/gorilla/websocket v1.5.0 [v1.5.3]
    github.com/kr/pretty v0.2.0 [v0.3.1]
    github.com/kr/pty v1.1.1 [v1.1.8]
    github.com/kr/text v0.1.0 [v0.2.0]
    github.com/pmezard/go-difflib v1.0.0
    github.com/stretchr/objx v0.1.0 [v0.5.2]
    github.com/stretchr/testify v1.4.0 [v1.10.0]
    gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 [v1.0.0-20201130134442-10cb98267c6c]
    gopkg.in/yaml.v2 v2.2.2 [v2.4.0]
```

А функция выглядит следующим образом:

```go
    func checkUpdates(directory string) ([]byte, error) {
        command := exec.Command("go", "list", "-u", "-m", "all")
        command.Dir = directory
        return command.Output()
    }
```

И снова проверим наличие ошибок.

```go
    updates, err := checkUpdates(goModDirectory)
	if err != nil {
		log.Fatalf("An error while checking updates: %v.", err)
	}
```

Теперь нам осталось просто вывести имя модуля, версию Го в проекте(которые мы предварительно спарсили) и пропарсить вывод команды ```go list -u -m all``` разбив его на строки и каждую строку на отдельные слова.

```go
    fmt.Printf("Module: %s\n", goModData.Module.Path)
	fmt.Printf("Go version: %s\n", goModData.Go)
	fmt.Println("Updates:")

	for _, line := range strings.Split(string(updates), "\n") {
		if line == "" {
			continue
		}
		var tokens []string = strings.Split(string(line), " ")
		if len(tokens) < 3 {
			continue
		}
		fmt.Printf("\t%s %s -> %s\n", tokens[0], tokens[1], tokens[2])
	}
```

В результате работы программы на примере с таким go.mod файлом

```go
    module test

    go 1.16

    require (
        github.com/adshao/go-binance/v2 v2.3.5 // indirect
        github.com/bitly/go-simplejson v0.5.0 // indirect
        github.com/gorilla/websocket v1.5.0 // indirect
    )
```

получим следующий вывод

```txt
    Module: test
Go version: 1.16
Updates:
        github.com/adshao/go-binance/v2 v2.3.5 -> [v2.8.3]
        github.com/bitly/go-simplejson v0.5.0 -> [v0.5.1]
        github.com/davecgh/go-spew v1.1.0 -> [v1.1.1]
        github.com/gorilla/websocket v1.5.0 -> [v1.5.3]
        github.com/kr/pretty v0.2.0 -> [v0.3.1]
        github.com/kr/pty v1.1.1 -> [v1.1.8]
        github.com/kr/text v0.1.0 -> [v0.2.0]
        github.com/stretchr/objx v0.1.0 -> [v0.5.2]
        github.com/stretchr/testify v1.4.0 -> [v1.10.0]
        gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 -> [v1.0.0-20201130134442-10cb98267c6c]
        gopkg.in/yaml.v2 v2.2.2 -> [v2.4.0]
```

Все предыдущие выводы брались также с этого примера.
Вот [ссылка](https://github.com/unbanan/test.git) на него, если он до сих пор приватный пожалуйста напишите, я открою.

Telegram: https://t.me/JLEXA0

Если запустить программу на ее собственном [репозитории](https://github.com/unbanan/mws-task.git) где в go.mod нет зависимостей, то мы получим следующий ожидаемый вывод:

```txt
    Module: mws-task
    Go version: 1.22.2
    Updates:
```

т. к. наш go.mod файл выглядит следующим образом:

```go
    module mws-task

    go 1.22.2
```

# Запуск программы

```Go version: 1.22.2```

```Git version: 2.43.0```

```sh
git clone https://github.com/unbanan/mws-task.git
cd mws-task
go build
./mws-task --link [URL TO YOUR REPO]
```

# Реализация №2
TODO