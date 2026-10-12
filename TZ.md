# ТЗ: PhotoList — шаг 6 (`6_jwt_sessions`)

## Цель

Вынести авторизацию за интерфейс `SessionManager`. Логин, middleware и логаут работают через него, а не через прямые вызовы `CheckSession` / `CreateSession` в `DBStorage`.

Добавить сессии на **JWT в cookie** (stateless) и вариант **JWT + `users.ver`** для отзыва токенов после смены пароля.

CSRF (`TokenManager`), upload, rate, комментарии и рейтинг с шага 5 **не убирать**. JWT в этом шаге — **сессия пользователя**, не CSRF-токен.

## Подшаги (порядок работы)

### Уже сделано (база шага 6)

- `session.go` — `SessionManager`, `authMiddleware`, сессия в context (`authKey` / `sessionFromContext`).
- `sessionDB.go` — `SessionDBManager` + `SessionStorage`, cookie `session_id`.
- `user.go` — логин / регистрация / логаут через `Sm` (без `change_pass`, без `ver` в модели).
- `sessionJWT.go` — `SessionJWTManager`: cookie `session`, `Check` / `Create` / `Destroy*`.

Допустимые отличия от эталона курса (если осознанно): `context` в методах менеджера и storage; на `GET /` опциональный `Check` и кладка сессии в context для редиректа на `/photos/`.

### Текущий подшаг: `SessionsJWTVer` и закрытие шага 6

Сделать **сейчас**, до приёмки шага 6:

1. **Миграция БД** — колонка `users.ver` (см. раздел «Миграция» ниже).
2. **`storage` / `User`** — при логине и проверке пароля: `SELECT id, login, ver, password …`; у нового пользователя `Ver: 0`. При необходимости выровнять типы с ТЗ (`uint32` для id).
3. **`sessionJWTVer.go`** (или аналог) — копия логики `SessionsJWT` + поле `ver` в claims; структура с `Secret` и доступом к БД.
4. **`Check` (JWTVer)** — после успешного парса JWT: `SELECT ver FROM users WHERE id = ?`; если `claims.ver != ver` → `ErrNoAuth`.
5. **`Create` (JWTVer)** — в JWT класть актуальный `user.Ver` (нужен `*User` или явная передача `ver`, не только `userID`).
6. **`UserHandler.ChangePassword`** — `GET`/`POST`, шаблон `change_password`; `UPDATE … ver = ver + 1`, `DestroyAll`, `Create`, редирект `/photos/`.
7. **`main`** — по умолчанию `NewSessionsJWTVer(secret, db)`; одной строкой можно переключить на `SessionsDB` / `SessionsJWT`.
8. **Ручная приёмка** — пункты из раздела «Приёмка» (логин без строк в `sessions`, смена пароля отзывает старый JWT, CSRF/upload с сессией из context).

Не обязательно на этом подшаге: переименование `authMiddleware` → `AuthMiddleware`, унификация json-тега `uid` vs `user_id`, TTL cookie 90 дней — можно после прохождения приёмки.

### Следующий подшаг: тесты `photolist_test.go`

После рабочего `SessionsJWTVer` в `main`:

- Привести тесты к актуальному API: `NewPhotoHandler(st, tmpl, tm)` — сейчас в тестах два аргумента, в коде три (`TokenManager` для CSRF на `List`).
- Обновить мок storage / `MockStorage` под интерфейс `PhotoStorage` (если сигнатуры менялись).
- Контекст сессии в тестах: тот же ключ, что в middleware (`authKey` или будущий `sessionKey`).
- Прогон: `go test ./...` без ошибок компиляции и с ожидаемыми статусами 200/500 в `TestList`.

## Что изменить

- `SessionManager`: `Check`, `Create`, `DestroyCurrent`, `DestroyAll`.
- Три реализации: `SessionsDB`, `SessionsJWT`, `SessionsJWTVer`.
- `AuthMiddleware(sm SessionManager, …)` вместо `authMiddleware(st *DBStorage, …)`.
- `UserHandler` вызывает `Sessions.Create` / `Destroy*`, не пишет cookie и не ходит в `sessions` сам.
- В `users` колонка `ver` (`int`, по умолчанию `0`).
- Маршрут `POST /user/change_pass`: смена пароля, `ver = ver + 1`, сброс сессий пользователя, новая сессия.
- `User.ID` / `Session.UserID` — `uint32` (как в эталоне).
- В `main` по умолчанию `NewSessionsJWTVer(secret, db)`; `SessionsDB` и `SessionsJWT` должны подключаться сменой одной строки.

Таблица `sessions` нужна только для режима `SessionsDB`. В JWT-режимах строки в `sessions` не создаются.

## Задачи

