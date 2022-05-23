# Alex
SDK для создания, тестирования на исторических данных и в песочнице, и выполнения на реальном счетё торговых стратегий, созданных на golang.

### Быстрый старт

**0. Зарегистрируйтесь в Tinkoff, и получите токен доступа к API**
[Токен для работы с TINKOFF INVEST API](https://tinkoff.github.io/investAPI/token/)

**1. Соберите проект**

Скопируйте исходный код проекта себе

`git clone https://github.com/go-trading/alex`

Для получения исполняемого файла командной строки, перейдите в каталог `alex` и выполните следующию команду (требуется установленнай компилятор golang версии 1.18 или выше): 

`go build .  `

Для получения docker образа в корневом каталоге выполните 

`docker build .				`

**2. Скачайте историю свечей по инструменту**

`./alex load --figi=BBG000000001 --token=******************`

Можно указать сразу несколько бумаг, указав атрибут `figi` несколько раз.

По умолчанию скачивается последняя неделя, с помощью аргументов `from` и `to` можно указать какой период интересует.

При загрузке указанный диапазон будет разбит на максимально доступные для такого размера свечей интервалы, и запросы будут выполняться с учётом лимитного грейда, замедляясь при достижении лимита.

См. все возможные аргументы с помощью аргумента `-h`. 

**3. Протестируйте робота на исторических данных**

`./alex bot history --figi=BBG000000001 --timeframe=7 --rsi4buy=45 --rsi4sell=55`

**4. Откройте счёт в песочнице**

`./alex sandbox open --token=**********`

**5. Положите на счет средства (по умолчанию кладётся 200 тысяч :))) **

`./alex sandbox pay-in --account=значение-с-предыдущего-шага --token=**********`

**6. Запустите робота в песочнице**

`./alex bot rsi --account=значение-с-предыдущего-шага --figi=BBG000000001 --timeframe=7 --rsi4buy=45 --rsi4sell=55 --token=**********`

** Если в параметре account, будет указан номер боевого счёта, то робот будет торгавать на бою**

**7. Узнайте номер боевого счёта**

`./alex accounts --token=**********`

## Структура проекта
`alex` — Утилита командной строки, позволяющая использовать функциональность библиотек. **[См. описание утилиты](https://github.com/go-trading/alex/wiki/%D0%9A%D0%BE%D0%BC%D0%B0%D0%BD%D0%B4%D0%BD%D0%B0%D1%8F-%D1%81%D1%82%D1%80%D0%BE%D0%BA%D0%B0)**

`bots` — Примеры торговых роботов.  **[См. написание торгового робота](https://github.com/go-trading/alex/wiki/%D0%A1%D0%BE%D0%B7%D0%B4%D0%B0%D0%BD%D0%B8%D0%B5-%D1%82%D0%BE%D1%80%D0%B3%D0%BE%D0%B2%D0%BE%D0%B9-%D1%81%D1%82%D1%80%D0%B0%D1%82%D0%B5%D0%B3%D0%B8%D0%B8)**

`history` — Клиент, реализация тестирования на исторических данных

`tinkoff` — Клиент для тестирования в песочнице, или торговле на реальном счёте Tinkoff

`grafana` - Исходники примера дашборта grafana, и его скриншот

`корневой каталог` — SDK для написания роботов


**MAINTAINER**

Alexey Nebotov