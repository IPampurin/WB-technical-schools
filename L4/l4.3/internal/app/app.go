// сборка и запуск зависимостей
package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/IPampurin/EventCalendar/internal/async/archiver"
	"github.com/IPampurin/EventCalendar/internal/async/logger"
	"github.com/IPampurin/EventCalendar/internal/async/scheduler"
	"github.com/IPampurin/EventCalendar/internal/configuration"
	"github.com/IPampurin/EventCalendar/internal/service"
	sqlDB "github.com/IPampurin/EventCalendar/internal/storage/sqlDB"
	calendarhttp "github.com/IPampurin/EventCalendar/internal/transport/http"
)

// App объединяет все зависимости и управляет их жизненным циклом
type App struct {
	cfg    *configuration.Config
	logger *logger.AsyncLogger
	ctx    context.Context
	cancel context.CancelFunc
}

// New создаёт новый экземпляр приложения, инициализирует логгер и контекст
func New(cfg *configuration.Config) (*App, error) {

	appLogger := logger.NewAsyncLogger(cfg.App.LogBufferSize)

	ctx, cancel := context.WithCancel(context.Background())

	return &App{
		cfg:    cfg,
		logger: appLogger,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Run запускает все компоненты приложения и блокируется до завершения
func (a *App) Run() error {

	// запускаем горутину обработки сигналов
	go a.handleSignals()

	// подключение к БД
	db, err := sqlDB.StartDB(a.ctx, &a.cfg.DB)
	if err != nil {
		a.logger.Error("ошибка подключения к БД", "error", err)
		return err
	}
	defer db.Close()

	// миграции
	if err := sqlDB.Migrations(a.ctx, db); err != nil {
		a.logger.Error("ошибка миграций", "error", err)
		return err
	}

	// репозиторий
	repo := sqlDB.NewStore(db)

	// получаем запускаем фоном планировщик напоминаний
	reminderScheduler := scheduler.NewScheduler(a.ctx, repo, a.logger, a.cfg.App.ReminderQueueSize)
	go reminderScheduler.Run()
	defer reminderScheduler.Stop()

	// сервис календаря
	calendarSvc, err := service.NewCalendarService(repo, reminderScheduler, a.logger, a.cfg.App.Timezone)
	if err != nil {
		a.logger.Error("ошибка создания сервиса", "error", err)
		return err
	}

	// получаем и запускаем фоном архиватор
	archiverWorker := archiver.NewArchiver(a.ctx, repo, a.logger, a.cfg.App.ArchiveEvery)
	go archiverWorker.Run()

	// HTTP-сервер
	srv := calendarhttp.NewServer(&a.cfg.HTTP, calendarSvc, a.logger)
	a.logger.Info("запуск HTTP", "addr", srv.Addr())

	if err := srv.Run(a.ctx); err != nil {
		a.logger.Error("ошибка HTTP-сервера", "error", err)
		return err
	}

	a.logger.Info("HTTP-сервер корректно остановлен")

	return nil
}

// handleSignals слушает SIGINT/SIGTERM и отменяет контекст
func (a *App) handleSignals() {

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	select {
	case <-a.ctx.Done():
		return
	case <-sigChan:
		a.cancel()
		return
	}
}

// Close закрывает ресурсы приложения
func (a *App) Close() error {

	if a.logger != nil {
		return a.logger.Close()
	}

	return nil
}
