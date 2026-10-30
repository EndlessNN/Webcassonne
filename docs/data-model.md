# Модель даних

> Сховище — пам'ять процесу ([ADR-0004](adr/0004-in-memory-storage.md)). Нижче — логічна схема: «таблиці», типи полів, ключі й обмеження, які гарантує код `server/internal/store`.
> Реалізація структур: [`server/internal/model/model.go`](../server/internal/model/model.go). Зміна схеми — спершу тут, потім у коді, в одному коміті.
> Промпт: [prompts/4.3_technical_contracts.md](../prompts/4.3_technical_contracts.md).

## Діаграма зв'язків

```text
Room 1 ──── 1..5 Player
  │
  └── 0..1 Game ─┬── 1..72  PlacedTile ──▶ TileType (статичний каталог)
                 ├── 0..35  Meeple ──▶ Player, PlacedTile
                 └── 0..N   ScoreEntry ──▶ Player
```

Типи: `string`, `int`, `bool`, `time` (UTC), `enum` (рядкові константи у ВЕРХНЬОМУ_РЕГІСТРІ).

---

## Room

| Поле | Тип | Обмеження | Примітка |
|---|---|---|---|
| `code` | string(6) | **PK**, унікальний серед усіх кімнат | Алфавіт `ABCDEFGHJKLMNPQRSTUVWXYZ23456789` |
| `status` | enum | `LOBBY` \| `PLAYING` \| `FINISHED` \| `ABANDONED` | Переходи — [technical-analysis.md §2](technical-analysis.md#room) |
| `hostPlayerId` | string | **FK → Player.id**, гравець цієї кімнати | Змінюється, якщо організатор вийшов з лобі |
| `createdAt` | time | not null | |
| `finishedAt` | time | null, поки статус не кінцевий | Для очищення через 1 годину |

## Player

| Поле | Тип | Обмеження | Примітка |
|---|---|---|---|
| `id` | string (UUID v4) | **PK** | Публічний |
| `roomCode` | string(6) | **FK → Room.code** | У Go — вкладеність `Room.Players` |
| `nickname` | string | 1–20 символів після `trim`; **UNIQUE (roomCode, lower(nickname))** | |
| `token` | string(32 hex) | **UNIQUE** глобально; секрет | Ніколи не надсилається іншим гравцям |
| `color` | enum \| null | `RED` \| `BLUE` \| `GREEN` \| `YELLOW` \| `BLACK`; **UNIQUE (roomCode, color)**; null у лобі | Призначається на старті |
| `seat` | int \| null | 0…4; **UNIQUE (roomCode, seat)**; null у лобі | Порядок ходу |
| `joinedAt` | time | not null | Визначає наступного організатора |

Обмеження кімнати: від 1 до 5 гравців; новий гравець — лише при `Room.status = LOBBY`.

## Game

Одна на кімнату (1:1), створюється при `start_game`.

| Поле | Тип | Обмеження | Примітка |
|---|---|---|---|
| `roomCode` | string(6) | **PK, FK → Room.code** | |
| `deck` | string[] | ID типів тайлів; спочатку 71 | Порядок секретний, клієнтам — лише довжина |
| `currentTileTypeId` | string \| null | FK → TileType.id | null після кінця стопки |
| `currentSeat` | int | 0…(кількість гравців − 1) | |
| `phase` | enum | `PLACE_TILE` \| `PLACE_MEEPLE` | `DRAW`/`SCORE` — миттєві кроки сервера, не зберігаються |
| `turnNumber` | int | ≥ 1 | Зростає на кожному ході |
| `lastPlacedX`, `lastPlacedY` | int \| null | | Куди можна поставити міпла |

## PlacedTile

| Поле | Тип | Обмеження | Примітка |
|---|---|---|---|
| `roomCode` | string(6) | **PK (roomCode, x, y)** | Одна клітинка — один тайл |
| `x`, `y` | int | | Стартовий тайл у (0, 0); `y` росте вниз |
| `tileTypeId` | string | **FK → TileType.id** | |
| `rotation` | int | `0` \| `90` \| `180` \| `270` (за годинниковою) | |
| `placedBy` | string \| null | FK → Player.id; null для стартового | |
| `turnNumber` | int | 0 для стартового | |

## Meeple

Зберігаються лише міпли на полі. У запасі гравця = `7 − кількість його Meeple`.

| Поле | Тип | Обмеження | Примітка |
|---|---|---|---|
| `roomCode`, `x`, `y`, `segment` | string, int, int, int | **PK (roomCode, x, y, segment)**; (x, y) → PlacedTile | `segment` — індекс у `TileType.segments` |
| `ownerId` | string | **FK → Player.id**; ≤ 7 на гравця | |
| `featureType` | enum | `ROAD` \| `CITY` \| `MONASTERY` \| `FIELD` | Копія типу сегмента для зручності |

Обмеження, яке перевіряє код (не ключ): на одному з'єднаному об'єкті (Feature) у момент постановки немає інших міплів.

## ScoreEntry

Рахунок не зберігається окремо: `score(player) = Σ points`. Розбивка для US-23 — групування за `category`.

| Поле | Тип | Обмеження | Примітка |
|---|---|---|---|
| `roomCode` | string(6) | FK → Room.code | |
| `playerId` | string | **FK → Player.id** | |
| `category` | enum | `ROAD` \| `CITY` \| `MONASTERY` \| `FIELD` | |
| `points` | int | > 0 | Нульові нарахування не записуються |
| `turnNumber` | int | ≥ 1 | |
| `final` | bool | | `true` — фінальний підрахунок (US-21, US-22) |

## Feature (обчислюється, не зберігається)

Об'єкт = множина з'єднаних сегментів кількох PlacedTile одного типу. Обчислюється з поля на вимогу (обхід графа / union-find) у пакеті `game`. Похідні значення: `tiles` (кількість унікальних тайлів), `pennants`, `complete`, `meeples`.

---

## TileType (статичний каталог)

Не змінюється під час роботи; живе в коді (`server/internal/game`) і документі `docs/tiles.md` (створюється перед US-09 і звіряється з фізичною копією). 72 тайли базової гри = набір типів з кількістю.

| Поле | Тип | Обмеження |
|---|---|---|
| `id` | string | **PK**, напр. `"A"`…`"X"` |
| `count` | int | Сума по всіх типах = 72, з них 1 — стартовий |
| `isStart` | bool | Рівно один тип має стартовий екземпляр |
| `segments` | Segment[] | Покривають усі 12 точок країв без перетинів |

**Точки країв.** Кожна сторона має 3 точки, за годинниковою стрілкою: `N0 N1 N2`, `E0 E1 E2`, `S0 S1 S2`, `W0 W1 W2`. Дорога займає центральну точку (`N1`), місто — всі три точки сторони, поле — решту.

| Поле Segment | Тип | Примітка |
|---|---|---|
| `type` | enum | `ROAD` \| `CITY` \| `FIELD` \| `MONASTERY` |
| `points` | string[] | Точки країв, напр. `["N0","N1","N2"]`; порожній для `MONASTERY` |
| `pennant` | bool | Лише для `CITY` |
| `adjacentCities` | int[] | Лише для `FIELD`: індекси CITY-сегментів цього тайла, з якими межує поле (US-22) |

**Стикування (US-12).** Сторони суміжних тайлів збігаються, якщо типи сегментів у дзеркальних точках однакові: `N0↔S2`, `N1↔S1`, `N2↔S0` (і так само `E↔W`), з урахуванням обертання.
