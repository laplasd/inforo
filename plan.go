package inforo

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/laplasd/inforo/api"
	"github.com/laplasd/inforo/model"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type PlanRegistry struct {
	plans      map[string]*model.Plan
	Components api.ComponentRegistry
	Tasks      api.TaskRegistry
	*StatusManager
	*Events
	mu     *sync.RWMutex
	logger *logrus.Logger
}

type PlanRegistryOptions struct {
	Logger        *logrus.Logger
	Components    api.ComponentRegistry
	Tasks         api.TaskRegistry
	StatusManager *StatusManager
	EventManager  *Events
}

func NewPlanRegistry(opts PlanRegistryOptions) (api.PlanRegistry, error) {
	return &PlanRegistry{
		mu:            &sync.RWMutex{},
		logger:        opts.Logger,
		StatusManager: opts.StatusManager,
		plans:         make(map[string]*model.Plan),
		Components:    opts.Components,
		Tasks:         opts.Tasks,
	}, nil
}

func (pr *PlanRegistry) Register(tasks []model.Task) (*model.Plan, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if len(tasks) == 0 {
		return nil, errors.New("plan must contain at least one task")
	}

	// 1. Подготовка данных и валидация
	taskMap := make(map[string]*model.Task)
	dependencyGraph := make(map[string][]string)
	reverseGraph := make(map[string][]string)

	// Первый проход: регистрация и валидация задач
	for _, task := range tasks {

		// Регистрируем задачу и получаем полную версию с заполненными полями
		registeredTask, err := pr.Tasks.Register(task)
		if err != nil {
			return nil, fmt.Errorf("failed to register task %s: %w", task.ID, err)
		}

		taskMap[task.ID] = registeredTask
		dependencyGraph[task.ID] = []string{}
		reverseGraph[task.ID] = []string{}
	}

	// Второй проход: построение графов зависимостей
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			if _, exists := taskMap[dep.ID]; !exists {
				return nil, fmt.Errorf("dependency %s not found", dep.ID)
			}

			dependencyGraph[task.ID] = append(dependencyGraph[task.ID], dep.ID)
			reverseGraph[dep.ID] = append(reverseGraph[dep.ID], task.ID)
		}
	}

	// 2. Разделение на независимые графы
	graphs, err := pr.buildIndependentGraphs(taskMap, dependencyGraph, reverseGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to build task graphs: %w", err)
	}

	// 3. Проверка на циклические зависимости в каждом графе
	for _, graph := range graphs {
		if err := pr.detectCycles(graph.Dependencies); err != nil {
			return nil, fmt.Errorf("cycle detected in graph %s: %w", graph.RootTaskID, err)
		}
	}

	// 4. Создание плана
	plan := &model.Plan{
		ID:            uuid.New().String(),
		TaskGraphs:    graphs,
		StatusHistory: pr.StatusManager.NewStatus(model.StatusCreated),
		EventHistory:  &model.EventHistory{},
		RollbackStack: make([]*model.RollbackCheckpoint, 0),
	}

	// 5. Сохранение и логирование
	pr.plans[plan.ID] = plan
	pr.logger.Infof("Created new plan %s with %d independent task graphs",
		plan.ID, len(graphs))

	return plan, nil
}

