package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/10Narratives/faas/internal/app/components/nats"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	managerActiveInstances = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "faas",
		Subsystem: "manager",
		Name:      "active_instances",
		Help:      "Number of active function instances",
	}, []string{"pod", "function"})

	managerTotalInstances = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "faas",
		Subsystem: "manager",
		Name:      "total_instances",
		Help:      "Total number of function instances (active + inactive)",
	}, []string{"pod"})

	managerInstanceCreationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "faas",
		Subsystem: "manager",
		Name:      "instance_creations_total",
		Help:      "Total number of instance creations",
	}, []string{"pod", "function"})

	managerInstanceDeletionsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "faas",
		Subsystem: "manager",
		Name:      "instance_deletions_total",
		Help:      "Total number of instance deletions",
	}, []string{"pod", "function"})

	consumerMessagesFetchedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "faas",
		Subsystem: "consumer",
		Name:      "messages_fetched_total",
		Help:      "Total number of messages fetched from queues",
	}, []string{"pod", "consumer_type", "function"})

	consumerMessagesProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "faas",
		Subsystem: "consumer",
		Name:      "messages_processed_total",
		Help:      "Total number of messages successfully processed",
	}, []string{"pod", "function"})

	consumerMessagesFailedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "faas",
		Subsystem: "consumer",
		Name:      "messages_failed_total",
		Help:      "Total number of messages that failed processing",
	}, []string{"pod", "function", "reason"})

	consumerPollEmptyTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "faas",
		Subsystem: "consumer",
		Name:      "poll_empty_total",
		Help:      "Total number of empty polls",
	}, []string{"pod", "consumer_type", "function"})

	taskExecutionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "faas",
		Subsystem: "task",
		Name:      "execution_duration_seconds",
		Help:      "Task execution duration in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"pod", "function"})

	taskPayloadSizeBytes = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "faas",
		Subsystem: "task",
		Name:      "payload_size_bytes",
		Help:      "Task payload size in bytes",
		Buckets:   []float64{256, 1024, 4096, 16384, 65536, 262144},
	}, []string{"pod", "function"})

	managerMaxInstancesReachedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "faas",
		Subsystem: "manager",
		Name:      "max_instances_reached_total",
		Help:      "Total number of times max instances limit was reached",
	}, []string{"pod"})
)

type ManagerConfig struct {
	MaxInstances     int
	InstanceLifetime time.Duration // Время жизни инстанса после последнего использования
	ColdStart        time.Duration // Задержка холодного старта
	NATSURL          string
	PodName          string
	MaxAckPending    int
	AckWait          time.Duration
	MaxDeliver       int
	Backoff          []time.Duration
}

type Manager struct {
	cfg               *ManagerConfig
	log               *zap.Logger
	us                *nats.UnifiedStorage
	workers           []*worker
	consumersMu       sync.RWMutex
	consumers         map[string]jetstream.Consumer
	subjects          []string
	hintBuffer        chan string
	lastExecutionTime map[string]time.Time // Для отслеживания холодных стартов
	lastExecMu        sync.RWMutex
	taskPool          sync.Pool
}

type worker struct {
	id       int
	current  string
	lastPoll time.Time
	active   bool
	fn       string
}

func NewManager(log *zap.Logger, cfg *ManagerConfig) (*Manager, error) {
	if cfg.PodName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("get hostname: %w", err)
		}
		cfg.PodName = fmt.Sprintf("faas-agent-%s", hostname)
		log.Info("generated pod name", zap.String("pod_name", cfg.PodName))
	}

	us, err := nats.NewUnifiedStorage(cfg.NATSURL)
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}

	// Create fixed pool of workers
	workers := make([]*worker, cfg.MaxInstances)
	for i := 0; i < cfg.MaxInstances; i++ {
		workers[i] = &worker{id: i}
		managerInstanceCreationsTotal.WithLabelValues(cfg.PodName, fmt.Sprintf("worker-%d", i)).Inc()
	}

	// Register pod-level metrics
	managerTotalInstances.WithLabelValues(cfg.PodName).Set(float64(cfg.MaxInstances))

	// Инициализация пула задач
	taskPool := sync.Pool{
		New: func() interface{} {
			return &Task{}
		},
	}

	log.Info("manager created",
		zap.String("nats", cfg.NATSURL),
		zap.String("pod_name", cfg.PodName),
		zap.Int("max_instances", cfg.MaxInstances),
		zap.Duration("cold_start", cfg.ColdStart),
		zap.Duration("instance_lifetime", cfg.InstanceLifetime))

	return &Manager{
		cfg:               cfg,
		log:               log,
		us:                us,
		workers:           workers,
		consumers:         make(map[string]jetstream.Consumer),
		hintBuffer:        make(chan string, 1000), // Увеличен буфер подсказок
		lastExecutionTime: make(map[string]time.Time),
		taskPool:          taskPool,
	}, nil
}

