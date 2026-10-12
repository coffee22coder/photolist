# ТЗ: PhotoList — шаг 7 (`7_oauth`)

Эталон: `/Users/ermakov/GO/course/10/photolist/7_oauth`  
Мини-пример OAuth: `/Users/ermakov/GO/course/10/oauth`

## Цель

Вход через **VK OAuth** в дополнение к логину/паролю. После успешного OAuth — тот же `SessionManager` (`SessionsJWTVer` по умолчанию), что и при обычном логине.

Шаг 6 (**SessionManager**, JWTVer, CSRF, upload, rate) **не откатывать**.

## Поток (как в курсе)

```text
Пользователь → «Войти через VK» (ссылка на authorize)
  → VK → redirect на /user/login_oauth?code=...
  → Exchange(code) → access token + extra: email, user_id
  → login := email или "vk" + user_id
  → найти/создать пользователя в users
  → Sm.Create → cookie session → 302 /photos/
```

Схема: `course/10/oauth/notes.txt` (sequence diagram).

## Зависимости

```text
golang.org/x/oauth2
golang.org/x/oauth2/vk
```

## Конфиг VK (свои значения)

1. Создать приложение: https://vk.com/apps?act=manage  
2. Документация: https://dev.vk.com/ru/api/getting-started  
3. В коде (константы, как в курсе — учебный проект):
   - `client_id` (APP_ID)
   - `client_secret` (APP_KEY)
   - **Redirect URI** строго: `http://localhost:8080/user/login_oauth`
   - Authorize URL: `response_type=code`, `scope=email` (чтобы в ответе Exchange был `email`)

В эталоне зашиты чужие id/secret — **заменить на свои**, иначе Exchange упадёт.

## Что изменить

### 1. `main`

- Маршрут: `mux.HandleFunc("/user/login_oauth", uh.LoginOauth)` (имя handler — как удобно, путь как выше).

### 2. `session.go` — публичные пути

Добавить в `noAuthPaths`:

```text
"/user/login_oauth"
```

Иначе callback с `code` упрётся в middleware без сессии → 401.

### 3. `login.html` + `UserHandler.Login` (GET)

- На форме логина — ссылка «Войти через VK» на authorize URL (в курсе в шаблон передаётся `VKAuthURL`).
- URL совпадает с `redirect_uri` в настройках VK и в `oauth2.Config.RedirectURL`.

### 4. Handler `LoginOauth`

**Вход:** `GET /user/login_oauth?code=...` (параметр `code` из query; в курсе `r.FormValue("code")`).

**Шаги:**

1. Нет `code` → `400`.
2. `oauth2.Config` с `vk.Endpoint`, `Exchange(ctx, code)`.
3. Из `token.Extra("email")` и `token.Extra("user_id")` собрать логин:
   - есть email → login = email;
   - иначе → `vk` + string(user_id) (в курсе `strconv`, `user_id` приходит как `float64` из JSON).
4. **Пользователь в БД:**
   - если login уже есть → взять `id`, `ver` (как `errUserExists` в курсе — **не ошибка**, всё равно логиним);
   - иначе → insert с случайным паролем (50 символов), `ver = 0` (пароль для OAuth-пользователя не используется при входе через VK).
5. `Sm.Create(ctx, userID, w)` (или `Create(w, user)` — как у тебя в интерфейсе).
6. `302` → `/photos/`.

Ошибки Exchange / БД — `500` + лог (как в эталоне).

**Не обязательно на этом шаге:** запрос `users.get` за аватаром (в `7_oauth/user.go` типы есть, в `LoginOauth` API VK не вызывается; полный пример — `course/10/oauth/main.go`).

### 5. Storage / user

Вынести общую логику «создать или вернуть существующего по login» (в курсе `createUser(login, pass)`):

- `SELECT id, ver FROM users WHERE login = ?`
- нет строки → `INSERT` с `hashPass`
- есть → вернуть пользователя + признак «уже был»

Обычный `Reg` по-прежнему не должен молча логинить существующего — только OAuth-path допускает «exists → OK».

### 6. Сессии

OAuth **не** пишет в `sessions` (режим JWTVer). Только cookie `session` через `Sm`.

## Приёмка

1. Логин/пароль и галерея работают как на шаге 6.
2. На `/user/login` есть ссылка VK → редирект на VK → после согласия возврат на `/user/login_oauth` → `/photos/` без ручного ввода пароля.
3. Повторный OAuth тем же VK-аккаунтом → тот же `login` в БД, новая сессия, дубликат пользователя не создаётся.
4. Новый VK → новая строка в `users`, `ver = 0`.
5. CSRF на upload/rate не сломан.

## Не делать

- Другие провайдеры (Google, GitHub) — вне шага.
- Привязка OAuth к отдельной таблице `oauth_ids` — в курсе нет.
- Менять redirect на `/user/login` (в эталоне отдельный `/user/login_oauth`).

## Следующие шаги курса (пока не трогать)

| Папка | Тема |
|--------|------|
| `8_frontend` | embed статики (`assets_gen.go`), air |
| `9_ajax` | JSON API, `token_middleware`, ajax-рейтинг |

## Подшаги (рекомендуемый порядок)

1. Зависимости + константы VK + приложение VK с redirect URI.
2. `noAuthPaths` + маршрут + пустой handler с проверкой `code`.
3. `Exchange` + разбор extras + логин.
4. `createUser` / find-or-create в storage.
5. `Sm.Create` + редирект.
6. Ссылка на login.html + ручной прогон в браузере.

После шага 7: при необходимости снова `go test ./...` (шаг 6 в `TZ.md` — тесты `photolist_test.go`).
