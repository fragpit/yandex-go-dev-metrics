# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m main template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/main .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:

- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Профилирование и оптимизация памяти

### Результаты

**До оптимизации:**

- Общее потребление: 5579 MB
- Основные проблемы: io.ReadAll, pgx.NamedArgs, 
  strings.Builder

**После оптимизации:**

- Общее потребление: 2458 MB (-56%)
- Устранено: 3121 MB аллокаций

### Изменения

1. Замена `io.ReadAll` + `json.Unmarshal` → `json.NewDecoder`
2. Замена `pgx.NamedArgs` → позиционные параметры
3. Условное чтение body в middleware (только debug)

### Команды для проверки

```bash
# Сравнение профилей
go tool pprof \
    -top \
    -sample_index=alloc_space \
    -diff_base=profiles/base.pprof \
    profiles/result.pprof
```

**Результат команды проверки:**

```txt
File: main
Type: alloc_space
Time: 2025-10-13 21:38:45 +03
Showing nodes accounting for -3356.99MB, 58.12% of 5776.45MB total
Dropped 199 nodes (cum <= 28.88MB)
      flat  flat%   sum%        cum   cum%
 -932.97MB 16.15% 16.15%  -934.97MB 16.19%  io.ReadAll
 -703.63MB 12.18% 28.33%  -703.63MB 12.18%  strings.(*Builder).WriteString (inline)
 -506.63MB  8.77% 37.10% -2374.83MB 41.11%  github.com/fragpit/yandex-go-dev-metrics/internal/storage/postgresql.(*Storage).SetOrUpdateMetricBatch
 -480.57MB  8.32% 45.42%  -480.57MB  8.32%  github.com/jackc/pgx/v5.namedArgState
 -434.03MB  7.51% 52.94%  -434.03MB  7.51%  github.com/jackc/pgx/v5.rawState
  327.40MB  5.67% 47.27%   330.40MB  5.72%  encoding/json.(*Decoder).refill
 -277.52MB  4.80% 52.07% -1895.75MB 32.82%  github.com/jackc/pgx/v5.rewriteQuery
 -259.18MB  4.49% 56.56% -3186.09MB 55.16%  github.com/fragpit/yandex-go-dev-metrics/internal/router.(*Router).slogMiddleware-fm.(*Router).slogMiddleware.func1
 -104.31MB  1.81% 58.37%  -104.31MB  1.81%  io.init.func1
   49.01MB  0.85% 57.52%    49.01MB  0.85%  encoding/json.NewDecoder (inline)
  -39.58MB  0.69% 58.20%   -39.58MB  0.69%  sync.(*Pool).pinSlow
   38.06MB  0.66% 57.54%    38.06MB  0.66%  github.com/jackc/pgx/v5/internal/pgio.AppendUint32 (inline)
   24.50MB  0.42% 57.12% -2417.38MB 41.85%  github.com/fragpit/yandex-go-dev-metrics/internal/router.Router.updatesHandler
  -24.50MB  0.42% 57.54%  -197.51MB  3.42%  encoding/json.Unmarshal
     -20MB  0.35% 57.89%      -20MB  0.35%  container/list.(*List).insertValue (inline)
  -17.03MB  0.29% 58.18%   -17.03MB  0.29%  bufio.NewWriterSize (inline)
       5MB 0.087% 58.10% -2408.88MB 41.70%  github.com/go-chi/chi/v5/middleware.(*Compressor).Handler-fm.(*Compressor).Handler.func1
      -2MB 0.035% 58.13% -1870.20MB 32.38%  github.com/jackc/pgx/v5/pgxpool.(*Pool).SendBatch
    1.50MB 0.026% 58.11% -3338.99MB 57.80%  net/http.(*conn).serve
   -0.50MB 0.0087% 58.12%   -26.05MB  0.45%  net/http.(*conn).readRequest
         0     0% 58.12%      -20MB  0.35%  container/list.(*List).PushBack (inline)
         0     0% 58.12%   523.40MB  9.06%  encoding/json.(*Decoder).Decode
         0     0% 58.12%   337.40MB  5.84%  encoding/json.(*Decoder).readValue
         0     0% 58.12%    19.50MB  0.34%  encoding/json.(*decodeState).array
         0     0% 58.12%    20.50MB  0.35%  encoding/json.(*decodeState).unmarshal
         0     0% 58.12%    19.50MB  0.34%  encoding/json.(*decodeState).value
         0     0% 58.12% -2418.38MB 41.87%  github.com/fragpit/yandex-go-dev-metrics/internal/router.(*Router).decompressMiddleware-fm.(*Router).decompressMiddleware.func1
         0     0% 58.12% -2418.38MB 41.87%  github.com/go-chi/chi/v5.(*Mux).Mount.func1
         0     0% 58.12% -3182.60MB 55.10%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 58.12% -2418.38MB 41.87%  github.com/go-chi/chi/v5.(*Mux).routeHTTP
         0     0% 58.12% -1879.71MB 32.54%  github.com/jackc/pgx/v5.(*Conn).SendBatch
         0     0% 58.12% -1895.75MB 32.82%  github.com/jackc/pgx/v5.NamedArgs.RewriteQuery
         0     0% 58.12%    38.06MB  0.66%  github.com/jackc/pgx/v5/internal/pgio.AppendInt32 (inline)
         0     0% 58.12%    20.05MB  0.35%  github.com/jackc/pgx/v5/pgconn.(*Pipeline).SendQueryPrepared
         0     0% 58.12%    38.06MB  0.66%  github.com/jackc/pgx/v5/pgproto3.(*Bind).Encode
         0     0% 58.12%    38.06MB  0.66%  github.com/jackc/pgx/v5/pgproto3.(*Frontend).SendBind
         0     0% 58.12% -1879.71MB 32.54%  github.com/jackc/pgx/v5/pgxpool.(*Conn).SendBatch
         0     0% 58.12%  -111.32MB  1.93%  io.Copy (inline)
         0     0% 58.12%  -112.82MB  1.95%  io.CopyN
         0     0% 58.12%  -111.32MB  1.93%  io.copyBuffer
         0     0% 58.12%  -111.32MB  1.93%  io.discard.ReadFrom
         0     0% 58.12%   -17.02MB  0.29%  log/slog.(*Logger).Info
         0     0% 58.12%   -17.02MB  0.29%  log/slog.(*Logger).log
         0     0% 58.12%   -17.52MB   0.3%  log/slog.(*TextHandler).Handle
         0     0% 58.12%   -17.52MB   0.3%  log/slog.(*commonHandler).handle
         0     0% 58.12%  -120.33MB  2.08%  net/http.(*chunkWriter).close
         0     0% 58.12%  -120.33MB  2.08%  net/http.(*chunkWriter).writeHeader
         0     0% 58.12%  -131.85MB  2.28%  net/http.(*response).finishRequest
         0     0% 58.12% -3186.09MB 55.16%  net/http.HandlerFunc.ServeHTTP
         0     0% 58.12% -3182.60MB 55.10%  net/http.serverHandler.ServeHTTP
         0     0% 58.12%  -140.87MB  2.44%  sync.(*Pool).Get
         0     0% 58.12%   -19.03MB  0.33%  sync.(*Pool).Put
         0     0% 58.12%   -39.58MB  0.69%  sync.(*Pool).pin
```

