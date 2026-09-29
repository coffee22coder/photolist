# ТЗ: PhotoList — шаг 2 (`2_tests`)

## Цель

Список фотографий хранится в MySQL. Обработчик зависит от интерфейса хранилища, не от конкретной реализации. Хранилище и `List` покрыты тестами. В тестах настоящая база не поднимается.

Загрузка файла, превью, шаблон, маршруты и редирект `302` на `/photos` не меняются. `userID` по-прежнему `0`.

## Что изменить

- `St` в обработчике имеет тип `Storage`, не `*StMem`. Память `StMem` удаляется.
- Появляется реализация `StDb` на `database/sql`.
- `GetPhotos` возвращает только фото этого `user_id`. Фильтр живёт в SQL.
- Тесты `Add` и `GetPhotos` на `sqlmock`.
- Тест `List` на `gomock` и `httptest`.

## Задачи

### Интерфейс

```go
type Storage interface {
    Add(*Photo) error
    GetPhotos(int) ([]*Photo, error)
}
```

`PhotolistHandler.St` — этот интерфейс. `List` и `Upload` по-прежнему вызывают `GetPhotos(userID)` и `Add(&Photo{UserID: userID, Path: md5Sum})`.

### MySQL

Драйвер `github.com/go-sql-driver/mysql`. При старте открыть соединение и сделать `Ping`. Ошибка `Ping` останавливает процесс.

Строка соединения эталона:

```text
root:love@tcp(127.0.0.1:3306)/photolist?charset=utf8&interpolateParams=true
```

Таблица `photos`: `id` (автоинкремент), `user_id`, `path`.

`Add`:

- `INSERT INTO photos(user_id, path) VALUES(?, ?)`
- ошибка запроса возвращается как есть
- `LastInsertId == 0` тоже ошибка

`GetPhotos(userID)`:

- `select id, user_id, path from photos where user_id = ?`
- каждая строка сканируется в `Photo`
- ошибка запроса и ошибка `Scan` возвращаются вызывающему

В обработчике передаётся `NewDbStorage(db)`.

### Тесты хранилища

Пакет `gopkg.in/DATA-DOG/go-sqlmock.v1`. База — `sqlmock.New()`, не MySQL.

`Add`, фото `{UserID: 1, Path: "test"}`:

- успешный `INSERT`, `LastInsertId = 1` — ошибки нет, ожидания мока выполнены
- ошибка `Exec` — `Add` возвращает ошибку
- ошибка `LastInsertId` — `Add` возвращает ошибку
- `LastInsertId = 0` — `Add` возвращает ошибку

`GetPhotos`:

- строки `(1, userID, "tree")` и `(2, userID, "minion")` совпадают с результатом
- ошибка запроса — метод возвращает ошибку
- строка с другим набором колонок — ошибка `Scan`

### Тест List

Мок интерфейса: `mockgen -source=handlers.go -destination=handlers_mock.go -package=main Storage`.

Зависимость `github.com/golang/mock/gomock`. Запрос через `httptest`, шаблон настоящий.

- `GetPhotos(0)` возвращает `[{ID: 1, UserID: 1, Path: "my_photo_name"}]` — в теле есть `"/images/my_photo_name_160.jpg"`
- `GetPhotos` возвращает ошибку — статус `500`
- шаблон с несуществующим полем — статус `500`
