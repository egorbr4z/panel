# План: VPN-панель `panel` (MVP)

## Context

Нужна self-hosted панель управления VPN/прокси-сервером уровня «**3x-ui по возможностям + подписки как в Marzban + анимации как у Remnawave, но без сложности нод/сквадов**». Один сервер (без многонодовости). Целевая ОС — **Debian 13**, железо очень скромное: **1 ГБ RAM / 1 vCPU** — экономия памяти является главным драйвером всех решений.

Репозиторий пустой (greenfield). Большинство запрошенных протоколов (VLESS/Reality, VMess, Trojan, Shadowsocks, Hysteria2, TUIC, AnyTLS, NaiveProxy, WireGuard/AmneziaWG) покрываются двумя ядрами — **Xray-core** (лучший для Reality) и **sing-box** (универсальное ядро). **Mieru** и **MTProto** в эти ядра не входят и вынесены в Phase 2 (закладываем точки расширения, но в MVP не реализуем).

### Решения (согласованы с пользователем)
- **Бэкенд:** Go (Gin + GORM) — один статический бинарник ~40-90 МБ, минимум RAM. Делает всё: API, отдача SPA, рендер подписок, супервизия ядер, учёт трафика, планировщик — без отдельных воркеров/Redis.
- **Фронтенд:** React + Vite + TypeScript, статическая сборка, встроенная в Go-бинарник через `go:embed`. Анимации — Framer Motion (эффект «ремнавейв» без рантайм-стоимости). TanStack Query + Zustand + Tailwind.
- **БД:** SQLite (WAL) по умолчанию, опция PostgreSQL через диалектор GORM.
- **Деплой:** Docker Compose + `install.sh` для Debian 13. Reverse proxy/TLS — Caddy (авто-ACME).

## Архитектура (ключевое)

- **Супервизия ядер** (`internal/core/`): общий `Process` (start/stop/restart с экспоненциальным backoff, crash-loop детект, кольцевой буфер логов в памяти — без логов на диск). Интерфейс `CoreManager` + реализации `xray`, `singbox`.
- **Связь с ядрами:**
  - Xray — gRPC: `HandlerService` (горячее добавление/удаление юзеров без рестарта), `StatsService` (`QueryStats(reset=true)` — атомарное чтение-обнуление счётчиков). Прото-стабы вендорятся под `internal/core/xray/pb`.
  - sing-box — Clash API / v2 stats по HTTP. Структурные изменения — «холодный путь» с дебаунсом ~500мс.
- **Учёт трафика на 1 vCPU** (`internal/scheduler/`): один таймер-поллер (10-30с) → накопитель в памяти (`map[userID]`) → батчевый flush в SQLite раз в ~60с + почасовые бакеты в `traffic_log`. В том же тике — энфорсмент квот/срока (disable + `RemoveUser`), сброс циклов (no_reset/day/week/month).

## Структура репозитория

```
panel/
├── cmd/vpanel/main.go            # сборка: config→DB→services→http→supervisor
├── internal/
│   ├── config/ database/ models/ repository/
│   ├── service/{auth,user,inbound,subscription,traffic,system}/
│   ├── core/{manager.go,process.go, xray/{config,api,pb}, singbox/{config,api}}
│   ├── api/{router.go, middleware/, handler/, dto/}
│   ├── scheduler/                # поллер, энфорсмент, сбросы
│   ├── subscription/             # рендереры base64/clash/singbox/template
│   └── web/                      # go:embed dist
├── frontend/                     # React+Vite → собирается в internal/web/dist
├── deploy/{docker-compose.yml,Caddyfile,Dockerfile}
├── scripts/{install.sh,gen-proto.sh}
└── docs/
```

## Модель данных (GORM, портируемая SQLite/Postgres)

- **admins**: username, password_hash (argon2id), role (`sudo`|`admin`), is_active, last_login_at.
- **users**: username, uuid, password, subscription_token (uniq), status, data_limit, used_up/down, reset_strategy, last_reset_at, expire_at, online_at, note, admin_id.
- **inbounds**: tag, core (`xray`|`singbox`), protocol, listen, port, network, security, settings_json, stream_json (TEXT/JSON-сериализатор), enabled, sort_order.
- **user_inbounds** (M:N), **hosts** (SNI/домен-оверрайды для подписок), **traffic_log** (почасовые бакеты, прун 30 дней), **settings** (k/v), **subscription_templates** (Go-template по target).

## Подписки (Marzban-style)

Один URL на юзера: `/sub/{token}`. Контент-негоциация по `?format=` / User-Agent + path-варианты `/sub/{token}/{clash|singbox|v2ray}`. Рендереры: **base64 v2ray** (с заголовком `Subscription-Userinfo: upload/download/total/expire`), **Clash/Clash.Meta YAML**, **sing-box JSON**, **кастомные шаблоны** (`text/template` над `SubContext`). Плюс анимированная страница `/sub/{token}/info` (кольцо использования, отсчёт срока, кнопки deep-link + QR). Кэш рендера в памяти по `(token,format,user.updated_at)`.

## Auth & API

- JWT (HS256, access ~15м + refresh), argon2id с низкими параметрами под 1 vCPU. Роли `sudo`/`admin`, скоуп данных по `admin_id`. Middleware: Recover→Log→CORS→RateLimit→JWT. Подписки — opaque-токен, не JWT.
- REST `/api`: auth, admins(sudo), users (+reset-traffic/revoke-sub/enable/disable/usage), inbounds (+hosts, reality-keys генератор), cores (status/restart/logs/config), system, settings, sub-templates. Публичные `/sub/...` вне `/api`.

