package models

import "github.com/IPampurin/ImageProcessor/pkg/domain"

// Task – задача, полученная из Kafka (копия domain.ImageTask)
type Task = domain.ImageTask

// Result – результат обработки (копия domain.ImageResult)
type Result = domain.ImageResult

// VariantResult – копия domain.VariantResult
type VariantResult = domain.VariantResult
