package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/IPampurin/Orders-Info-Menedger/service/pkg/cache"
	"github.com/IPampurin/Orders-Info-Menedger/service/pkg/db"
	"github.com/IPampurin/Orders-Info-Menedger/service/pkg/server"

	// для трейсинга
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace/noop"

	// для Prometheus метрик

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// провайдер трейсов для сервиса
// var tracer trace.Tracer

// initTracing - инициализация трейсинга для сервиса
func initTracing() (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// экспорт трейсов в jaeger через otlp/grpc
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithEndpoint("jaeger:4317"), otlptracegrpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("не удалось создать экспортер трейсов: %w", err)
	}

	// создаем провайдер трейсов
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("order-service"),
			semconv.ServiceVersion("1.0.0"),
		)),
	)

	// устанавливаем глобальный провайдер
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	log.Println("Трейсинг сервиса инициализирован (jaeger:4317).")
	return tp, nil
}

func main() {

	// запускаем сервер для метрик
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		port := ":8890"
		log.Printf("Prometheus метрики сервиса доступны на http://localhost%s/metrics\n", port)
		if err := http.ListenAndServe(port, nil); err != nil {
			log.Printf("Ошибка запуска сервера метрик: %v\n", err)
		}
	}()

	// инициализируем трейсинг
	var tp *sdktrace.TracerProvider
	var err error

	tp, err = initTracing()
	if err != nil {
		log.Printf("Ошибка инициализации трейсинга: %v, работаем без трейсинга.\n", err)
		// создаем noop-трейсер для работы без трассировки
		_ = noop.NewTracerProvider().Tracer("noop-service")
	} else {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Ошибка остановки трейсинга: %v\n", err)
			}
		}()
		// получаем реальный трейсер из провайдера
		_ = otel.Tracer("order-service")
	}

	// подключаем базу данных
	err = db.ConnectDB()
	if err != nil {
		fmt.Printf("ошибка вызова db.ConnectDB: %v\n", err)
		return
	}
	defer db.CloseDB()

	// инициализируем кэш
	err = cache.InitRedis()
	if err != nil {
		fmt.Printf("кэш отвалился, ошибка вызова cache.InitRedis: %v\n", err)
	}

	// создаем контекст для сигналов отмены
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// запускаем сервер и ждем его завершения
	if err := server.Run(ctx); err != nil {
		log.Printf("Ошибка сервера: %v\n", err)
		return
	}

	log.Println("Приложение корректно завершено")
}
