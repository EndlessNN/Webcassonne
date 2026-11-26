# API: REST + WebSocket

> Рішення: [ADR-0003](adr/0003-backend-stack.md), [ADR-0005](adr/0005-guest-identity-and-realtime.md). Сутності: [data-model.md](data-model.md).
> Типи клієнта: [`web/src/api/types.ts`](../web/src/api/types.ts). Зміна контракту — спершу тут, потім сервер і клієнт в одному коміті.
> Промпт: [prompts/4.3_technical_contracts.md](../prompts/4.3_technical_contracts.md).

Базовий шлях: `/api`. Формат: JSON, UTF-8, поля в `camelCase`. Статус реалізації: ✅ реалізовано, ⏳ реалізується разом з історією. Shortcut — реалізовано спрощено до вказаної історії.

---

## 1. Єдиний формат помилки

Будь-яка помилка (REST-відповідь з кодом 4xx/5xx або WebSocket-повідомлення `error`) має тіло:

```json
{
  "error": {
    "code": "NICKNAME_TAKEN",
    "message": "Цей нікнейм уже зайнятий у кімнаті",
    "details": { "field": "nickname" }
  }
}
```

| Поле | Обов'язкове | Опис |
|---|---|---|
| `code` | так | Машинний код з таблиці нижче. UI вирішує поведінку за ним |
| `message` | так | Текст українською, який можна показати користувачу |
| `details` | ні | Додаткові дані, напр. `field` для помилки валідації |

### Коди помилок

| Код | HTTP | Де | Коли | Історія |
|---|---|---|---|---|
| `VALIDATION_ERROR` | 400 | REST, WS | Нікнейм не 1–20 символів, неправильний JSON або поля; у WS також невідомий `type` і `rotation` поза {0, 90, 180, 270} | US-01, US-03, US-12 |
| `INVALID_TOKEN` | 401 | WS handshake | Токен відсутній або не належить кімнаті | — |
| `NOT_FOUND` | 404 | REST | Невідомий ендпоінт | — |
| `ROOM_NOT_FOUND` | 404 | REST, WS | Кімнати з таким кодом немає | US-03 |
| `ROOM_FULL` | 409 | REST | У кімнаті вже 5 гравців | US-03 |
| `GAME_ALREADY_STARTED` | 409 | REST, WS | Статус кімнати не `LOBBY`: вхід у кімнату або повторний `start_game` | US-03, US-05 |
| `NICKNAME_TAKEN` | 409 | REST | Нікнейм уже є в кімнаті (без урахування регістру) | US-03 |
| `NOT_HOST` | — | WS | `start_game` надіслав не організатор | US-05 |
| `NOT_ENOUGH_PLAYERS` | — | WS | `start_game` при 1 гравці | US-05 |
| `NOT_YOUR_TURN` | — | WS | Дію надіслав не активний гравець | US-07 |
| `WRONG_PHASE` | — | WS | Дія не відповідає фазі або статусу кімнати (напр. `place_meeple` у `PLACE_TILE`, `place_tile` у `LOBBY`, будь-яка ігрова дія у `FINISHED`/`ABANDONED`) | US-12…US-14 |
| `ILLEGAL_PLACEMENT` | — | WS | Клітинка зайнята, без сусідів або сторони не збігаються | US-12 |
| `ILLEGAL_MEEPLE` | — | WS | Сегмент не з `legalMeepleSegments` | US-13 |
| `NO_MEEPLES_LEFT` | — | WS | У гравця 0 міплів у запасі | US-13 |
| `INTERNAL` | 500 | REST, WS | Непередбачена помилка сервера (деталі лише в лозі) | — |

WS-помилки надсилаються лише автору дії і не змінюють стан гри.

### ✅ Порядок перевірок і тексти WS-помилок (US-05…US-14)

Сервер перевіряє дію в такому порядку і повертає першу помилку: формат повідомлення → статус кімнати → автор (організатор / активний гравець) → фаза → правило гри. Тому, напр., `place_tile` не у свій хід завжди дає `NOT_YOUR_TURN`, навіть якщо клітинка ще й зайнята.

