# ТЗ: PhotoList — шаг 5 (`5_csrf_token`)

## Цель

Закрыть XSS и CSRF с шага 4.

Комментарий снова экранируется. Изменяющие запросы (`upload`, `rate`) принимаются только с валидным CSRF-токеном, привязанным к текущей сессии.

Авторизация через cookie `session_id` и таблицу `sessions` не меняется. Это **не** замена сессии на JWT для логина. JWT здесь — один из способов **подписи CSRF-токена**. Полноценные JWT-сессии — следующий шаг курса, не этот.

## Что изменить

- Список фото — снова `html/template` (не `text/template`).
- Интерфейс `TokenManager`: `Create` / `Check`.
- Три реализации токена: HMAC, AES-GCM, JWT.
- В `List` создать токен, отдать в шаблон как `CSRFToken`.
- `Upload` проверяет токен из формы, `Rate` — из заголовка.
- В `main` подключить одну реализацию (по умолчанию JWT); остальные должны собираться и уметь подставляться сменой одной строки.

Секреты для подписи/шифрования — константы в коде для учёбы, не из env.

## Задачи

### XSS

Страница списка — только `html/template`. `{{.Comment}}` экранируется: `<script>` на странице виден как текст, не выполняется.

Логин и регистрация уже на `html/template` — оставить.

Проверка: загрузить комментарий `<script>alert(1)</script>`, открыть `/photos/` — alert нет, в HTML сущности вроде `<script>`.

### Интерфейс токена

```text
TokenManager
  Create(sess *Session, expUnix int64) (token string, err error)
  Check(sess *Session, token string) (ok bool, err error)
```

`Create` получает срок жизни Unix-временем (сейчас + 24 часа).

`Check` должен подтвердить:

- токен не битый / не подделан;
- не истёк;
- внутри те же `session ID` и `user ID`, что у текущей сессии из context.

Иначе `ok == false` или `err != nil`.

`PhotolistHandler` хранит `Tokens TokenManager` рядом со storage и шаблонами.

### Три реализации

Все три делают одно и то же снаружи (`Create`/`Check`), разный способ упаковки данных `sessionID`, `userID`, `exp`.

#### 1. HMAC (`HashToken`)

Данные для подписи: `sessionID:userID:exp`.

`Create`: HMAC-SHA256 с секретом → hex + `:` + `exp` строкой.  
Пример вида: `fbc1fd86...:1567618546`.

`Check`: разрезать по `:`, проверить `exp`, заново посчитать HMAC, сравнить через `hmac.Equal`.

Секрет — произвольная строка, например `"golangcourse"`.

#### 2. AES-GCM (`CryptToken`)

Полезная нагрузка — JSON:

```text
{ SessionID, UserID, Exp }
```

`Create`: AES-GCM, случайный nonce, `nonce || ciphertext`, целиком в base64.

`Check`: base64 → decrypt → JSON → `exp` не просрочен → `SessionID`/`UserID` совпадают с сессией.

Ключ AES — 32 байта (AES-256), строка длины 32.

#### 3. JWT (`JwtToken`)

Claims: `sid` (session id), `uid` (user id), стандартные `exp` / `iat`.  
Подпись: HS256, секрет — байты строки (тот же 32-символьный ключ ок).

`Create`: `jwt.NewWithClaims` + `SignedString`.

`Check`: `ParseWithClaims`, метод только HMAC HS256, `Valid()`, затем `sid`/`uid` == текущая сессия.

Зависимость: `github.com/golang-jwt/jwt` (как в курсе) либо актуальный совместимый пакет — главное HS256 и те же claims.

В `main` по умолчанию включить JWT. HMAC и AES оставить рядом и закомментированными конструкторами, чтобы можно было переключить без переписывания handler’ов.

### Выдача токена (`List`)

После `GetPhotos`:

1. `token, err := Tokens.Create(sess, now+24h)`.
2. Ошибка создания → `500`.
3. В шаблон:

```text
{ Items, CSRFToken }
```

Один и тот же `CSRFToken` используется и формой, и JS.

### Доставка на клиенте

Форма upload — скрытое поле:

```html
<input type="hidden" name="csrf-token" value="{{.CSRFToken}}" />
```

JS голоса — заголовок (не query):

```text
Header: csrf-token: <тот же CSRFToken>
POST /photos/rate?id=...&vote=up|down
```

Имена: поле формы `csrf-token`, заголовок `csrf-token`.

### Проверка на сервере

`Upload` (до работы с файлом):

- `token := r.FormValue("csrf-token")`
- `Check(sess, token)` не ок → `401`, тело `bad token`
- дальше прежняя логика загрузки

`Rate`:

- `token := r.Header.Get("csrf-token")`
- не ок → `401`, JSON `{"err": "bad token"}`
- дальше прежняя логика `id` / `vote` / JSON `{"id": ...}`

Сессию по-прежнему берёшь из context (middleware). Без сессии до проверки CSRF запрос не доходит (`401` No auth), как раньше.

Явно требовать только `POST` для `Rate` не обязательно: header картинка всё равно не выставит, старый CSRF через `<img>` уже не пройдёт.

### Как убедиться

**XSS закрыт.** Комментарий со `<script>alert(1)</script>` не выполняется.

**CSRF rate закрыт.**

- Кнопки `+`/`−` на `/photos/` работают (header с токеном).
- `GET`/`img` на `/photos/rate?id=1&vote=up` без header → `401` / bad token, рейтинг не растёт.
- `evil.html` с `<img src="http://localhost:8080/photos/rate?...">` не меняет рейтинг.

**CSRF upload закрыт.**

- Обычная загрузка с `/photos/` работает (hidden field).
- POST на `/upload` без поля / с чужим токеном → `401 bad token`.

**Смена реализации.** В `main` вместо JWT включить HMAC или AES — upload и rate продолжают работать без правок handler’ов.

### Не делать

- Не переносить логин на JWT вместо `sessions` (это шаг `6_jwt_sessions`).
- Не хранить CSRF-токены в БД: все три варианта — самодостаточные (подпись / шифр), сервер только считает `Check`.
- Не класть CSRF в cookie вместо form/header — смысл шага в synchronizer token на странице.
- Не оставлять список на `text/template`.
- Тесты на `TokenManager` не обязательны; моки storage поправить, если интерфейс handler’а разъехался.

