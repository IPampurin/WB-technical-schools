package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/cache"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/db"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/models"
	"github.com/IPampurin/WB-technical-schools/L0/service/pkg/shutdown"
	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

// GetOrderByID выдаёт данные о заказе по order_uid
func GetOrderByID(w http.ResponseWriter, r *http.Request) {

	// проверяем не останавливается ли сервер
	if shutdown.IsShuttingDown() {
		http.Error(w, "Сервер находится в процессе остановки. Операция невозможна.", http.StatusServiceUnavailable)
		return
	}

	// получаем OrderUID из параметров запроса
	orderUID := chi.URLParam(r, "order_uid")

	if orderUID == "" {
		log.Printf("Ошибка: order_uid не указан")
		http.Error(w, "Параметр order_uid обязателен", http.StatusBadRequest)
		return
	}

	// создаем экземпляр заказа
	var order models.Order
	// создаём экземпляр для ответа
	var resp []byte

	// проверяем есть ли такой заказ в кэше
	cacheKey := fmt.Sprintf("order:%s", orderUID)
	// если данные в кэше есть
	if jsonData, err := cache.GetCache(cacheKey); err == nil {
		// если данные не корректные
		if err = json.Unmarshal(jsonData, &order); err != nil {
			log.Printf("Битые данные в кэше: %s. Удаляем ключ.", cacheKey)
			// убираем мусор
			if err := cache.DelCache(cacheKey); err != nil {
				log.Printf("Ошибка удаления битых данных из кэша %s: %v", cacheKey, err)
			}
		} else {
			// если данные корректные
			// маршалим даные в JSON с отступами для читаемости
			resp, err = json.MarshalIndent(order, "", "    ")
			if err != nil {
				log.Printf("Ошибка при маршалинге данных: %v", err)
				http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError)
				return
			}
			log.Printf("Заказ с UID %s успешно найден в кэше", orderUID)
		}
	} else {
		// если данных в кэше нет,
		// то получаем заказ из базы данных
		result := db.DB.Db.First(&order, "order_uid = ?", orderUID)
		// если что-то не так
		if result.Error != nil {
			// если просто такого заказа нет
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				log.Printf("Заказ с UID %s не найден", orderUID)
				http.Error(w, "Заказ не найден", http.StatusNotFound)
				return
			}
			// если что-то невообразимое
			log.Printf("Ошибка при получении заказа: %v", result.Error)
			http.Error(w, "Ошибка при получении заказа", http.StatusInternalServerError)
			return
		}
		// если заказ в базе есть и штатно получен, то записываем заказ в кэш
		if err := cache.SetCahe(cacheKey, order, cache.GetTTL()); err != nil {
			log.Printf("Ошибка кэширования при запросе по uid: %v", err)
		}

		log.Printf("Заказ с UID %s найден в БД и занесён в кэш", orderUID)

		// и возвращаем читабельный ответ
		resp, err = json.MarshalIndent(order, "", "    ")
		if err != nil {
			log.Printf("Ошибка при маршалинге данных: %v", err)
			http.Error(w, "Ошибка при формировании ответа", http.StatusInternalServerError)
			return
		}
	}

	// формируем ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)

	log.Printf("Заказ с UID %s успешно получен", orderUID)
}
