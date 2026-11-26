# Звіт з тестування

> Розділи звіту (покриття, ручний чекліст, баги, рефлексія) заповнюються в [prompts/6.3](../prompts/6.3_qa_audit_and_debugging.md). Поки тут лише журнали.

## Журнал: знайдені баги

| Сценарій | Очікувано | Фактично | Ймовірна причина (файл:рядок) |
|---|---|---|---|
| US-05 «Успішний старт»: «у стопці 71 тайл» | За AC US-05 — лічильник стопки 71 | `deckCount` = 70: перший гравець одразу тягне тайл, а `api.md` (`deckCount`) і AC US-09 (50 → 49) рахують стопку без поточного тайла | **Прогалина вимог**, не баг коду: AC US-05 суперечив `docs/api.md` і AC US-09 (`docs/requirements.md:299`). Виправлено формулювання AC: 71 = 70 у стопці + 1 у гравця, лічильник показує 70. Код не змінювався |
| Ручна звірка `docs/tiles-preview.html` з коробкою (6.0) | Номер сегмента стоїть на своєму сегменті; мури міста — на кожному тайлі з містом, по межі міста з полем | Номери не на своїх сегментах: поле на монастирі в A і B, номери 0 і 2 накладаються в D. Мури малювались лише на частині тайлів і вздовж краю тайла | **Недоробка превʼю**, не каталогу (`docs/tiles-preview.html`; дані `tiles.md` правильні). Виправлено в 6.0 (промпт 3); те саме правило мурів застосовано до поля гри |
| Перевірка звіту 6.0: межі інкременту | Інкремент без підрахунку очок: `finalScores` з нулями (shortcut, `api.md`); US-20, US-23 — пізніше, тести першими (AGENTS.md §6) | ШІ реалізував підрахунок рахунку, розбивку за категоріями й визначення переможця (US-20, US-23) без тестів першими | **Вихід за скоуп** у `server/internal/game/score.go`, всупереч AGENTS.md. Прибрано, лишено `ZeroFinalScores` як shortcut |

## Журнал: матриця покриття

Історії US-05, US-07, US-12 (з US-09 і US-11 у `game`), на клієнті US-10 і US-11. Позначення тайлів і сторін — `docs/tiles.md`; сторона записана трьома точками (`CCC` — місто, `FFF` — поле, `FRF` — поле-дорога-поле).
Ручний розрахунок для лише стартового D у (0, 0), обертання 0: N = `CCC`, E = `FRF`, S = `FFF`, W = `FRF`. Обертання за годинниковою (П-6): при 90° західна сторона стає північною.