func (pr *PlanRegistry) buildIndependentGraphs(
	tasks map[string]*model.Task,
	deps map[string][]string,
	revDeps map[string][]string,
) ([]*model.TaskGraph, error) {

	visited := make(map[string]bool)
	var graphs []*model.TaskGraph

	// Сначала находим все корневые задачи (без зависимостей)
	var roots []string
	for taskID := range tasks {
		if len(deps[taskID]) == 0 {
			roots = append(roots, taskID)
		}
	}

	// Если все задачи имеют зависимости, начинаем с любой задачи
	if len(roots) == 0 {
		for taskID := range tasks {
			roots = append(roots, taskID)
			break
		}
	}

	// Строим графы для каждого корня
	for _, root := range roots {
		if visited[root] {
			continue
		}

		graph := &model.TaskGraph{
			RootTaskID:   root,
			Tasks:        make(map[string]*model.Task),
			Dependencies: make(map[string][]string),
			Dependents:   make(map[string][]string),
		}

		// Очередь для обхода в ширину
		queue := []string{root}
		visited[root] = true

		for len(queue) > 0 {
			currentID := queue[0]
			queue = queue[1:]

			// Добавляем текущую задачу в граф
			graph.Tasks[currentID] = tasks[currentID]
			graph.Dependencies[currentID] = deps[currentID]
			graph.Dependents[currentID] = revDeps[currentID]

			// Добавляем в очередь все задачи, которые зависят от текущей
			for _, dependentID := range revDeps[currentID] {
				if !visited[dependentID] {
					visited[dependentID] = true
					queue = append(queue, dependentID)
				}
			}

			// Добавляем в очередь все зависимости текущей задачи
			for _, dependencyID := range deps[currentID] {
				if !visited[dependencyID] {
					visited[dependencyID] = true
					queue = append(queue, dependencyID)
				}
			}
		}

		graphs = append(graphs, graph)
	}

	// Проверяем, что все задачи вошли в графы
	if len(visited) != len(tasks) {
		return nil, errors.New("some tasks are not connected to any graph")
	}

	// Теперь убедимся, что все зависимости между задачами сохранены
	for _, graph := range graphs {
		for taskID := range graph.Tasks {
			// Обновляем зависимости для каждой задачи в графе
			graph.Dependencies[taskID] = deps[taskID]
			graph.Dependents[taskID] = revDeps[taskID]
		}
	}

	return graphs, nil
}

// GetPlan retrieves a plan by ID.
func (pr *PlanRegistry) Get(id string) (*model.Plan, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plan, ok := pr.plans[id]
	if !ok {
		return nil, errors.New("plan not found")
	}
	return plan, nil
}

// UpdatePlan updates tasks in a plan (e.g. reordering, changing metadata).
func (pr *PlanRegistry) Update(id string, updated model.Plan) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plan, exists := pr.plans[id]
	if !exists {
		return errors.New("plan not found")
	}

	if updated.StatusHistory.LastStatus != "" {
		plan.StatusHistory = pr.StatusManager.NextStatus(updated.StatusHistory.LastStatus, *plan.StatusHistory)
	}

	pr.plans[id] = plan
	return nil
}

// DeletePlan removes a plan from storage.
func (pr *PlanRegistry) Delete(id string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.plans[id]; !exists {
		return errors.New("plan not found")
	}
	delete(pr.plans, id)
	return nil
}

// ListPlans returns all plans.
func (pr *PlanRegistry) List() ([]*model.Plan, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plans := make([]*model.Plan, 0, len(pr.plans))
	for _, p := range pr.plans {
		plans = append(plans, p)
	}
	return plans, nil
}

func (pr *PlanRegistry) RunAsync(planID string, executionID string) (string, error) {
	if executionID == "" {
		executionID = uuid.New().String()
	}
	pr.logger.Debugf("[] PlanRegistry.RunAsync() - planID: %s", executionID)
	pr.mu.Lock()

	_, exists := pr.plans[planID]
	if !exists {
		return "", errors.New("plan not found")
	}
	pr.mu.Unlock()
	// Start execution in a goroutine
	go pr.Run(planID, executionID)

	pr.logger.Debugf("[%s] PlanRegistry.RunAsync() - started Run()", executionID)
	return executionID, nil
}

func (pr *PlanRegistry) Run(planID string, executionID string) (string, error) {
	if executionID == "" {
		executionID = uuid.New().String()
	}
	pr.logger.Infof("[%s] PlanRegistry.Run() - call()", executionID)

	plan, err := pr.Get(planID)
	if err != nil {
		pr.logger.Errorf("[%s] Plan not found during execution", executionID)
		return "", err
	}
	// Check if plan is already running
	currentStatus := plan.StatusHistory.LastStatus
	if currentStatus == model.StatusRunning {
		return "", errors.New("plan is already running")
	}
	if currentStatus == model.StatusSuccess {
		return "", errors.New("cannot run already completed plan")
	}
	pr.mu.Lock()
	// Update plan status
	plan.StatusHistory = pr.StatusManager.NextStatus(model.StatusRunning, *plan.StatusHistory)
	pr.plans[planID] = plan
	pr.mu.Unlock()

	// Канал для обработки ошибок выполнения
	errChan := make(chan error, len(plan.TaskGraphs))
	var wg sync.WaitGroup

	// Запускаем выполнение каждого графа задач
	for _, graph := range plan.TaskGraphs {
		wg.Add(1)
		go func(g *model.TaskGraph) {
			defer wg.Done()
			if err := pr.executeTaskGraph(planID, executionID, g); err != nil {
				errChan <- fmt.Errorf("graph %s failed: %w", g.RootTaskID, err)
			}
		}(graph)
	}

	// Ждем завершения всех графов
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Обрабатываем результаты выполнения
	var executionErr error
	for err := range errChan {
		if executionErr == nil {
			executionErr = err
		} else {
			executionErr = fmt.Errorf("%v; %w", executionErr, err)
		}
	}

	// Обновляем статус плана
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plan = pr.plans[planID] // Перечитываем план, так как он мог измениться
	if executionErr != nil {
		plan.StatusHistory = pr.StatusManager.NextStatus(model.StatusFailed, *plan.StatusHistory)
		pr.logger.Errorf("[%s] Plan execution failed: %v", executionID, executionErr)
	} else {
		plan.StatusHistory = pr.StatusManager.NextStatus(model.StatusSuccess, *plan.StatusHistory)
		pr.logger.Infof("[%s] Plan executed successfully", executionID)
	}
	pr.plans[planID] = plan

	return "", nil
}