```bash
# Сравнение профилей
go tool pprof \
    -top \
    -diff_base=profiles/base.pprof \
    profiles/result.pprof
```

**Результат команды проверки:**

```txt
File: main
Type: inuse_space
Time: 2025-10-13 21:38:45 +03
Showing nodes accounting for 1536kB, 42.77% of 3591.15kB total
Dropped 8 nodes (cum <= 17.96kB)
      flat  flat%   sum%        cum   cum%
    1539kB 42.86% 42.86%  1026.44kB 28.58%  runtime.allocm
    -515kB 14.34% 28.51%     -515kB 14.34%  regexp/syntax.(*compiler).inst (inline)
     513kB 14.29% 42.80%      513kB 14.29%  bufio.NewWriterSize (inline)
 -512.56kB 14.27% 28.53%  -512.56kB 14.27%  github.com/jackc/pgx/v5/pgproto3.NewFrontend
 -512.56kB 14.27% 14.25%  -512.56kB 14.27%  runtime.makeProfStackFP (inline)
  512.05kB 14.26% 28.51%   512.05kB 14.26%  context.(*cancelCtx).Done
  512.05kB 14.26% 42.77%   512.05kB 14.26%  github.com/jackc/pgx/v5/pgconn/ctxwatch.(*ContextWatcher).Watch.func1
  512.05kB 14.26% 57.03%   512.05kB 14.26%  runtime.acquireSudog
 -512.02kB 14.26% 42.77%  -512.02kB 14.26%  github.com/fragpit/yandex-go-dev-metrics/internal/router.(*Router).slogMiddleware-fm.(*Router).slogMiddleware.func1
         0     0% 42.77%     -515kB 14.34%  github.com/Masterminds/semver/v3.init.0
         0     0% 42.77%  -512.02kB 14.26%  github.com/go-chi/chi/v5.(*Mux).ServeHTTP
         0     0% 42.77%  -512.56kB 14.27%  github.com/jackc/pgx/v5.ConnectConfig
         0     0% 42.77%  -512.56kB 14.27%  github.com/jackc/pgx/v5.connect
         0     0% 42.77%  -512.56kB 14.27%  github.com/jackc/pgx/v5/pgconn.ConnectConfig
         0     0% 42.77%  -512.56kB 14.27%  github.com/jackc/pgx/v5/pgconn.ParseConfigWithOptions.func1
         0     0% 42.77%  -512.56kB 14.27%  github.com/jackc/pgx/v5/pgconn.connectOne
         0     0% 42.77%  -512.56kB 14.27%  github.com/jackc/pgx/v5/pgconn.connectPreferred
         0     0% 42.77%  -512.56kB 14.27%  github.com/jackc/pgx/v5/pgxpool.NewWithConfig.func3
         0     0% 42.77%  -512.56kB 14.27%  github.com/jackc/puddle/v2.(*Pool[go.shape.*uint8]).initResourceValue.func1
         0     0% 42.77%   512.08kB 14.26%  github.com/jackc/tern/v2/migrate.init
         0     0% 42.77%      513kB 14.29%  net/http.(*conn).readRequest
         0     0% 42.77%  -512.02kB 14.26%  net/http.HandlerFunc.ServeHTTP
         0     0% 42.77%      513kB 14.29%  net/http.newBufioWriterSize
         0     0% 42.77%  -512.02kB 14.26%  net/http.serverHandler.ServeHTTP
         0     0% 42.77%   512.05kB 14.26%  os/signal.NotifyContext.func1
         0     0% 42.77%     -515kB 14.34%  regexp/syntax.(*compiler).compile
         0     0% 42.77%     -515kB 14.34%  regexp/syntax.(*compiler).rune
         0     0% 42.77%     -515kB 14.34%  regexp/syntax.Compile
         0     0% 42.77%   512.05kB 14.26%  runtime.ensureSigM.func1
         0     0% 42.77%      513kB 14.29%  runtime.goexit0
         0     0% 42.77%  -512.56kB 14.27%  runtime.handoffp
         0     0% 42.77%  -512.56kB 14.27%  runtime.mProfStackInit (inline)
         0     0% 42.77%   513.44kB 14.30%  runtime.mcall
         0     0% 42.77%  -512.56kB 14.27%  runtime.mcommoninit
         0     0% 42.77%      513kB 14.29%  runtime.mstart
         0     0% 42.77%      513kB 14.29%  runtime.mstart0
         0     0% 42.77%      513kB 14.29%  runtime.mstart1
         0     0% 42.77%  1026.44kB 28.58%  runtime.newm
         0     0% 42.77%     1539kB 42.86%  runtime.resetspinning
         0     0% 42.77%  1026.44kB 28.58%  runtime.schedule
         0     0% 42.77%   512.05kB 14.26%  runtime.selectgo
         0     0% 42.77%  1026.44kB 28.58%  runtime.startm
         0     0% 42.77%  -512.56kB 14.27%  runtime.stoplockedm
         0     0% 42.77%     1539kB 42.86%  runtime.wakep
```
