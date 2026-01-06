package ratelimiter

import "time"

type Limiter interface {
	Allow(ip string) (bool, time.Duration)
}

// отвечаем может ли пользователь с этим ip делать запросы,
// 429 и если не может отправляем сколько ему еще ждать до отправки новых запросов