func (m *Manager) Run(ctx context.Context) error {
	m.log.Info("manager started", zap.Int("max_instances", m.cfg.MaxInstances))

	hintsCtx, hintsCancel := context.WithCancel(ctx)
	hintsErr := make(chan error, 1)

	go func() {
		defer hintsCancel()
		m.log.Info("hints consumer goroutine started")
		hintsErr <- m.runHintsConsumer(hintsCtx)
	}()

	// Start all workers
	for i, w := range m.workers {
		go func(workerID int, w *worker) {
			m.workerLoop(ctx, w)
		}(i, w)
	}

	ticker := time.NewTicker(1 * time.Second) // Обновляем метрики чаще
	defer ticker.Stop()

	lastProcessed := make(map[string]int)
	for {
		select {
		case <-ctx.Done():
			m.log.Info("manager context cancelled")
			return ctx.Err()
		case err := <-hintsErr:
			return fmt.Errorf("hints consumer failed: %w", err)
		case <-ticker.C:
			m.updateInstanceMetrics()
			m.logProcessingRate(lastProcessed)
		}
	}
}

func (m *Manager) logProcessingRate(lastProcessed map[string]int) {
	currentProcessed := make(map[string]int)
	total := 0

	m.consumersMu.RLock()
	for subject, cons := range m.consumers {
		if stats, err := cons.Info(context.Background()); err == nil {
			fn := strings.TrimPrefix(subject, "task.")
			current := int(stats.Delivered.Consumer)
			diff := current - lastProcessed[fn]
			if diff > 0 {
				currentProcessed[fn] = diff
				total += diff
			}
		}
	}
	m.consumersMu.RUnlock()

	if total > 0 {
		m.log.Info("processing rate",
			zap.Int("total_events/sec", total),
			zap.Any("by_function", currentProcessed))
	}

	// Обновляем прошлые значения
	for fn, count := range currentProcessed {
		lastProcessed[fn] = count
	}
}

func (m *Manager) updateInstanceMetrics() {
	activeByFunction := make(map[string]int)
	activeTotal := 0

	for _, w := range m.workers {
		if w.active {
			activeTotal++
			if w.fn != "" {
				activeByFunction[w.fn]++
			}
		}
	}

	// Update metrics
	managerActiveInstances.WithLabelValues(m.cfg.PodName, "all").Set(float64(activeTotal))
	for fn, count := range activeByFunction {
		managerActiveInstances.WithLabelValues(m.cfg.PodName, fn).Set(float64(count))
	}
}

func (m *Manager) workerLoop(ctx context.Context, w *worker) {
	// Локальный буфер для обработки пачки сообщений
	const batchSize = 25
	msgs := make([]jetstream.Msg, 0, batchSize)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			m.processWorkBatch(ctx, w, msgs[:0])
		}
	}
}

func (m *Manager) processWorkBatch(ctx context.Context, w *worker, msgs []jetstream.Msg) {
	// Сначала пробуем обработать текущую очередь пачкой
	if w.current != "" {
		if m.processQueueBatch(ctx, w, msgs) {
			return
		}
	}

	// Проверяем подсказки
	select {
	case hint := <-m.hintBuffer:
		w.current = hint
		w.lastPoll = time.Now()
		return
	default:
	}

	// Round-robin по известным очередям
	if time.Since(w.lastPoll) > 200*time.Millisecond && len(m.subjects) > 0 {
		w.current = m.subjects[w.id%len(m.subjects)]
		w.lastPoll = time.Now()
	}
}

func (m *Manager) processQueueBatch(ctx context.Context, w *worker, msgs []jetstream.Msg) bool {
	cons, err := m.getConsumer(ctx, w.current)
	if err != nil {
		return false
	}

	// Забираем сразу пачку сообщений
	batch, err := cons.Fetch(25, jetstream.FetchMaxWait(20*time.Millisecond))
	if err != nil {
		if !errors.Is(err, jetstream.ErrNoMessages) {
			m.log.Debug("batch fetch failed", zap.String("subject", w.current), zap.Error(err))
		}
		consumerPollEmptyTotal.WithLabelValues(m.cfg.PodName, "task", w.current).Inc()
		w.lastPoll = time.Now()
		return false
	}

	msgCount := 0
	for msg := range batch.Messages() {
		if msgCount >= 25 { // Лимит на пачку
			break
		}
		msgs = append(msgs, msg)
		msgCount++
	}

	if msgCount == 0 {
		return false
	}

	consumerMessagesFetchedTotal.WithLabelValues(m.cfg.PodName, "task", w.current).Add(float64(msgCount))
	fnName := strings.TrimPrefix(w.current, "task.")

	// Атомарно обновляем состояние воркера
	w.active = true
	w.fn = fnName
	defer func() {
		w.active = false
		w.fn = ""
	}()

	// Обрабатываем все сообщения в пачке
	for _, msg := range msgs {
		startTime := time.Now()
		m.processSingleMessage(ctx, w, msg, fnName, startTime)
	}

	return true
}

