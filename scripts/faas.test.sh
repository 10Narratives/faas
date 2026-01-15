#!/bin/bash
# generate_tasks_bg.sh - Запускает 20 потоков генерации задач в фоновом режиме

# Исправлено: правильно задаем дефолтные значения
COUNT=${COUNT:-10000}        
MIN_TIME=${MIN_TIME:-10ms}   
MAX_TIME=${MAX_TIME:-100ms} 
WAIT=${WAIT:-10ms}
NATS_URL=${NATS_URL:-"nats://localhost:4222"}

FUNCTIONS=("image-processor" "data-analyzer" "report-generator" "video-editor" "audio-processor")
PID_FILE="/tmp/faas_generator_pids"

# Очистка старого файла PIDs
> "$PID_FILE"

echo "🚀 Запуск 20 потоков генерации задач в фоновом режиме..."
echo "   NATS URL: $NATS_URL"
echo "   Задач на поток: $COUNT"
echo "   Диапазон времени: $MIN_TIME - $MAX_TIME"
echo "   Пауза: $WAIT"
echo "   Функции: ${FUNCTIONS[*]}"
echo "----------------------------------------"
echo "ℹ️  PID файл: $PID_FILE"
echo "ℹ️  Для остановки выполните: pkill -P \$(cat $PID_FILE) && rm $PID_FILE"
echo "----------------------------------------"

for i in $(seq 1 20); do
    func_idx=$(( (i-1) % ${#FUNCTIONS[@]} ))
    function="${FUNCTIONS[$func_idx]}"
    
    prefix="thread$(printf "%02d" $i)"
    
    (
        # Перенаправляем вывод в /dev/null для фоновой работы
        exec >/dev/null 2>&1
        
        sleep $(( (i-1) * 100 ))ms
        NATS_URL="$NATS_URL" \
        go run cmd/generator/main.go \
            -count=$COUNT \
            -min-time=$MIN_TIME \
            -max-time=$MAX_TIME \
            -wait=$WAIT \
            -function="$function"
    ) &
    
    echo $! >> "$PID_FILE"
    
    # Прогресс для интерактивного запуска
    if [ -t 1 ]; then
        progress=$(( i * 100 / 20 ))
        printf "\r📈 Запущено: [%-20s] %d%%" "$(printf "#%.0s" $(seq 1 $((progress/5))))" "$progress"
        sleep 0.05
    fi
done

if [ -t 1 ]; then
    printf "\n✅ Все 20 потоков запущены в фоновом режиме\n"
fi

echo "----------------------------------------"
echo "ℹ️  Процессы работают в фоне. PID сохранены в $PID_FILE"
echo "ℹ️  Для проверки: ps -p \$(cat $PID_FILE | tr '\n' ',')"
echo "ℹ️  Для остановки: bash -c 'pkill -P \$(cat $PID_FILE) 2>/dev/null; kill \$(cat $PID_FILE) 2>/dev/null; rm -f $PID_FILE'"
echo "----------------------------------------"
exit 0