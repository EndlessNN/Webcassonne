# Тестова стратегія Webcassonne

> Стек тестів — [ADR-0006](adr/0006-testing-stack.md). Очікувані значення — з [requirements.md §4](requirements.md#4-критерії-приймання-acceptance-criteria), [api.md](api.md), [tiles.md](tiles.md). Промпт: [prompts/6.1](../prompts/6.1_testing_strategy.md).

## 1. Межі

| Що | Як перевіряємо | Чому |
|---|---|---|
| Правила гри: стикування, допустимі позиції, скидання тайла, стопка | Автотести `game` | Найризикованіша логіка (README, принцип 1); чисті функції без мережі |
| Стан кімнати й ходу: старт, черга, фази, помилки, `ABANDONED`, `FINISHED` | Автотести `room` | Тут поєднуються гравці й `game`; порядок перевірок з `api.md` §1 |
| Контракт REST і WebSocket: статуси, коди й тексти помилок, поля `room_state` | Автотести `httpapi` | Клієнт довіряє лише серверу; розбіжність з `api.md` ламає UI |
| Чисті функції клієнта: `nextRotation`, `highlightedCells`, `tileShapes` | Vitest у `web/src/game` | Логіку US-10, US-11 видно без браузера |
| Вигляд поля й тайлів (мури, щити), панорамування мишею, буфер обміну, клавіша R, два реальні вікна браузера | Ручний smoke-чекліст ([testing-report.md](testing-report.md)) | Залежить від браузера й очей; E2E не робимо (ADR-0006) |
| Відновлення після відключення, мобільна верстка, навантаження | Не перевіряємо | Поза MVP (requirements.md §2) |

## 2. Рівні й інструменти

| Рівень | Пакет / тека | Що перевіряє | Інструмент |
|---|---|---|---|
| Модульний | `server/internal/game` | `CheckPlacement`, `LegalPlacements`, `NewDeck`, `New`, `PlaceTile`, `SkipMeeple`, `ValidateCatalog` | `testing`, табличні тести |
| Сервісний | `server/internal/room` | `StartGame`, `PlaceTile`, `SkipMeeple`, `PlaceMeeple`, `Disconnect`, знімок (`Snapshot`) | `testing`, справжні `store.Memory` і `game` |
| Інтеграційний | `server/internal/httpapi` | REST і WS через справжній сервер: розбір повідомлень, `error` лише автору, `room_state` усім, `GET /api/tiles` | `net/http/httptest`, `github.com/coder/websocket` |
| Модульний (клієнт) | `web/src/game` | Обертання, підсвітка, геометрія тайла | Vitest, середовище `node` |

**Правило:** логіку правил тестуємо в `game`, а не через WebSocket. `httpapi` перевіряє лише контракт і доставку (по одному сценарію на код помилки, а не всі варіанти стикування).

## 3. Історії → рівні

Стан після аудиту 6.3. Повна таблиця «сценарій → тест» — [testing-report.md §1](testing-report.md#1-покриті-історії-та-сценарії), ручні кроки — там само, §2.

| Історія | Рівень | Автотести | Лише вручну |
|---|---|---|---|
| US-01 | room, httpapi | `TestUS01_CreateRoom`, `TestCreateAndJoin_StatusCodes` | Форма на головній |
| US-02 | — | — | Копіювання, запасне поле |
| US-03 | room, httpapi | `TestUS03_Join`, `TestCreateAndJoin_StatusCodes`, `TestPreview`, `TestSocket_InvalidTokenRejectedBeforeUpgrade` | Сторінка запрошення |
| US-04 | room, httpapi | `TestUS04_*` (зокрема `TestUS04_LeaveGrace`), `TestHostLeavesLobby_*`, `TestLastPlayerLeaves_*`, `TestReconnect_*`, `TestSocket_LobbyUpdates` | Час появи й зникнення в лобі |
| US-05 | game, room, httpapi | `TestUS05_NewGame`, `TestUS05_StartGame`, `TestUS05_TurnOrderIsShuffled`, `TestSocket_US05_*` | Кнопка, підказка, напис для не організатора |
| US-06 | httpapi, web | `TestUS06_TileCatalog`, `TestSocket_US05_StartGame` (`tiles`), `tileShapes.test.ts` | Вигляд поля, мурів і щитів |
| US-07 | game, room, httpapi | `TestUS07_NotYourTurn` (game, room), `TestSocket_US07_US12_GameErrors` | «Ваш хід» / «Хід: …» |
| US-08 | — | — | Перетягування |
| US-09 | game, room | `TestUS09_DrawTile`, `TestUS09_DrawAfterTurn` | Повідомлення про скидання |
| US-10 | web | `rotation.test.ts` | Кнопка, клавіша R |
| US-11 | game, web | `TestUS11_LegalPlacements`, `highlight.test.ts` | Вигляд підсвітки |
| US-12 | game, room, httpapi | `TestUS12_PlaceTile` (game — тест першим; room), `TestUS12_PlaceTileTwoNeighbors`, `TestUS12_PlaceTileRejected`, `TestSocket_US07_US12_GameErrors`, `TestSocket_MessageFormat` | Клік, підказка |
| US-13 (shortcut) | room, httpapi | `TestUS13_PlaceMeepleRejectedUntilImplemented`, `TestSocket_US07_US12_GameErrors` | — |
| US-14 | room, httpapi | `TestUS14_SkipMeeple`, `TestSocket_US15_US21_FullGame` | Кнопка «Без міпла» |
| US-15 | room, httpapi | `TestSocket_US15_US21_FullGame` (однаковий знімок у обох гравців 71 хід), `TestUS15_SnapshotIsIndependentCopy` | ≤ 2 с у двох вікнах |
| US-16 | game, room, httpapi | `TestUS09_DrawTile` (50 → 49), `TestUS14_SkipMeeple`, `TestSocket_US15_US21_FullGame` (`deckCount`, `meeplesInSupply`) | Лічильники на екрані |
| US-21 (shortcut), `FINISHED` | game, room, httpapi | `TestUS21_EndOfDeck` (game, room), `TestSocket_US15_US21_FullGame` | Екран «Партію завершено» (71 хід — не проходимо) |
| `ABANDONED` | room, httpapi | `TestAbandoned_DisconnectDuringGame`, `TestUS04_LeaveGrace`, `TestSocket_Abandoned` | «Партію перервано» |
| US-13 (міпли), US-18…US-23 | game | Наступний інкремент, **тести першими** (AGENTS.md §6) | — |

## 4. Організація

| Що | Правило |
|---|---|
| Файли | Go: `*_test.go` у тому ж пакеті, поруч з кодом (`game/placement_test.go`). Web: `web/src/**/*.test.ts` поруч з функцією (`game/rotation.test.ts`) |
| Імена | `TestUSxx_<Що>` (`TestUS12_PlaceTile`); підтести — назва Gherkin-сценарію українською (`"клітинка без сусідів"`). Негативні випадки з `api.md` без сценарію в AC — назва з кодом помилки |
| Хелпери | У `helpers_test.go` свого пакета (`newTestService`, `newTestServer`, `newTestGame`); між пакетами не діляться |
| Vitest | Імпорт `describe`/`it`/`expect` явно з `'vitest'` (без глобалів: `tsconfig.app.json` має `types: ["vite/client"]`) |

## 5. Тестові дані та ізоляція

| Правило | Як у нашому коді |
|---|---|
| Свій стан у кожному тесті | Кожен тест створює `store.NewMemory()`, `room.NewService`, `httptest.NewServer` з `t.Cleanup(srv.Close)`; глобальних змінних у тестах немає |
| Без очікування виходу з лобі | `LeaveGrace = 0` (`newTestService`, `newTestServer`); сам таймер перевіряє `TestUS04_LeaveGrace` з паузою 50 мс і `select` з `time.After` |
| Випадковість | `game` отримує `*rand.Rand` параметром — у тестах `rand.New(rand.NewPCG(1, 2))`; у `room` — поле `rng` (тести в тому ж пакеті). Перевіряємо властивості: місця — перестановка 0…n−1, кольори унікальні за П-9, у стопці 71 тайл з кількостями з `tiles.md`; не конкретний порядок |
| Поле під сценарій | `model.Game` будується хелпером з типів `tiles.md` (стартовий D у (0, 0) + потрібні тайли й `Deck`), а не проходом через випадкову стопку |
| Без зависань | Без `time.Sleep`. Канали — `select` з `time.After`; WebSocket — `context.WithTimeout` (як у `TestSocket_LobbyUpdates`) |
| Джерело очікувань | Літерали з AC, `api.md`, `tiles.md` з коментарем «звідки»; не з виводу програми і не тією ж функцією (`edgeTypes` не рахує очікуване для `CheckPlacement`) |

## 6. Запуск

| Команда | Що робить |
|---|---|
| `npm test` (з кореня) | Go-тести + Vitest, ненульовий код при будь-якому падінні |
| `npm run test:go` / `go -C server test ./...` | Лише Go |
| `npm run test:web` / `npm --prefix web test` | Лише Vitest |
| `go -C server test -race ./...` | Гонки даних. Linux/macOS; на Windows без gcc — без `-race` (prompts/5.2). Незалежність знімка від стану кімнати `-race` сам не ловить (мутація без `cloneGame` проходила) — її перевіряє `TestUS15_SnapshotIsIndependentCopy` |
| `go -C server test -cover ./...` | Орієнтир, не ціль. Зараз (після аудиту 6.3): `game` 77.8 %, `room` 86.1 %, `httpapi` 90.5 % |

## 7. Контроль якості тестів: «зламай код навмисно»

| Крок | Як |
|---|---|
| Що ламаємо | Умову, від якої залежить сценарій: дзеркальні точки в `sidesMatch` (`2-k` → `k`), `neighbors == 0` у `CheckPlacement`, `seat != g.CurrentSeat` у `PlaceTile`, `len(r.Players) < model.MinPlayers` у `StartGame`, `g.Discarded` у `draw`, фільтр за `rotation` у `highlightedCells` |
| Як | Одна мутація за раз (`==` → `!=`, `<` → `<=`, прибрати перевірку) → `npm run test:go` або `test:web` → записати, який тест упав |
| Повернення | `git checkout -- <файл>`; після всіх мутацій `git diff` коду продукту порожній |
| Якщо не впав жоден тест | Тест слабкий: посилити й повторити мутацію. Журнал — `testing-report.md` → «Журнал: зламай код навмисно» |
