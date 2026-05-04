package main

import (
	"bottelegram/server/config"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type CreateEventRequest struct {
	Event      EventInput      `json:"event" binding:"required"`
	Recurrence RecurrenceInput `json:"recurrence" binding:"required"`
	Reminders  []ReminderInput `json:"reminders" binding:"required,min=1,dive"`
}

type TelegramAuthContext struct {
	ChatID int64
	UserID int64
}

type EventInput struct {
	ChatID          *int64  `json:"chat_id"`
	CreatedByUserID *int64  `json:"created_by_user_id"`
	TargetUserID    *int64  `json:"target_user_id"`
	Type            string  `json:"type" binding:"required,oneof=birthday reminder custom"`
	Title           string  `json:"title" binding:"required"`
	Description     *string `json:"description"`
	IsAllDay        bool    `json:"is_all_day"`
	EventDate       *string `json:"event_date"`
	EventAt         *string `json:"event_at"`
	Timezone        string  `json:"timezone" binding:"required"`
	IsActive        bool    `json:"is_active"`
}

type RecurrenceInput struct {
	Frequency       string `json:"frequency" binding:"required,oneof=none daily weekly monthly yearly"`
	IntervalValue   int    `json:"interval_value" binding:"required,gte=1"`
	UntilAt         string `json:"until_at"`
	OccurrenceCount *int   `json:"occurrence_count"`
}

type ReminderInput struct {
	OffsetMinutes   int     `json:"offset_minutes"`
	MessageTemplate *string `json:"message_template"`
	IsActive        bool    `json:"is_active"`
}

type PageViewInput struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

func main() {
	if err := config.Init(); err != nil {
		fmt.Printf("failed to load environment configuration: %v\n", err)
		return
	}

	env := config.Current
	gin.SetMode(env.GinMode)

	dbPool, err := CreatePool(env)
	if err != nil {
		fmt.Printf("failed to initialize database pool: %v\n", err)
		return
	}
	defer dbPool.Close()

	router := gin.Default()

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = env.CORSAllowedOrigins
	corsConfig.AllowMethods = []string{"GET", "POST", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Telegram-Init-Data", "X-WebApp-Context"}
	router.Use(cors.New(corsConfig))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	router.HEAD("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	router.GET("/api/v1/events", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "events endpoint placeholder",
		})
	})

	router.POST("/api/v1/events", func(c *gin.Context) {
		var input CreateEventRequest
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event payload", "details": err.Error()})
			return
		}

		authContext, err := validateTelegramInitData(c.GetHeader("X-Telegram-Init-Data"), env.TelegramBotToken)
		if err != nil {
			authContext, err = validateWebAppContextToken(c.GetHeader("X-WebApp-Context"), env.WebAppContextSecret)
		}
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Telegram WebApp context"})
			return
		}

		input.Event.ChatID = &authContext.ChatID
		input.Event.CreatedByUserID = &authContext.UserID

		if err := validateEventRequest(input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := insertEventWithRelations(c.Request.Context(), dbPool, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist event", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":        "created",
			"event_id":      result.EventID,
			"recurrence_id": result.RecurrenceID,
		})
	})

	router.POST("/api/v1/telemetry/page-view", func(c *gin.Context) {
		var input PageViewInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid page-view payload"})
			return
		}

		if input.Path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
			return
		}

		fmt.Printf("page_view path=%s source=%s at=%s\n", input.Path, input.Source, time.Now().UTC().Format(time.RFC3339))

		c.JSON(http.StatusCreated, gin.H{"status": "tracked"})
	})

	_ = router.Run(":" + env.Port)
}
