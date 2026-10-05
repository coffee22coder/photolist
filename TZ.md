# ТЗ: PhotoList — шаг 3 (`3_auth`)

## Цель

Появляется пользователь. Список и загрузка фотографий работают только после входа. Кто вошёл, определяется сессией в cookie и строкой в таблице `sessions`, не JWT.

`userID` больше не константа `0`. `List` и `Upload` берут его из сессии в `context`. Каждый пользователь видит и создаёт только свои фото.

Сохранение JPEG, превью, интерфейс `Storage` и MySQL не меняются. JWT на этом шаге не делать.

## Что изменить

- Таблицы `users` и `sessions`.
- Регистрация, логин, логаут.
- Cookie `session_id` + запись в `sessions`.
- Middleware: без сессии закрытые URL отвечают `401`.
- Маршруты фотографий: `/photos/`, `/photos/upload`.
- `GetPhotos` / `Add` получают `UserID` из сессии.

## Задачи

### Таблицы

`users`: `id` автоинкремент, `login` уникальная строка, `password` — `VARBINARY` (сырые байты, не строка).

`sessions`: `id` строка (32 символа), `user_id`.

`photos` без изменений. `user_id` в фото — это `users.id`.

### Пароль

Соль — случайные 8 символов. Хеш: `argon2.IDKey(password, salt, 1, 64*1024, 4, 32)`. В базу пишется `salt + hash` одним срезом байт.

Проверка на логине: из записи берутся первые 8 байт как соль, тот же `hashPass`, сравнение через `bytes.Equal` с тем, что лежит в `users.password`.

Зависимость: `golang.org/x/crypto/argon2`.

### Сессия

```text
Session { UserID, ID }
```

`CreateSession`: случайный `id` 32 символа, `INSERT INTO sessions(id, user_id)`, cookie `session_id`, срок 90 дней, `Path=/`.

`CheckSession`: cookie `session_id` → `SELECT user_id FROM sessions WHERE id = ?`. Нет cookie или нет строки — `ErrNoAuth`.

`DestroySession`: `DELETE` по `id` из контекста, cookie с истекшим сроком.

Сессию в запрос кладёт middleware: `context.WithValue`. Тип ключа — не `string`, отдельный `ctxKey`. Достать: `SessionFromContext`.

### Middleware

Закрытые URL требуют сессию, иначе `401` и тело `No auth`.

Открытые без обязательной сессии: `/user/login`, `/user/reg`, `/`.

Если cookie валидна, сессию всё равно клади в context на открытых URL: иначе `/` не отличит гостя от вошедшего.

`/images/` отдаётся без middleware, как статика.

`/user/logout` закрыт: выйти можно только с сессией.

### Пользователь

`UserHandler` с `*sql.DB` и шаблонами логина/регистрации.

`GET /user/login` — форма, поля `login`, `password`, `POST` на `/user/login`.  
`POST`: найти пользователя, сверить пароль. Нет пользователя / неверный пароль → `400`. Успех → сессия, редирект `302` на `/photos/`.

`GET /user/reg` — форма, `POST` на `/user/reg`.  
`POST`: соль, хеш, `INSERT INTO users(login, password)`. Ошибка вставки → `500`. Успех → сессия, редирект `302` на `/photos/`.

`GET/POST /user/logout` — уничтожить сессию, редирект `302` на `/user/login`.

`GET /` — нет сессии → `302` на `/user/login`. Есть сессия → `302` на `/photos/`.

### Фотографии

`List` и `Upload` снимают сессию из context. `GetPhotos(sess.UserID)`. `Add` с `UserID: sess.UserID` и `Path: md5Sum`.

Форма загрузки: `action="/photos/upload"`.

Маршруты:

```text
/photos/        List
/photos/upload  Upload
/user/login
/user/logout
/user/reg
/               Index
/images/        статика, без авторизации
```

Внутренний mux с этими обработчиками оборачивается middleware. На `DefaultServeMux`: `/` → middleware(mux), отдельно `/images/`.

### Не делать

JWT, refresh, CSRF, «забыли пароль», отдельный пакет auth. Тесты `Upload` не обязательны. Существующие тесты `List`/`GetPhotos` поправить только если перестала собираться сигнатура (`userID` из сессии, не литерал `0`).