## Фронтенд

Страницы: Login, Dashboard (RAM/CPU/online/график, статус ядер), Users (таблица+drawer+детали с QR/usage), Inbounds (форма с учётом протокола, генератор Reality-ключей, hosts), Cores (рестарт, live-логи, просмотр конфига), Settings (sudo). Анимации: `AnimatePresence` переходы, `layoutId` морфинг карточек/drawer, stagger списков, пружинные микроинтеракции, анимированные счётчики; уважать `prefers-reduced-motion`. Лайв-данные — TanStack Query с `refetchInterval`.

## Деплой

`docker-compose`: контейнер **vpanel** (debian-slim, бандлит vpanel+xray+sing-box, ядра — дочерние процессы под супервизией, `mem_limit:384m`, `GOMEMLIMIT≈256MiB`, `GOGC=50`, `cap_add:NET_ADMIN` для WG) + **caddy:2-alpine** (80/443, авто-TLS только для панели/подписок; протокольные порты публикуются напрямую). Опц. профиль `postgres`. `install.sh`: проверки (root/Debian13/arch/RAM), установка Docker, генерация `.env`+JWT+Reality-ключей, запрос домена/админа, миграции, `vpanel admin create`, swap-файл как страховка, `--uninstall`.

## Последовательность сборки (ранний рабочий срез)

0. Скелет: Go-модуль, Gin, config, GORM+SQLite+миграции, healthcheck, embed-заглушка, Makefile/Dockerfile.
1. Auth + оболочка админки: admin-модель, argon2, JWT, `vpanel admin create`, React login+пустой dashboard.
2. **Inbounds + супервизия Xray** (критический срез): CRUD, генератор конфига, `Process`, gRPC-клиент → поднять VLESS+Reality.
3. Users + провижининг: CRUD, user_inbounds, горячее add/remove через gRPC.
4. **Подписки**: токены, негоциация, рендереры (base64→clash→singbox), `Subscription-Userinfo`, страница info.
5. Трафик/энфорсмент: поллер, накопитель, батч-flush, traffic_log, квоты/срок, сбросы, графики.
6. **sing-box**: конфиг+супервизия+stats → Hysteria2/TUIC/Naive/AnyTLS/AmneziaWG и их рендереры.
7. Полиш: анимации, log-viewer, system-страница, settings, Caddy+compose+install.sh, docs.

Критический путь до «первый VPN через панель» = шаги 0→4 (Xray + подписки). sing-box намеренно отложен до шага 6.

## Точки расширения (Phase 2)

- **Новые ядра (mieru, mtg/MTProto):** новый `internal/core/<name>/{config,api}` + регистрация в `CoreManager`; enum `inbounds.core` и `settings_json` уже это позволяют без миграций.
- **Новые форматы подписок:** через `subscription_templates` (данные, не код), интерфейс `Render(ctx) ([]byte, contentType, error)`.
- **sing-box live-user API**, **Postgres**, **Reality SNI-sharing на 443**, **группы/шаблоны юзеров** — все локализованы и аддитивны.

## Главные риски

1. **Per-user учёт трафика в sing-box** — наименее предсказуемая интеграция (зависит от версии). Пин версии + спайк-валидация на шаге 6, фолбэк на inbound-level учёт. **Проверить рано.**
2. Дрейф Xray gRPC-прото между версиями → пин версии + вендоринг стабов + CI-тест против реального Xray.
3. Restart-штормы → дебаунс + горячее add через gRPC.
4. OOM на 1 ГБ → `mem_limit`/`GOMEMLIMIT`, swap в install.sh, рекоменд. лимит юзеров.

### Бюджет RAM (оба ядра активны, умеренная нагрузка)
Debian+docker ~120-180 + vpanel ~40-90 + Xray ~30-80 + sing-box ~30-80 + Caddy ~30-50 + SQLite ~5-20 = **~300-500 МБ** (пик ~600-750). Влезает с запасом; нативная (без docker) установка экономит ещё ~60-80 МБ.

## Verification

- **Сборка:** `make build` → бинарник стартует, `GET /api/health` отвечает; `vpanel admin create` создаёт sudo-админа; вход в панель.
- **Ядро:** создать VLESS+Reality inbound → Xray стартует (статус в `/api/cores`), порт слушается; создать юзера → подключиться реальным клиентом (v2rayNG) вручную.
- **Подписки:** вставить `/sub/{token}` в v2rayNG и Clash.Meta → конфиги парсятся и подключаются; заголовок `Subscription-Userinfo` показывает квоту/срок.
- **Трафик:** прогнать трафик → `used_*` растёт в UI; выставить маленький `data_limit` → юзер автоматически отключается; сброс цикла включает обратно.
- **sing-box:** поднять Hysteria2 inbound → второе ядро online, подписка отдаёт hy2-конфиг, клиент подключается.
- **Деплой:** на чистой Debian 13 VM `bash install.sh` → панель доступна по HTTPS (Caddy-сертификат), `docker compose ps` показывает оба контейнера в пределах `mem_limit`.
- **Тесты:** unit на рендереры подписок и генераторы конфигов ядер; CI: lint+test+build+образ.