func (pr *PlanRegistry) executeTaskGraph(planID, executionID string, graph *model.TaskGraph) error {
	pr.logger.Infof("[%s] PlanRegistry.executeTaskGraph() Executing task graph with root %s", executionID, graph.RootTaskID)

	// Получаем топологический порядок выполнения задач
	executionOrder, err := pr.getExecutionOrder(graph.Dependencies)
	if err != nil {
		return fmt.Errorf("failed to get execution order: %w", err)
	}
	pr.logger.Infof("[%s] PlanRegistry.executeTaskGraph() executionOrder %s", executionID, executionOrder)

	// Выполняем задачи в порядке зависимостей
	for _, taskID := range executionOrder {
		pr.logger.Infof("[%s] PlanRegistry.executeTaskGraph() Exec task %s", executionID, taskID)
		task := graph.Tasks[taskID]

		// Создаем точку отката перед выполнением задачи
		checkpoint := &model.RollbackCheckpoint{
			GraphID: graph.RootTaskID,
			TaskID:  task.ID,
			//State:     pr.getCurrentState(task.Components),
			Timestamp: time.Now(),
		}

		// Выполняем задачу
		if _, err := pr.Tasks.Fork(task.ID, executionID); err != nil {
			pr.logger.Errorf("[%s] Task %s failed: %v", executionID, taskID, err)

			// Пытаемся откатить выполненные задачи
			if rollbackErr := pr.rollbackGraph(planID, executionID, graph, taskID); rollbackErr != nil {
				return fmt.Errorf("execution failed: %v, rollback failed: %w", err, rollbackErr)
			}
			return fmt.Errorf("task %s failed: %w", taskID, err)
		}

		// Сохраняем точку отката
		pr.saveCheckpoint(planID, checkpoint)
	}

	return nil
}

