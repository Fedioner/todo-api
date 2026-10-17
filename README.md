# Todo API

HTTP-сервис на Go для управления задачами. Хранит задачи в PostgreSQL. В учебном Kubernetes-стенде API и база данных работают на трёх рабочих узлах. GitHub Actions проверяет изменения, собирает образы и может развёртывать их в кластере.

## Быстрый запуск

Нужны Docker с Compose и Python 3 для проверки.

```bash
docker compose up -d --build
curl --fail http://127.0.0.1:8080/readyz
python3 scripts/smoke.py http://127.0.0.1:8080
```

Если порт занят, запустите Compose с `TODO_PORT=8081` и используйте этот порт в запросах. Учебный пароль `todo-local` можно заменить через `POSTGRES_PASSWORD` до первого запуска БД. База данных в Compose не публикует порт на хосте. Для остановки используйте `docker compose down`. Том с задачами при этом сохраняется.

```bash
curl -X POST http://127.0.0.1:8080/api/tasks \
  -H 'Content-Type: application/json' \
  -d '{"title":"Подготовиться к защите"}'
curl http://127.0.0.1:8080/api/tasks
```

| Метод и путь | Операция | Успешный ответ |
| --- | --- | --- |
| `GET /api/tasks` | Получить список задач | `200` |
| `POST /api/tasks` | Создать задачу с полем `title` | `201` |
| `PATCH /api/tasks/{id}` | Установить поле `completed` | `200` |
| `DELETE /api/tasks/{id}` | Удалить задачу | `204` |
| `GET /livez` | Проверить, что HTTP-процесс отвечает | `200` |
| `GET /readyz` | Проверить доступ к таблице на первичном сервере БД | `200` |

При недоступной БД прикладной запрос возвращает `503`. Ответы содержат заголовок `X-App-Version`. В локальной сборке его значение равно `dev`. В образе CI оно совпадает с SHA коммита.

## Учебный стенд Kubernetes

Нужны Docker, kind 0.32.0, kubectl и доступ к публичным реестрам. Стенд использует Kubernetes 1.36.1 и CloudNativePG 1.30.2.

```bash
export KUBECONFIG="$PWD/.kubeconfig"
bash scripts/prepare-kind.sh
kubectl --context kind-todo-lab -n todo get pods -o wide
kubectl --context kind-todo-lab -n todo port-forward svc/todo-api 8080:80
```

В другом терминале запустите `python3 scripts/smoke.py http://127.0.0.1:8080`. Для повторной сборки локального образа запустите подготовку ещё раз. Если Deployment использует `todo-api:dev`, перезапустите его через `kubectl --context kind-todo-lab -n todo rollout restart deployment/todo-api`.

```text
                         ┌─── Pod API 1 ───┐
Клиент → Service todo-api ─── Pod API 2 ────→ todo-db-rw → PostgreSQL
                         └─── Pod API 3 ───┘
```

Каждый экземпляр PostgreSQL получает собственный том. CloudNativePG переключает `todo-db-rw` на новый первичный экземпляр при его отказе. API работает без локального хранилища и распределён по рабочим узлам. В Deployment используется последовательное обновление с запуском дополнительной реплики.

### Проверка отказа

Для наблюдения используйте клиент на управляющем узле. Он обращается к самому Service и остаётся на месте при остановке рабочего узла.

```bash
kubectl --context kind-todo-lab apply -f deploy/probe.yaml
kubectl --context kind-todo-lab -n todo wait --for=condition=Ready pod/todo-probe --timeout=120s
kubectl --context kind-todo-lab -n todo exec todo-probe -- sh -c '
  while true; do
    date -u
    curl -sS --max-time 3 -o /dev/null -w "%{http_code}\n" http://todo-api/api/tasks
    sleep 1
  done'
```

В другом терминале выведите `kubectl --context kind-todo-lab -n todo get pods -o wide` и удалите один выбранный Pod API. Сравните ответы клиента и дождитесь восстановления трёх готовых реплик. Измерения отдельных отказов выполняйте по очереди.

Для проверки PostgreSQL создайте контрольную задачу и сохраните её идентификатор. На стенде с именем `todo-lab` команды ниже останавливают рабочий узел с первичным экземпляром БД.

```bash
primary=$(kubectl --context kind-todo-lab -n todo get cluster todo-db \
  -o jsonpath='{.status.currentPrimary}')
node=$(kubectl --context kind-todo-lab -n todo get pod "$primary" \
  -o jsonpath='{.spec.nodeName}')
case "$node" in
  todo-lab-worker|todo-lab-worker2|todo-lab-worker3) docker stop --timeout 1 "$node" ;;
  *) echo "Узел не относится к стенду todo-lab: $node" >&2 ;;
esac
kubectl --context kind-todo-lab -n todo get cluster todo-db -w
```

После смены первичного экземпляра остановите просмотр через Ctrl+C. Проверьте чтение контрольной задачи и создание новой. Затем верните узел в кластер командой `docker start "$node"` и дождитесь готовности трёх экземпляров БД.

Логические узлы находятся на одном компьютере. Отказ Docker Desktop или самого компьютера останавливает стенд целиком. Репликация PostgreSQL настроена асинхронно, поэтому при переключении возможна потеря ещё не доставленных на реплики изменений.

## CI/CD

- Pull request запускает `go vet`, тесты с детектором гонок и сквозную проверку HTTP через Compose.
- Изменение в `main` после успешных проверок публикует образ для `linux/amd64` и `linux/arm64` в GHCR. Тег образа совпадает с SHA коммита, развёртывание использует digest.
- CD запускается, когда переменная репозитория `TODO_DEPLOY_ENABLED` равна `true`. Нужен self-hosted runner с меткой `todo-lab`, Bash, Python 3, kubectl и доступом к кластеру. Контекст задаётся через `TODO_KUBE_CONTEXT`, по умолчанию используется `kind-todo-lab`.
- На кластере примените `deploy/cd-rbac.yaml`. Настройте kubeconfig исполнителя для ServiceAccount `todo-deployer` в пространстве имён `todo`. Для закрытого пакета GHCR настройте чтение образа из кластера. Учётные данные хранятся вне репозитория.
- При сбое обновления или проверки API сценарий возвращает предыдущий образ. Статус задания остаётся ошибкой. Схема БД при этом сохраняется.

## Разработка

```bash
go test -race ./...
go vet ./...
```

Точка входа сервиса находится в `cmd/todo-api`, обработчики и запросы к БД — в `internal/todo`, схема — в `db/schema.sql`, конфигурация Kubernetes — в `deploy`. Для запуска API вне контейнера передайте `DATABASE_URL` и при необходимости `HTTP_ADDR`, затем выполните `go run ./cmd/todo-api`. Исходную консольную версию можно запустить командой `go run .`.

Проект курсовой работы по дисциплине «Настройка и администрирование сервисного программного обеспечения», РТУ МИРЭА.
