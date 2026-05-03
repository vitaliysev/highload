# k6 нагрузочные тесты

| Скрипт      | Что тестирует                          | VU     | Время  |
|-------------|----------------------------------------|--------|--------|
| `smoke.js`  | Базовая работоспособность              | 5      | 30 сек |
| `load.js`   | Устойчивая нагрузка (reads + writes)   | 100+30 | 7 мин  |
| `stress.js` | Поиск потолка RPS                      | 300+60 | 7 мин  |
| `spike.js`  | 3x спайк и восстановление              | 300+80 | 6 мин  |

## Запуск

```bash
# smoke
./loadtest/run.sh smoke

# load
./loadtest/run.sh load

# stress
./loadtest/run.sh stress

# spike
./loadtest/run.sh spike
```

Если удалённая VM:

```bash
k6 run -e BASE_URL=http://<VM_IP>:8080 loadtest/stress.js
```

Отчёты сохраняются в `loadtest/reports/`.