func (m *Manager) processSingleMessage(ctx context.Context, w *worker, msg jetstream.Msg, fnName string, startTime time.Time) {
	task := m.taskPool.Get().(*Task)
	defer func() {
		*task = Task{} // Очищаем структуру
		m.taskPool.Put(task)
	}()

	if err := json.Unmarshal(msg.Data(), task); err != nil {
		consumerMessagesFailedTotal.WithLabelValues(m.cfg.PodName, fnName, "unmarshal").Inc()
		msg.Nak()
		return
	}

	if task.FunctionID != fnName {
		consumerMessagesFailedTotal.WithLabelValues(m.cfg.PodName, fnName, "mismatch").Inc()
		msg.Ack()
		return
	}

	taskPayloadSizeBytes.WithLabelValues(m.cfg.PodName, fnName).Observe(float64(len(msg.Data())))

	// Выполняем задачу
	if err := m.executeTask(ctx, task, fnName); err != nil {
		consumerMessagesFailedTotal.WithLabelValues(m.cfg.PodName, fnName, "execute").Inc()
		time.Sleep(10 * time.Millisecond) // Короткая задержка при ошибке
		msg.Nak()
		return
	}

	taskExecutionDuration.WithLabelValues(m.cfg.PodName, fnName).Observe(time.Since(startTime).Seconds())
	consumerMessagesProcessedTotal.WithLabelValues(m.cfg.PodName, fnName).Inc()

	if err := msg.Ack(); err != nil {
		m.log.Debug("task ack failed", zap.String("task_id", task.TaskID), zap.Error(err))
	}
}

func (m *Manager) executeTask(ctx context.Context, task *Task, fnName string) error {
	// Проверяем необходимость холодного старта без блокировки
	m.lastExecMu.RLock()
	lastExec, exists := m.lastExecutionTime[fnName]
	needsColdStart := !exists || time.Since(lastExec) > m.cfg.InstanceLifetime
	m.lastExecMu.RUnlock()

	// Создаем контекст с запасом
	timeout := task.ExecutionTime + m.cfg.ColdStart + 500*time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Симулируем холодный старт только если нужен
	if needsColdStart {
		// Обновляем время последнего выполнения
		m.lastExecMu.Lock()
		m.lastExecutionTime[fnName] = time.Now()
		m.lastExecMu.Unlock()

		select {
		case <-time.After(m.cfg.ColdStart):
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Симулируем выполнение задачи
	select {
	case <-time.After(task.ExecutionTime):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *Manager) getConsumer(ctx context.Context, subject string) (jetstream.Consumer, error) {
	m.consumersMu.RLock()
	if cons, ok := m.consumers[subject]; ok {
		m.consumersMu.RUnlock()
		return cons, nil
	}
	m.consumersMu.RUnlock()

	m.consumersMu.Lock()
	defer m.consumersMu.Unlock()

	// Проверяем еще раз после получения блокировки
	if cons, ok := m.consumers[subject]; ok {
		return cons, nil
	}

	durable := fmt.Sprintf("%s-cons-%s", m.cfg.PodName, strings.ReplaceAll(strings.TrimPrefix(subject, "task."), "_", "-"))
	cfg := jetstream.ConsumerConfig{
		Durable:       durable,
		FilterSubject: subject,
		AckPolicy:     jetstream.AckExplicitPolicy,
		AckWait:       m.cfg.AckWait,
		MaxAckPending: m.cfg.MaxAckPending,
		MaxDeliver:    m.cfg.MaxDeliver,
		BackOff:       m.cfg.Backoff,
	}

	cons, err := m.us.TaskStream.CreateOrUpdateConsumer(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create consumer %s: %w", durable, err)
	}

	m.consumers[subject] = cons
	if !contains(m.subjects, subject) {
		m.subjects = append(m.subjects, subject)
	}
	return cons, nil
}

func (m *Manager) runHintsConsumer(ctx context.Context) error {
	cons, err := m.getConsumer(ctx, "task.hints")
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Забираем подсказки пачками
			msgs, err := cons.Fetch(100, jetstream.FetchMaxWait(50*time.Millisecond))
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				time.Sleep(10 * time.Millisecond)
				continue
			}

			for msg := range msgs.Messages() {
				subject := strings.TrimSpace(string(msg.Data()))
				if strings.HasPrefix(subject, "task.") {
					select {
					case m.hintBuffer <- subject:
					default:
						m.log.Debug("hint buffer full", zap.String("subject", subject))
					}
				}
				msg.Ack()
			}
		}
	}
}

func (m *Manager) Stop(ctx context.Context) error {
	if m.us != nil {
		m.us.Conn.Close()
	}
	return nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