| Код | `message` | Коли |
|---|---|---|
| `VALIDATION_ERROR` | `Невідомий тип повідомлення` | `type` не з таблиці «Клієнт → сервер» |
| `VALIDATION_ERROR` | `Неправильне повідомлення` | Неправильний JSON, відсутні або зайві поля, `rotation` поза {0, 90, 180, 270} |
| `GAME_ALREADY_STARTED` | `Партія вже почалася` | `start_game` не в `LOBBY` |
| `NOT_HOST` | `Почати гру може лише організатор` | `start_game` від не організатора |
| `NOT_ENOUGH_PLAYERS` | `Потрібно щонайменше 2 гравці` | `start_game` при 1 гравці (текст — з AC US-05) |
| `WRONG_PHASE` | `Ця дія зараз недоступна` | Ігрова дія не в `PLAYING` або не в тій фазі |
| `NOT_YOUR_TURN` | `Зараз хід іншого гравця` | Ігрова дія від неактивного гравця (US-07) |
| `ILLEGAL_PLACEMENT` | `Клітинка вже зайнята` | `place_tile` на клітинку з тайлом |
| `ILLEGAL_PLACEMENT` | `Тайл має межувати з іншим тайлом` | `place_tile` на клітинку без сусідів (AC US-12 «Клітинка без сусідів») |
| `ILLEGAL_PLACEMENT` | `Сторони тайлів мають збігатися` | Хоча б одна суміжна сторона не збігається (текст — з AC US-12) |
| `ILLEGAL_MEEPLE` | `Міпла не можна поставити на цей сегмент` | `place_meeple` із сегментом не з `legalMeepleSegments` |

**Shortcut до US-13:** поки міплів немає, `legalMeepleSegments` завжди `[]`, тож будь-який `place_meeple` у фазі `PLACE_MEEPLE` отримує `ILLEGAL_MEEPLE`. Хід завершується лише через `skip_meeple`.

---

## 2. REST

### ✅ `GET /api/health` — перевірка здоров'я

`200 OK`
```json
{ "status": "ok", "storage": { "type": "memory", "rooms": 0 } }
```

### ✅ `POST /api/rooms` — створити кімнату (US-01)

Запит:
```json
{ "nickname": "Олег" }
```

`201 Created` — творець стає гравцем і організатором:
```json
{ "roomCode": "K7QM2X", "playerId": "3f1c…", "token": "9a8b…" }
```

Помилки: `400 VALIDATION_ERROR`.

### ✅ `GET /api/rooms/{code}` — попередній перегляд для запрошення (US-03)

Відкриває сторінку `/r/{code}`, щоб одразу показати «заповнена / вже почалася», ще до введення нікнейму.

`200 OK`
```json
{ "code": "K7QM2X", "status": "LOBBY", "playerCount": 3, "maxPlayers": 5 }
```

Помилки: `404 ROOM_NOT_FOUND`.

### ✅ `GET /api/tiles` — каталог типів тайлів (US-06, US-09)