| Сценарій | Рівень | Назва тесту | Очікуване значення | Звідки значення |
|---|---|---|---|---|
| US-05 «Успішний старт»: стартовий тайл, 70 у стопці + 1 у гравця | game | `TestUS05_NewGame/успішний_старт` | Тайли = {(0,0): D, 0}; `len(Deck)` = 70; `CurrentTile` ≠ ""; місце 0, фаза `PLACE_TILE`, хід 1; Deck + CurrentTile = A2 B4 C1 D3 E5 F2 G1 H3 I2 J3 K3 L3 M2 N3 O2 P3 Q1 R3 S2 T1 U8 V9 W4 X1 (=71) | AC US-05; `tiles.md` (D: 4 − 1 стартовий); `api.md` `deckCount`, `turnNumber`. Скидань немає: будь-яка сторона тайла (`CCC`/`FFF`/`FRF`) збігається з N, S або E стартового D |
| US-05 «Успішний старт» (3 гравці) | room | `TestUS05_StartGame/успішний_старт` | `PLAYING`; місця — перестановка {0,1,2}; кольори за місцем RED, BLUE, GREEN; у кожного 7 міплів; `deckCount` 70; поточний гравець — місце 0; кожна підписка отримує знімок `PLAYING` | AC US-05, П-9, `api.md` |
| US-05 межа: 2 гравці | room | `TestUS05_StartGame/2_гравці_—_мінімум` | Старт успішний, кольори RED, BLUE | `api.md` `start_game` «2–5 гравців» |
| US-05 межа: 5 гравців | room | `TestUS05_StartGame/5_гравців_—_максимум` | Кольори RED, BLUE, GREEN, YELLOW, BLACK за місцями 0…4 | П-9 |
| US-05 «Недостатньо гравців» — кнопка неактивна з підказкою | вручну | — | Підказка «Потрібно щонайменше 2 гравці» | AC US-05 |
| US-05 «Недостатньо гравців» — сервер | room | `TestUS05_StartGame/NOT_ENOUGH_PLAYERS` | `NOT_ENOUGH_PLAYERS` «Потрібно щонайменше 2 гравці»; `LOBBY`, `Game` = nil, місце −1, колір порожній | `api.md` §1 |
| US-05 «Не організатор» — немає кнопки, напис | вручну | — | «Очікуємо, поки організатор почне гру» | AC US-05 |
| US-05 «Не організатор» — сервер | room, httpapi | `TestUS05_StartGame/NOT_HOST`, `TestSocket_US05_StartGame` | `NOT_HOST` «Почати гру може лише організатор» лише автору; `LOBBY`, `Game` = nil | `api.md` §1 |
| US-05 повторний старт (без сценарію) | room, httpapi | `TestUS05_StartGame/GAME_ALREADY_STARTED`, `TestSocket_US05_StartGame` | `GAME_ALREADY_STARTED` «Партія вже почалася» і від організатора, і від не організатора (статус перевіряється раніше за автора); стопка, тайли, місця не змінилися | `api.md` §1, порядок перевірок |
| US-05 одна кнопка — знімок усім; без токенів | httpapi | `TestSocket_US05_StartGame` | Обидва з'єднання: `PLAYING`, `deckCount` 70, `tiles` = [{0,0,D,0}], однаковий `currentPlayerId` = гравець з `seat` 0; у JSON немає токенів | AC US-05, `api.md` «Чого в знімку немає» |
| US-05 `NOT_ENOUGH_PLAYERS` через WS | httpapi | `TestSocket_US05_NotEnoughPlayers` | `error` `NOT_ENOUGH_PLAYERS` «Потрібно щонайменше 2 гравці» | `api.md` §1 |
| US-07 «Мій хід» — напис «Ваш хід» / «Хід: Ірина», виділення | вручну | — | — | AC US-07 |
| US-07 «Мій хід» — дані для напису | httpapi | `TestSocket_US05_StartGame` | `currentPlayerId` однаковий в обох знімках і дорівнює гравцю з `seat` 0 | `api.md` `currentPlayerId` |
| US-07 «Спроба діяти не у свій хід» | game | `TestUS07_NotYourTurn/покласти_тайл_не_у_свій_хід` | `ErrNotYourTurn`; тайлів 1, стопка, місце, фаза, хід не змінилися | AC US-07 |
| US-07 порядок: чужий хід раніше за зайняту клітинку | game, room | `TestUS07_NotYourTurn/чужий_хід_на_зайняту_клітинку`, `TestUS07_NotYourTurn` (room) | `NOT_YOUR_TURN`, а не `ILLEGAL_PLACEMENT` | `api.md` порядок перевірок |
| US-07 «Спроба діяти не у свій хід» — сервер | room | `TestUS07_NotYourTurn` | `NOT_YOUR_TURN` «Зараз хід іншого гравця» для `place_tile` і `skip_meeple`; тайлів 1, `deckCount` 70, поточний гравець, фаза, `currentTile` не змінилися | AC US-07, `api.md` §1 |
| US-07 помилка лише автору | httpapi | `TestSocket_US07_US12_GameErrors` | Неактивний отримує `NOT_YOUR_TURN` «Зараз хід іншого гравця»; наступне повідомлення активного — його власний результат, а не чужа помилка | `api.md` «WS-помилки надсилаються лише автору» |
| US-09 «Отримання тайла»: стопка 50 → 49 | game | `TestUS09_DrawTile/отримання_тайла_на_початку_ходу` | Стопка 50 (верх — U, далі B), після `SkipMeeple` попереднього гравця: `CurrentTile` = U, `len(Deck)` = 49, місце 1, фаза `PLACE_TILE`, хід +1, `Discarded` порожній | AC US-09 |
| US-09 «Тайл, який неможливо покласти» | game | `TestUS09_DrawTile/тайл_який_неможливо_покласти` | Поле: D (0,0), E 180° у (0,−1). Верх стопки C → скинуто: `Discarded` = [C], `CurrentTile` = U, стопка −2 | AC US-09; розрахунок: C — місто на всіх сторонах; відкриті сторони поля лише `FFF` (E 180° на N, E, W; D на S) і `FRF` (D на E, W) |
| US-09 скинуті очищуються на наступному витягуванні (без сценарію) | game | `TestUS09_DrawTile/скинуті_тайли_очищуються` | `Discarded` = [] після наступного витягування | `api.md` `discardedTiles` |
| US-09 / US-14 помилки `SkipMeeple` (без сценарію) | game | `TestUS09_DrawTile/NOT_YOUR_TURN:_skip_meeple_від_неактивного_гравця`, `TestUS09_DrawTile/WRONG_PHASE:_skip_meeple_у_фазі_PLACE_TILE` | `ErrNotYourTurn` / `ErrWrongPhase`, стан не змінився | `api.md` §1 |
| US-09 скидання через сервер | room | `TestUS09_DrawAfterTurn` | Після E 180° у (0,−1) і `skip_meeple`: `discardedTiles` = [C], `currentTile` = U, `deckCount` = 2 (було 4), хід 2, поточний — інший гравець | AC US-09, `api.md` |
| US-09 повідомлення «Тайл неможливо покласти, береться новий» | вручну | — | — | AC US-09 |
| US-10 «Поворот на 90°» | web | `nextRotation › поворот на 90°` | 0 → 90 | AC US-10, П-6 |
| US-10 «Повний оберт» | web | `nextRotation › повний оберт`, `nextRotation › 4 повороти з N повертають у ту саму орієнтацію` | 4 натискання: 90, 180, 270, 0; з будь-якого обертання 4 рази → те саме | AC US-10 |
| US-10 кнопка «Повернути», клавіша R | вручну | — | — | AC US-10 |
| US-11 «Підсвітка позицій» — правила | game | `TestUS11_LegalPlacements/тайл_E…`, `/тайл_U…`, `/тайл_C…` | E: (0,−1) [180], (0,1) [90 180 270]; U: (−1,0) [90 270], (1,0) [90 270], (0,1) [90 270]; C: (0,−1) [0 90 180 270] | Ручний розрахунок за `tiles.md` (див. вище) |
| US-11 «жодна клітинка, де сторони не збігаються» | game, web | ті самі + `/тайл,_який_неможливо_покласти`; `highlightedCells › жодна клітинка, де сторони не збігаються, не підсвічена` | Рівність множин позицій і обертань (порядок не в контракті); для C на полі D + E 180° — 0 позицій; U і E в обертанні 0 — [] | Розрахунок як для US-09 |
| US-11 «Підсвітка позицій» — клієнт | web | `highlightedCells › підсвітка позицій` | U, обертання 90 → (−1,0), (1,0), (0,1) | `legalPlacements` з розрахунку для U |
| US-11 «Оновлення після обертання» | web | `highlightedCells › оновлення після обертання` | U: 90 → 3 клітинки, 180 → 0; E: 180 → (0,−1), (0,1); 270 → (0,1) | AC US-11 |
| US-11 «Чужий хід» | web | `highlightedCells › чужий хід` | [] | AC US-11 |
| US-11 фаза `PLACE_MEEPLE` (без сценарію) | web | `highlightedCells › фаза PLACE_MEEPLE` | [] | `api.md` `legalPlacements`; коментар `highlight.ts` |
| US-11 вигляд підсвітки | вручну | — | — | AC US-11 |
| US-12 «Розміщення на допустиму позицію» | game | `TestUS12_PlaceTile/розміщення_на_допустиму_позицію_праворуч_від_стартового` (6.0) | U 90° у (1,0); фаза `PLACE_MEEPLE` | AC US-12 |
| US-12 «Недопустима позиція» | game | `TestUS12_PlaceTile/недопустима_позиція:_місто_проти_поля` (6.0) | `ErrEdgeMismatch`, стан не змінився | AC US-12 |
| US-12 «Клітинка без сусідів» | game | `TestUS12_PlaceTile/клітинка_без_сусідів`, `/лише_діагональний_сусід…` (6.0) | `ErrNoNeighbor` | AC US-12 |
| US-12 зайнята клітинка (без сценарію) | game | `TestUS12_PlaceTile/зайнята_клітинка` (6.0) | `ErrOccupied` | `api.md` |
| US-12 два сусіди, збігається лише один (без сценарію) | game | `TestUS12_PlaceTileTwoNeighbors` | Поле D (0,0), E 180° (0,−1), U 90° (1,0). E 0° у (1,−1) — ок; E 180° — `ErrEdgeMismatch` (W `FFF` ↔ E-сторона E 180° `FFF` збігається, S `CCC` ↔ N U 90° `FFF` ні) | `api.md` «хоча б одна суміжна сторона»; розрахунок |
| US-12 обертання поза {0,90,180,270}; фаза `PLACE_MEEPLE` (без сценарію) | game | `TestUS12_PlaceTileRejected/VALIDATION_ERROR`, `/WRONG_PHASE` | `ErrInvalidRotation` / `ErrWrongPhase`, стан не змінився | `api.md` §1 |
| US-12 через сервер: успіх і тексти помилок | room | `TestUS12_PlaceTile` (room) | Успіх: `PLACE_MEEPLE`, тайлів 2, `legalPlacements` порожні, хід і стопка ті самі. Помилки: «Сторони тайлів мають збігатися», «Тайл має межувати з іншим тайлом», «Клітинка вже зайнята» (`ILLEGAL_PLACEMENT`), `VALIDATION_ERROR`, `WRONG_PHASE` «Ця дія зараз недоступна»; стан не змінився | AC US-12, `api.md` §1 |
| US-12 тексти WS і знімок усім | httpapi | `TestSocket_US07_US12_GameErrors` | `ILLEGAL_PLACEMENT` «Клітинка вже зайнята»; `VALIDATION_ERROR` «Неправильне повідомлення» (rotation 45) і «Невідомий тип повідомлення»; після допустимого `place_tile` обидва отримують `PLACE_MEEPLE`, `tiles` = 2, `legalPlacements` = [] | `api.md` §1, §3 |
| US-12 клік, підказка на екрані | вручну | — | — | AC US-12 |