### Общий контракт

```text
SessionManager
  Check(r *http.Request) (*Session, error)
  Create(w http.ResponseWriter, user *User) error
  DestroyCurrent(w http.ResponseWriter, r *http.Request) error
  DestroyAll(w http.ResponseWriter, user *User) error
```

```text
User { ID uint32, Login string, Ver int32 }
Session { UserID uint32, ID string }
```

Контекст запроса: `SessionFromContext` / ключ как в эталоне (`sessionKey`). Handlers фото и CSRF берут сессию только из context.

`AuthMiddleware`: пути `/`, `/user/login`, `/user/reg` — без обязательной сессии (как в эталоне курса). Остальные — `sm.Check`, при ошибке `401` `No auth`, иначе сессия в context.

`Index` (`GET /`): нет сессии в context → `302` `/user/login`, есть → `302` `/photos/`. (В эталоне middleware на `/` сессию не кладёт — учти при ручной проверке; свой вариант с cookie на `/` можно оставить, если уже сделан.)

### SessionsDB

Поведение как на шаге 3:

- Cookie `session_id`, значение — id строки в `sessions`.
- `Check`: cookie → `SELECT user_id FROM sessions WHERE id = ?`.
- `Create`: случайный id 32 символа, `INSERT`, cookie ~90 дней.
- `DestroyCurrent`: `DELETE` по id из context, протухшая cookie.
- `DestroyAll`: `DELETE FROM sessions WHERE user_id = ?`.

### SessionsJWT

- Cookie **`session`**, значение — подписанный JWT (HS256).
- Claims: `uid` (`UserID`), `jti` через `StandardClaims.Id` (случайная строка), `exp`, `iat`.
- `Check`: парсинг JWT, `Valid()`, без запроса в БД.
- `Create`: выпустить JWT, положить в cookie.
- `DestroyCurrent`: только сброс cookie (токен на клиенте остаётся валидным до `exp` — осознанный минус stateless).
- `DestroyAll`: no-op (другие устройства не отозвать).

Зависимость: `github.com/golang-jwt/jwt` (как в эталоне).

### SessionsJWTVer

Как `SessionsJWT`, плюс в claims поле `ver` (текущее `user.Ver`).

`Check` после парсинга JWT:

- `SELECT ver FROM users WHERE id = ?`
- если `claims.ver != ver` в БД → `ErrNoAuth`

`Create` кладёт в JWT актуальный `user.Ver`.

Смена пароля увеличивает `ver` в БД — все старые JWT этого пользователя перестают проходить `Check`, без таблицы `sessions`.

### UserHandler

- `Login` / `Reg`: после проверки пароля — `Sessions.Create(w, user)`, редирект `/photos/`.
- `Logout`: `Sessions.DestroyCurrent`, редирект `/user/login`.
- `ChangePassword`:
  - `GET` — форма `change_password`;
  - `POST`: `pass1 == pass2`, проверка `old_password`, `UPDATE users SET password = ?, ver = ver + 1`, `user.Ver++`, `DestroyAll`, `Create`, редирект `/photos/`.

Логин при выборке пользователя: `SELECT id, login, ver, password …` (общий метод проверки пароля, как в эталоне).

Регистрация: новый пользователь с `Ver: 0`.

### main

```text
// sm := NewSessionsDB(db)
// sm := NewSessionsJWT("golangcourseSessionSecret")
sm := NewSessionsJWTVer("golangcourseSessionSecret", db)

http.Handle("/", AuthMiddleware(sm, mux))
```

`UserHandler.Sessions = sm`. Убрать из middleware зависимость от `DBStorage` для сессий.

Секрет JWT — константа в коде (учебный проект).

### Миграция

`users`: добавить `ver TINYINT NOT NULL DEFAULT 0` (или `INT`).

Существующие пользователи: `ver = 0`.

### Приёмка

1. **SessionsDB** (если включить в main): логин, галерея, логаут — как на шаге 3; после логаута закрытые URL → `401`.
2. **SessionsJWTVer** (режим по умолчанию): логин → cookie `session`, `/photos/` открывается; в БД новых строк в `sessions` нет.
3. Смена пароля: старый JWT перестаёт работать (`401` на `/photos/`), после успешного change — новая сессия, галерея снова доступна.
4. CSRF upload/rate с шага 5 по-прежнему работают с сессией из context.
5. Два пользователя видят разные фото (`UserID` из JWT).

### Не делать

- Не смешивать cookie `session_id` (БД) и `session` (JWT) в одной реализации.
- Не хранить JWT-сессии в таблице `sessions`.
- Не удалять `TokenManager` и проверки CSRF.
- Refresh-токены, OAuth, Redis blacklist JWT — вне шага.
