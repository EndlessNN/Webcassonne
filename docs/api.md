# API: REST + WebSocket

> Рішення: [ADR-0003](adr/0003-backend-stack.md), [ADR-0005](adr/0005-guest-identity-and-realtime.md). Сутності: [data-model.md](data-model.md).
> Типи клієнта: [`web/src/api/types.ts`](../web/src/api/types.ts). Зміна контракту — спершу тут, потім сервер і клієнт в одному коміті.
> Промпт: [prompts/4.3_technical_contracts.md](../prompts/4.3_technical_contracts.md).

Базовий шлях: `/api`. Формат: JSON, UTF-8, поля в `camelCase`. Статус реалізації: ✅ є в каркасі, ⏳ реалізується разом з історією.

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
| `VALIDATION_ERROR` | 400 | REST, WS | Нікнейм не 1–20 символів, неправильний JSON або поля | US-01, US-03 |
| `INVALID_TOKEN` | 401 | WS handshake | Токен відсутній або не належить кімнаті | — |
| `NOT_FOUND` | 404 | REST | Невідомий ендпоінт | — |
| `ROOM_NOT_FOUND` | 404 | REST, WS | Кімнати з таким кодом немає | US-03 |
| `ROOM_FULL` | 409 | REST | У кімнаті вже 5 гравців | US-03 |
| `GAME_ALREADY_STARTED` | 409 | REST | Статус кімнати не `LOBBY` | US-03 |
| `NICKNAME_TAKEN` | 409 | REST | Нікнейм уже є в кімнаті (без урахування регістру) | US-03 |
| `NOT_HOST` | — | WS | `start_game` надіслав не організатор | US-05 |
| `NOT_ENOUGH_PLAYERS` | — | WS | `start_game` при 1 гравці | US-05 |
| `NOT_YOUR_TURN` | — | WS | Дію надіслав не активний гравець | US-07 |
| `WRONG_PHASE` | — | WS | Дія не відповідає фазі (напр. `place_meeple` у `PLACE_TILE`) | US-12…US-14 |
| `ILLEGAL_PLACEMENT` | — | WS | Клітинка зайнята, без сусідів або сторони не збігаються | US-12 |
| `ILLEGAL_MEEPLE` | — | WS | Сегмент не з `legalMeepleSegments` | US-13 |
| `NO_MEEPLES_LEFT` | — | WS | У гравця 0 міплів у запасі | US-13 |
| `INTERNAL` | 500 | REST, WS | Непередбачена помилка сервера (деталі лише в лозі) | — |

WS-помилки надсилаються лише автору дії і не змінюють стан гри.

---

## 2. REST

### ✅ `GET /api/health` — перевірка здоров'я

`200 OK`
```json
{ "status": "ok", "storage": { "type": "memory", "rooms": 0 } }
```

### ⏳ `POST /api/rooms` — створити кімнату (US-01)

Запит:
```json
{ "nickname": "Олег" }
```

`201 Created` — творець стає гравцем і організатором:
```json
{ "roomCode": "K7QM2X", "playerId": "3f1c…", "token": "9a8b…" }
```

Помилки: `400 VALIDATION_ERROR`.

### ⏳ `GET /api/rooms/{code}` — попередній перегляд для запрошення (US-03)

Відкриває сторінку `/r/{code}`, щоб одразу показати «заповнена / вже почалася», ще до введення нікнейму.

`200 OK`
```json
{ "code": "K7QM2X", "status": "LOBBY", "playerCount": 3, "maxPlayers": 5 }
```

Помилки: `404 ROOM_NOT_FOUND`.

### ⏳ `POST /api/rooms/{code}/players` — приєднатися (US-03)

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

### ⏳ `GET /api/rooms/{code}/ws?token={token}` — канал кімнати (US-04…US-23)

- Успіх: `101 Switching Protocols`, одразу після підключення сервер надсилає `room_state`.
- Відмова до апгрейду: `401 INVALID_TOKEN`, `404 ROOM_NOT_FOUND` (REST-формат помилки).
- Закриття з'єднання: у `LOBBY` гравець видаляється з кімнати; у `PLAYING` кімната переходить в `ABANDONED`.

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
| `game.deckCount` | US-16 | Довжина стопки без поточного тайла |
| `game.currentTile` | US-09 | Бачать усі; `null` після кінця стопки |
| `game.legalPlacements` | US-11 | Лише у фазі `PLACE_TILE`, для всіх 4 обертань; інакше `[]` |
| `game.legalMeepleSegments` | US-13 | Лише у фазі `PLACE_MEEPLE`, сегменти щойно покладеного тайла; інакше `[]` |
| `game.finalScores` | US-21…US-23 | Не `null` лише при `FINISHED` |

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
| US-09, US-16 | `currentTile`, `deckCount`, `meeplesInSupply` |
| US-11, US-12 | `legalPlacements`, `place_tile` |
| US-13, US-14 | `legalMeepleSegments`, `place_meeple`, `skip_meeple` |
| US-18, US-19, US-20 | `players[].score`, зменшення `meeples` |
| US-21, US-22, US-23 | `finalScores` |