Рівні, яких немає в матриці: US-12 «Розміщення…» на рівні `game` уже покрито тестом першим з 6.0 (`TestUS12_PlaceTile`, 8 підтестів) — не дублювали, доповнили двома сусідами, обертанням поза множиною і фазою.

### Прогалини вимог

| Що | Чому прогалина | Як вчинили |
|---|---|---|
| Порядок `legalPlacements` і `rotations` | `api.md` не задає порядок; «за y, потім за x» є лише в коментарі `game/placement.go:62` | Тести порівнюють множини (game) і сортують клітинки (web), порядок не перевіряють |
| AC US-11 «Оновлення після обертання: підсвічено 3 позиції» | Не сказано, який тайл і яке поле | Узято U поруч з одним стартовим D: 90° → 3 клітинки, 180° → 0 (ручний розрахунок вище) |
| AC US-05 «порядок ходів визначено випадково» | Випадковість не перевіряється одним прогоном | Перевіряємо властивість: місця — перестановка 0…n−1, колір за місцем (П-9) |
| Дзеркальні точки стикування (`N0↔S2`, data-model.md) | За `tiles.md` кожна сторона — `CCC`, `FFF` або `FRF` (паліндром), тож правило дзеркала не відрізнити від прямого порівняння жодним тайлом каталогу | Мутація M1 — еквівалентна; див. журнал нижче |