// getExecutionOrder возвращает задачи в топологическом порядке (алгоритм Кана)
func (pr *PlanRegistry) getExecutionOrder(dependencies map[string][]string) ([]string, error) {
	inDegree := make(map[string]int)
	var queue []string
	var order []string

	// Инициализация входящих степеней
	for node := range dependencies {
		inDegree[node] = 0
	}

	// Вычисление входящих степеней (сколько задач зависят от данной)
	for _, deps := range dependencies {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// Добавляем в очередь задачи без входящих зависимостей
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	// Обработка очереди
	for len(queue) > 0 {
		// Чтобы порядок был детерминированным, сортируем задачи с одинаковым inDegree
		sort.Strings(queue)
		u := queue[0]
		queue = queue[1:]
		order = append(order, u)

		// Уменьшаем степень для зависимых задач
		for _, v := range dependencies[u] {
			inDegree[v]--
			if inDegree[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	// Проверка на циклы
	if len(order) != len(dependencies) {
		return nil, errors.New("cycle detected in dependency graph")
	}

	// Разворачиваем порядок, чтобы зависимости выполнялись первыми
	reverse(order)

	return order, nil
}

func reverse(a []string) {
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}

func (pr *PlanRegistry) rollbackGraph(planID, executionID string, graph *model.TaskGraph, failedTaskID string) error {
	pr.logger.Infof("[%s] Starting rollback for graph %s after task %s failure",
		executionID, graph.RootTaskID, failedTaskID)

	// Получаем порядок отката (обратный порядку выполнения)
	executionOrder, err := pr.getExecutionOrder(graph.Dependencies)
	if err != nil {
		return fmt.Errorf("failed to get rollback order: %w", err)
	}

	// Ищем с какой задачи начать откат
	startRollback := false
	for i := len(executionOrder) - 1; i >= 0; i-- {
		taskID := executionOrder[i]

		if taskID == failedTaskID {
			startRollback = true
			continue
		}

		if !startRollback {
			continue
		}

		// Восстанавливаем состояние из точки отката
		if err := pr.restoreCheckpoint(planID, graph.RootTaskID, taskID); err != nil {
			return fmt.Errorf("failed to rollback task %s: %w", taskID, err)
		}

	}

	return nil
}

// Вспомогательные методы:

func (pr *PlanRegistry) saveCheckpoint(planID string, checkpoint *model.RollbackCheckpoint) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plan := pr.plans[planID]
	plan.RollbackStack = append(plan.RollbackStack, checkpoint)
}

func (pr *PlanRegistry) restoreCheckpoint(planID, graphID, taskID string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plan := pr.plans[planID]

	// Ищем последнюю точку отката для задачи
	for i := len(plan.RollbackStack) - 1; i >= 0; i-- {
		cp := plan.RollbackStack[i]
		if cp.GraphID == graphID && cp.TaskID == taskID {
			// Восстанавливаем состояние компонентов
			for compID, _ := range cp.State {
				if _, err := pr.Components.Get(compID); err == nil {
					//if err := comp.Restore(state); err != nil {
					return fmt.Errorf("failed to restore component %s: %w", compID, err)
					//}
				}
			}
			return nil
		}
	}

	return fmt.Errorf("checkpoint not found for task %s", taskID)
}

// Status returns the current status of a plan
func (pr *PlanRegistry) Status(planID string) (model.Status, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	plan, exists := pr.plans[planID]
	if !exists {
		return "", errors.New("plan not found")
	}

	return plan.StatusHistory.LastStatus, nil
}

// Stop terminates execution of a running plan
func (pr *PlanRegistry) Stop(planID string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plan, exists := pr.plans[planID]
	if !exists {
		return errors.New("plan not found")
	}

	currentStatus := plan.StatusHistory.LastStatus
	if currentStatus != model.StatusRunning && currentStatus != model.StatusPaused {
		return fmt.Errorf("cannot stop plan in status '%s'", currentStatus)
	}

	plan.StatusHistory = pr.StatusManager.NextStatus(model.StatusStopped, *plan.StatusHistory)
	pr.plans[planID] = plan

	// In a real implementation, you would also:
	// 1. Cancel any running tasks
	// 2. Clean up resources
	// 3. Notify dependent systems

	pr.logger.Infof("Plan '%s' stopped", planID)
	return nil
}

// Pause temporarily halts execution of a running plan
func (pr *PlanRegistry) Pause(planID string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	plan, exists := pr.plans[planID]
	if !exists {
		return errors.New("plan not found")
	}

	if plan.StatusHistory.LastStatus != model.StatusRunning {
		return fmt.Errorf("cannot pause plan in status '%s'", plan.StatusHistory.LastStatus)
	}

	plan.StatusHistory = pr.StatusManager.NextStatus(model.StatusPaused, *plan.StatusHistory)
	pr.plans[planID] = plan

	// In a real implementation, you would:
	// 1. Pause any running tasks
	// 2. Save state for resumption

	pr.logger.Infof("Plan '%s' paused", planID)
	return nil
}

// detectCycles checks for circular dependencies using Kahn's algorithm
func (pr *PlanRegistry) detectCycles(graph map[string][]string) error {
	// Implementation of cycle detection
	inDegree := make(map[string]int)
	var queue []string

	// Initialize in-degree count
	for node := range graph {
		inDegree[node] = 0
	}

	// Calculate in-degree for each node
	for _, deps := range graph {
		for _, dep := range deps {
			inDegree[dep]++
		}
	}

	// Enqueue nodes with zero in-degree
	for node, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	count := 0
	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]
		count++

		for _, v := range graph[u] {
			inDegree[v]--
			if inDegree[v] == 0 {
				queue = append(queue, v)
			}
		}
	}

	if count != len(graph) {
		return errors.New("cycle detected in dependency graph")
	}
	return nil
}
