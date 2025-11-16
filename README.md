# Тестовое задание
___
Задание можно посмотреть [тут](https://github.com/avito-tech/tech-internship/tree/main/Tech%20Internships/Backend/Backend-trainee-assignment-autumn-2025)
### Требования
```markdown
Go 1.25.1
Docker
```
### Запуск:
```markdown
1. git clone https://github.com/lein3000zzz/PRAssigner.git
2. cd PRAssigner/deployments
3. (optional) docker compose build --no-cache
4. docker compose up
```
- p.s Конфиг можно менять под себя; админский jwt токен, который пропадет через год: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3OTQ4MDk4OTEsInJvbGUiOiJhZG1pbiJ9.ECCsDmmWSFyyyPyY3K7a5WiTMEocvG_JCd6vCKicGqY
` 
1. API доступен на <http://localhost:8080/>
2. PostgreSQL доступен на <http://localhost:5432>
3. grafana доступна на <http://localhost:3000/>
   - victoriaMetrics (ее порт не экспоузится для безопасности) уже указана в источниках, через файл `/deployments/grafana-config/provisioning/datasources/datasource.yml`)
   - Построение дашборда требует лишь захода в графану и добавления предложенной визуализации
### Дополнительные комментарии (основная часть выполнена)
1. В задании в .md ничего не сказано про админский токен и авторизацию, но в openapi спецификации было указано в setActive про админский токен 
   - Добавил авторизацию через jwt токен, который можно сгенерировать отдельным скриптом. 
     - Рабочий токен для текущего секрета, который будет работать еще год: `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3OTQ4MDk4OTEsInJvbGUiOiJhZG1pbiJ9.ECCsDmmWSFyyyPyY3K7a5WiTMEocvG_JCd6vCKicGqY
`
     - Нужно использовать в хедерах "Authorization": "Bearer SECRET_KEY"
     - Также добавил эту на эндпоинт массового
     - Код генерации ключа с текущим секретом: 
     ```go
     package main

     import (
         "fmt"
         "time"
    
         "github.com/golang-jwt/jwt/v5"
     )
    
     var AdminSecret = []byte("Abobus")
    
	 func main() {
	 	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	        "role": "admin",
	    	  "exp":  time.Now().Add(365 * 24 * time.Hour).Unix(),
		})
    
		s, _ := token.SignedString(AdminSecret)
		fmt.Println(s)
     }
     ```
     - Логика jwt в сервисе основана на [gin-jwt](https://github.com/appleboy/gin-jwt), что в текущей ситуации является оверкиллом (легковеснее было бы написать свое решение (свою милдварь)), но хорошо подходит для расширения логики (потому что её описания вообще не было)
2. Добавил эндпоинт для сбора статистики по команде:
   - Метод: `GET` 
   - Эндпоинт: `/team/pr-stats` / <http://localhost:8080/team/pr-stats>
   - Параметры: `team_name` (string)
   - Примеры ответов:
     - Ответ на запрос к существующей команде:
       ```json
       {
         "team_name": "test6",
         "team_stats": {
           "u127": {
             "open_count": 0,
             "merged_count": 0
           },
           "u128": {
             "open_count": 1,
             "merged_count": 1
           },
           "u129": {
             "open_count": 0,
             "merged_count": 2
           }
         }
       }
       ``` 
     - Ответ на запрос к несуществующей команде:
       ```json
       {
         "error": {
           "code": "NOT_FOUND",
           "message": "resource not found"
         }
       }
       ``` 
3. Сделал безопасную переназначаемость открытых PR
4. Добавил деактивацию юзеров по названию команды (метод массовой деактивации пользователей команды)
    - Метод: `POST`
    - Эндпоинт: `/users/deactivateTeam` / <http://localhost:8080/user/deactivateTeam>
    - Пример тела запроса:
   ```json
    {
      "team_name": "test8"
    }
   ```
    - Примеры ответов:
        - Ответ на запрос к существующей команде:
          ```json
          {
            "team_name": "test6",
            "team_stats": {
              "u127": {
                "open_count": 0,
                "merged_count": 0
              },
              "u128": {
                "open_count": 1,
                "merged_count": 1
              },
              "u129": {
                "open_count": 0,
                "merged_count": 2
              }
            }
          }
          ``` 
        - Ответ на запрос к несуществующей команде:
          ```json
          {
            "error": {
              "code": "NOT_FOUND",
              "message": "resource not found"
            }
          }
          ``` 
5. Покрыл код репозиториев юнит-тестами и получил файлы покрытия.
6. Добавил возможность получать pprof дампы в local.env окружении по эндпоинту `/debug/pprof` / <http://localhost:8080/debug/pprof>
7. Добавил метрики с [prometheus api](https://github.com/prometheus/client_golang/) и подключил к victoriaMetrics, которая лучше прометеуса по производительности
   - Добавил grafana, в конфиге сразу указан нужный datasource для создания визуализаций.
   - В приложении метрики скрыты, работают на отдельном порту из конфига по эндпоинту `/metrics`
8. Провел нагрузочное тестирование приложения с помощью [vegeta](https://github.com/tsenart/vegeta)
   - Версия использования vegeta: [библиотечная](https://github.com/tsenart/vegeta?tab=readme-ov-file#usage-library).
   - Параметры vegeta:
    ```go
    rate := vegeta.Rate{
        Freq: 100, Per: time.Second,
    }
    duration := 100 * time.Second
    ```
   - Вывод моей программы по результатам нагрузочного тестирования:
    ```markdown
    Requests: 10000
    Rate: 100.01
    Success: 100.00%
    99th latency: 4.579689ms
    ```
   - Метрики во время нагрузочного тестирования:

<p align="center">
  <img height="300" src="https://github.com/lein3000zzz/PRAssigner/blob/main/assets/images/readme/loadTestGrafana.png?raw=true" alt="🦍"/>
</p>

9. Описал конфигурацию линтера в .golangci.yml
   - Выполняется командой: 
   ```console
   golangci-lint run ./...  
   ```
10. (Не реализовано) При E2E / интеграционном тестировании использовал бы [testcontainers](https://golang.testcontainers.org/)

<details>
    <summary>Cat picture</summary>
    <p align="center">
        <img align="center" height="300" src="https://github.com/lein3000zzz/PRAssigner/blob/main/assets/images/readme/me_irl.png?raw=true" alt="🦍"/>
    </p>
</details>