## Журнал: зламай код навмисно

Одна мутація за раз → `go test` пакета або `vitest run src/game` → `git checkout -- <файл>`. Після всіх мутацій `git diff` коду продукту порожній (змінено лише тести й цей звіт).

| Файл:рядок | Мутація | Тест, що впав / «вижила» → що посилено |
|---|---|---|
| `server/internal/game/edges.go:59` | `b[opposite*3+2-k]` → `b[opposite*3+k]` (без дзеркала) | **Вижила.** Еквівалентна для каталогу: усі сторони — паліндроми (`CCC`/`FFF`/`FRF`), тайла, який би її вбив, у `tiles.md` немає; вигадувати тайл заборонено (AGENTS.md §7). Перевірили сусідні умови стикування — M1b, M1c |
| `server/internal/game/edges.go:57` | `(side + 2) % 4` → `side` (порівняння не з протилежною стороною) | `TestUS11_LegalPlacements` (4 підтести), `TestUS12_PlaceTile/недопустима_позиція…`, `/місто_до_міста…`, `TestUS09_DrawTile/тайл,_який_неможливо_покласти` |
| `server/internal/game/edges.go:48` | `(i+shift)%12` → `(i+12-shift)%12` (обертання проти годинникової) | `TestUS12_PlaceTileTwoNeighbors/місто_у_бік_без_сусіда`, `/місто_проти_поля_з_другого_сусіда`. **Знахідка для рефлексії:** тести US-12 з 6.0 не ловили обертання проти годинникової стрілки (у них 90° і 270° давали однаковий результат); виявлено мутацією, закрито `TestUS12_PlaceTileTwoNeighbors` |
| `server/internal/game/placement.go:55` | `neighbors == 0` → `neighbors < 0` | `TestUS12_PlaceTile/клітинка_без_сусідів`, `/лише_діагональний_сусід_—_це_не_сусід` |
| `server/internal/game/placement.go:36` | перевірку зайнятої клітинки прибрано | `TestUS12_PlaceTile/зайнята_клітинка` |
| `server/internal/game/game.go:78` | `seat != g.CurrentSeat` → `false` у `PlaceTile` | `TestUS07_NotYourTurn/покласти_тайл_не_у_свій_хід`, `/чужий_хід_на_зайняту_клітинку` |
| `server/internal/game/game.go:81` | перевірку фази `PLACE_TILE` прибрано | `TestUS12_PlaceTileRejected/WRONG_PHASE:_другий_тайл_у_фазі_PLACE_MEEPLE` |
| `server/internal/game/game.go:103` | `seat != g.CurrentSeat` → `false` у `SkipMeeple` | `TestUS09_DrawTile/NOT_YOUR_TURN:_skip_meeple_від_неактивного_гравця` |
| `server/internal/game/game.go:112` | `(seat + 1) % players` → `seat % players` (хід не переходить) | `TestUS09_DrawTile/отримання_тайла_на_початку_ходу`, `/тайл,_який_неможливо_покласти` |
| `server/internal/game/game.go:122` | `g.Discarded = nil` прибрано | `TestUS09_DrawTile/скинуті_тайли_очищуються_на_наступному_витягуванні` |
| `server/internal/game/game.go:126` | `canPlaceAnywhere(...)` → `true` (без скидання) | `TestUS09_DrawTile/тайл,_який_неможливо_покласти` |
| `server/internal/room/game.go:54` | `len(r.Players) < MinPlayers` → `<=` | `TestUS05_StartGame/2_гравці_—_мінімум` (і всі тести з партією на 2 гравців) |
| `server/internal/room/game.go:52` | перевірку організатора прибрано | `TestUS05_StartGame/NOT_HOST` |
| `server/internal/room/game.go:50` | перевірку статусу `LOBBY` прибрано | `TestUS05_StartGame/GAME_ALREADY_STARTED` |
| `server/internal/room/game.go:63` | `colorsBySeat[seat]` → `colorsBySeat[i]` (колір за порядком входу) | `TestUS05_StartGame/успішний_старт`, `/2_гравці_—_мінімум`, `/5_гравців_—_максимум` |
| `server/internal/room/game.go:62` | `Seat = seat` → `Seat = 0` (не перестановка) | `TestUS05_StartGame/успішний_старт`, `/2_гравці…`, `/5_гравців…`, `TestUS07_NotYourTurn`, `TestUS12_PlaceTile`, `TestUS09_DrawAfterTurn` |
| `server/internal/room/game.go:173` | `legalPlacements` у будь-якій фазі | `TestUS12_PlaceTile/розміщення_на_допустиму_позицію` |
| `server/internal/room/game.go:148` | `ErrEdgeMismatch` → `ErrNoNeighbor` (не той текст) | `TestUS12_PlaceTile/недопустима_позиція` |
| `web/src/game/highlight.ts:12` | фільтр за `rotation` прибрано | `highlightedCells › жодна клітинка…`, `› оновлення після обертання` |
| `web/src/game/highlight.ts:11` | перевірку активного гравця прибрано | `highlightedCells › чужий хід` |
| `web/src/game/highlight.ts:11` | перевірку фази `PLACE_TILE` прибрано | `highlightedCells › фаза PLACE_MEEPLE` |
| `web/src/game/rotation.ts:13` | `270 → 0` замінено на `270 → 270` | `nextRotation › повний оберт`, `› 4 повороти з 0/90/180…` |
| `web/src/game/rotation.ts:7` | `0 → 90` замінено на `0 → 270` (проти годинникової) | `nextRotation › поворот на 90°`, `› повний оберт` |