Статичний каталог з [docs/tiles.md](tiles.md) у форматі [TileType](data-model.md#tiletype-статичний-каталог). Клієнт малює тайли схематично із сегментів і не тримає власної копії каталогу. Склад стопки не таємний; таємний лише її порядок.

`200 OK`
```json
{
  "tiles": [
    {
      "id": "D", "count": 4, "isStart": true,
      "segments": [
        { "type": "CITY", "points": ["N0", "N1", "N2"], "pennant": false, "adjacentCities": [] },
        { "type": "ROAD", "points": ["E1", "W1"], "pennant": false, "adjacentCities": [] },
        { "type": "FIELD", "points": ["E0", "W2"], "pennant": false, "adjacentCities": [0] },
        { "type": "FIELD", "points": ["E2", "S0", "S1", "S2", "W0"], "pennant": false, "adjacentCities": [] }
      ]
    }
  ]
}
```
Приклад показує формат, а не дані: вміст — лише з `docs/tiles.md`.

### ✅ `POST /api/rooms/{code}/players` — приєднатися (US-03)

Запит:
```json
{ "nickname": "Ірина" }
```

`201 Created`
```json
{ "playerId": "b72e…", "token": "41fd…" }
```

Помилки: `400 VALIDATION_ERROR`, `404 ROOM_NOT_FOUND`, `409 ROOM_FULL`, `409 GAME_ALREADY_STARTED`, `409 NICKNAME_TAKEN`.

---

## 3. WebSocket

### ✅ `GET /api/rooms/{code}/ws?token={token}` — канал кімнати (лобі US-04 ✅; хід без міплів US-05…US-12, US-14…US-16 ✅; міпли US-13 і підрахунок US-18…US-23 ⏳)

- Успіх: `101 Switching Protocols`, одразу після підключення сервер надсилає `room_state`.
- Відмова до апгрейду: `401 INVALID_TOKEN`, `404 ROOM_NOT_FOUND` (REST-формат помилки).
- Повторне підключення з тим самим токеном (друга вкладка) закриває попереднє з'єднання.
- Повідомлення з невідомим `type` отримує `error` з кодом `VALIDATION_ERROR` (див. «Порядок перевірок» у §1).
- Закриття з'єднання: у `LOBBY` гравець видаляється з кімнати, якщо не підключився знову за 2 с (покриває подвійне монтування React StrictMode; AC US-04 дозволяє до 5 с). Якщо вийшов організатор, ним стає гравець, що приєднався найраніше (П-8); порожня кімната видаляється.
- ✅ У `PLAYING` після тих самих 2 с без повторного підключення кімната переходить в `ABANDONED`: гравець лишається в `players`, `finishedAt` заповнюється, усім підключеним приходить `room_state`, після чого будь-яка ігрова дія отримує `WRONG_PHASE`. Клієнт показує «Партію перервано» (G-2). У `FINISHED` і `ABANDONED` закриття з'єднання стан не змінює.

Кожне повідомлення — JSON з полем `type`.

### Клієнт → сервер

| `type` | Поля | Хто / коли | Історія |
|---|---|---|---|
| `start_game` | — | Організатор, `LOBBY`, 2–5 гравців | US-05 |
| `place_tile` | `x: int`, `y: int`, `rotation: 0\|90\|180\|270` | Активний гравець, фаза `PLACE_TILE` | US-10, US-12 |
| `place_meeple` | `segment: int` | Активний гравець, фаза `PLACE_MEEPLE` | US-13 |
| `skip_meeple` | — | Активний гравець, фаза `PLACE_MEEPLE` | US-14 |

Приклад:
```json
{ "type": "place_tile", "x": 1, "y": 0, "rotation": 90 }
```

Обертання до кладіння (US-10) і панорамування (US-08) — локальні дії клієнта, на сервер не надсилаються.

### Сервер → клієнт

**`room_state`** — повний знімок після кожної зміни, кожному гравцю.

```json
{
  "type": "room_state",
  "room": {
    "code": "K7QM2X",
    "status": "PLAYING",
    "hostPlayerId": "3f1c…",
    "youPlayerId": "b72e…",
    "players": [
      { "id": "3f1c…", "nickname": "Олег", "color": "RED", "seat": 0, "meeplesInSupply": 6, "score": 12 },
      { "id": "b72e…", "nickname": "Ірина", "color": "BLUE", "seat": 1, "meeplesInSupply": 7, "score": 4 }
    ],
    "game": {
      "phase": "PLACE_TILE",
      "turnNumber": 9,
      "currentPlayerId": "b72e…",
      "deckCount": 62,
      "currentTile": { "tileTypeId": "D" },
      "discardedTiles": [],
      "legalPlacements": [ { "x": 1, "y": 0, "rotations": [0, 180] } ],
      "legalMeepleSegments": [],
      "tiles": [ { "x": 0, "y": 0, "tileTypeId": "D", "rotation": 0 } ],
      "meeples": [ { "playerId": "3f1c…", "x": 0, "y": 0, "segment": 1 } ],
      "finalScores": null
    }
  }
}
```

| Поле | Історія | Правило |
|---|---|---|
| `players[].color`, `seat` | US-05 | `null` у `LOBBY` |
| `players[].meeplesInSupply` | US-16 | `7 − міпли на полі` |
| `players[].score` | US-20 | Сума `ScoreEntry` |
| `game` | US-06 | `null` у `LOBBY` |
| `game.currentPlayerId` | US-07 | |
| `game.turnNumber` | US-07 | Перший хід — 1; зростає, коли хід переходить до наступного гравця. Стартовий тайл має `turnNumber` 0 |
| `game.deckCount` | US-16 | Довжина стопки без поточного тайла. Одразу після `start_game` — 70: з 71 тайла один уже в першого гравця (AC US-05); так само, як 50 → 49 в AC US-09 |
| `game.currentTile` | US-09 | Бачать усі; `null` після кінця стопки і в `ABANDONED` |
| `game.discardedTiles` | US-09, US-21 | Тайли (`{ "tileTypeId" }`), які сервер автоматично скинув під час останнього витягування, бо їх неможливо покласти; інакше `[]`. Непорожній масив — сигнал клієнту показати «Тайл неможливо покласти, береться новий». Очищується на наступному витягуванні |
| `game.legalPlacements` | US-11 | Лише у фазі `PLACE_TILE` статусу `PLAYING`, для всіх 4 обертань; інакше `[]` |
| `game.legalMeepleSegments` | US-13 | Лише у фазі `PLACE_MEEPLE`, сегменти щойно покладеного тайла; інакше `[]`. Shortcut до US-13: завжди `[]` |
| `game.phase`, `game.currentPlayerId` | US-07 | У `FINISHED` і `ABANDONED` лишаються значеннями на момент завершення; UI спершу дивиться на `room.status` |
| `game.finalScores` | US-21…US-23 | Не `null` лише при `FINISHED` |

✅ **Кінець стопки (shortcut до US-21–US-23).** Якщо після `skip_meeple` стопка порожня або всі тайли, що лишилися, автоматично скинуто (AC US-21 «Останній тайл скинуто»), кімната переходить у `FINISHED`: `currentTile = null`, `deckCount = 0`, `finalScores` — рядок на кожного гравця з нулями в усіх категоріях і `winner: true` у всіх (нічия на 0, бо підрахунку ще немає). З US-18 і US-21–US-23 нулі замінить справжній підрахунок; формат не зміниться.

`finalScores`:
```json
[
  { "playerId": "3f1c…", "road": 14, "city": 30, "monastery": 9, "field": 15, "total": 68, "winner": true }
]
```
Відсортовано за `total` спадно; при нічиїй `winner: true` у всіх лідерів. `road + city + monastery + field = total`.

**`error`** — помилка дії, лише автору.
```json
{ "type": "error", "error": { "code": "NOT_YOUR_TURN", "message": "Зараз хід іншого гравця" } }
```

### Чого в знімку немає (заборона 9)

`token` будь-якого гравця, порядок `deck`, внутрішні `ScoreEntry` (лише суми).

---

## 4. Відповідність історіям MVP

| Історія | Контракт |
|---|---|
| US-01 | `POST /api/rooms` |
| US-02 | Клієнт: посилання `/r/{roomCode}`, API не потрібен |
| US-03 | `GET /api/rooms/{code}`, `POST /api/rooms/{code}/players` |
| US-04 | WS `room_state.players` |
| US-05 | WS `start_game` |
| US-06, US-07, US-15 | WS `room_state.game` |
| US-08, US-10 | Лише клієнт |
| US-06 | `GET /api/tiles`, `room_state.game.tiles` |
| US-09, US-16 | `currentTile`, `discardedTiles`, `deckCount`, `meeplesInSupply` |
| US-11, US-12 | `legalPlacements`, `place_tile` |
| US-13, US-14 | `legalMeepleSegments`, `place_meeple`, `skip_meeple` |
| US-18, US-19, US-20 | `players[].score`, зменшення `meeples` |
| US-21, US-22, US-23 | `finalScores` |
