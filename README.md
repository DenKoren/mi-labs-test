## Дано (предоставляется нами):


Docker контейнер небольшого веб-сервиса. Работает он так:

- Для запуска сервиса надо указать параметр SEED={some_seed} через переменную среды
- Инициализация сервиса занимает около 120 секунд (предположим, что при старте он что-то такое считает, что ему упрощает обработку запросов)
- Сервис слушает порт 8080; порт открывается только после инициализации сервиса
- На порту висят два ендпоинта:
    * `GET /health`
    * `GET /calculate/{some_input}`
      главный ендпоинт сервиса, который что-то считает и возвращает в ответ простую строку с результатом вычисления. Время вычисления 0-120 секунд, тоже может быть долгое.
- Сервис является чистой функцией относительно параметров some_seed и some_input, т.е. для одинаковых параметров всегда вернет одинаковое значение

Комментарий: реальный аналог такого сервсиса - это, например, сервис по выравниванию ДНК: на старте он загружает в оперативку базу данных генов, параметр старта SEED - это сид генератора случайных чисел, который используется в нечетких поисках, параметр input - последовательность ДНК, в результатет /calculate сервис отдает позицию в геноме человека и разметку этой ДНК на гены.

## Задача:

Написать небольшой сервис, в виде программы на go (или любом другом языке программирования), которая бы экспонировала вот такой endpoint:

- `GET /calculate/{some_seed}/{some_input}`
  запускает по необходимости контейнер с параметром SEED={some_seed} и дергает их /calculate/{some_input} , отдавая результат юзеру

К сервису предъявляются следующие требования:

- Запускать больше одного контейнера с определенным значением SEED не надо; запускать же несколько контейнеров с разными SEED нужно по необходимости
- Если к конкретному контейнеру нет запросов, он должен прожить еще некоторое время (пусть 120 секунд), чтобы заново не тратить время на его запуск, если придет очередной запрос, и после этого должен быть убит, чтобы высвободить ресурсы
- Если в данный момент к сервису есть несколько запросов с одинаковыми параметрами SEED и input, реально на вычисление к контейнеру должен пойти только один, и его результат вернуться всем (дедупликация вычислений)
- Если пользователь отвалился, и больше никому не надо считать результат с такими настройками, надо запрос к сервису тоже отменить (сервис правильно отработает завершение соединения и прекратит начатый счет)

Сервис должен работать на одной тачке соизмеримой по мощи с ноутбуком и выдерживать десятки запросов в минуту.


## Как дебагать:
Собрать образ-заглушку сервиса, который запускатор будет поднимть по запросу:
```bash
cd '<git repo root>/dev/compute'
env GOOS=linux go build .
docker build . --tag 'mi-labs-test:latest'
```

Собрать образ с самим сервисом
```bash
cd '<git repo root>'
docker build . --tag 'mi-labs-debug:latest' 
```

Запустить и подключиться через dlv на порт 2345
```bash
docker run \
  --publish '4334:4334' \
  --publish '4224:4224' \
  --publish '2345:2345' \
  --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
  mi-labs-debug:latest
```

## Как использовать
Собрать релизный образ
```bash
cd '<git repo root>'
docker build --file ./Dockerfile.release . --tag mi-labs-release:latest
```

Запустить:
```bash
docker run \
  --publish '127.0.0.1:4334:4334/tcp' \
  --publish '127.0.0.1:4224:4224/tcp' \
  --mount type=bind,src=/var/run/docker.sock,dst=/var/run/docker.sock \
  mi-labs-release:latest
```

Сходить в REST API или gRPC API, дождаться старта контейнера и ответа от него:
```bash
curl 'http://127.0.0.1:4224/v1/calculate/myseed/my-awesome-input-line'
```
