package routers

import (
	"freescholar-backend/api/handlers"
	"freescholar-backend/api/middleware"
	"freescholar-backend/config"
	"freescholar-backend/pkg/elasticsearch"
	"freescholar-backend/pkg/redis"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter configures the Gin router
func SetupRouter(cfg *config.Config, db *gorm.DB, redisClient *redis.Client, esClient *elasticsearch.Client) *gin.Engine {
	// Set Gin mode
	if cfg.Server.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Apply middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH", "VIEW"}
	router.Use(cors.New(corsConfig))

	// Initialize handlers
	userHandler := handlers.NewUserHandler(db, redisClient, cfg)
	//publicationHandler := handlers.NewPublicationHandler(db, esClient, cfg)
	//authorHandler := handlers.NewAuthorHandler(db, esClient, cfg)
	//scholarPortalHandler := handlers.NewScholarPortalHandler(db, cfg)
	//relationHandler := handlers.NewRelationHandler(db, cfg)
	//searchListHandler := handlers.NewSearchListHandler(db, esClient, cfg)
	//messageCenterHandler := handlers.NewMessageCenterHandler(db, cfg)
	//filesHandler := handlers.NewFilesHandler(db, cfg)
	//serializationHandler := handlers.NewSerializationHandler(db, cfg)

	// Set up auth middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWT.Secret, redisClient)

	// API routes
	api := router.Group("/api")
	{
		// User routes
		userRoutes := api.Group("/user")
		{
			userRoutes.POST("/register", userHandler.Register)
			userRoutes.POST("/login", userHandler.Login)
			userRoutes.GET("/logout", authMiddleware.RequireAuth(), userHandler.Logout)
			userRoutes.GET("/profile", authMiddleware.RequireAuth(), userHandler.GetProfile)
			userRoutes.PUT("/profile", authMiddleware.RequireAuth(), userHandler.UpdateProfile)
			userRoutes.POST("/reset-password", userHandler.RequestPasswordReset)
			userRoutes.POST("/reset-password/:token", userHandler.ResetPassword)
		}
	
		/*
		// Publication routes
		publicationRoutes := api.Group("/publication")
		{
			publicationRoutes.GET("", publicationHandler.GetPublications)
			publicationRoutes.GET("/:id", publicationHandler.GetPublication)
			publicationRoutes.POST("", authMiddleware.RequireAuth(), publicationHandler.CreatePublication)
			publicationRoutes.PUT("/:id", authMiddleware.RequireAuth(), publicationHandler.UpdatePublication)
			publicationRoutes.DELETE("/:id", authMiddleware.RequireAuth(), publicationHandler.DeletePublication)
		}

		// Author routes
		authorRoutes := api.Group("/author")
		{
			authorRoutes.GET("", authorHandler.GetAuthors)
			authorRoutes.GET("/:id", authorHandler.GetAuthor)
			authorRoutes.POST("", authMiddleware.RequireAuth(), authorHandler.CreateAuthor)
			authorRoutes.PUT("/:id", authMiddleware.RequireAuth(), authorHandler.UpdateAuthor)
		}

		// ScholarPortal routes
		scholarRoutes := api.Group("/ScholarPortal")
		{
			scholarRoutes.GET("", scholarPortalHandler.GetScholars)
			scholarRoutes.GET("/:id", scholarPortalHandler.GetScholar)
			scholarRoutes.POST("", authMiddleware.RequireAuth(), scholarPortalHandler.CreateScholar)
			scholarRoutes.PUT("/:id", authMiddleware.RequireAuth(), scholarPortalHandler.UpdateScholar)
		}

		// Relation routes
		relationRoutes := api.Group("/relation")
		{
			relationRoutes.GET("", relationHandler.GetRelations)
			relationRoutes.POST("", authMiddleware.RequireAuth(), relationHandler.CreateRelation)
			relationRoutes.DELETE("/:id", authMiddleware.RequireAuth(), relationHandler.DeleteRelation)
		}

		// SearchList routes
		searchRoutes := api.Group("/searchList")
		{
			searchRoutes.GET("", searchListHandler.Search)
			searchRoutes.GET("/history", authMiddleware.RequireAuth(), searchListHandler.GetSearchHistory)
			searchRoutes.POST("/save", authMiddleware.RequireAuth(), searchListHandler.SaveSearch)
		}

		// MessageCenter routes
		messageRoutes := api.Group("/MessageCenter")
		{
			messageRoutes.GET("", authMiddleware.RequireAuth(), messageCenterHandler.GetMessages)
			messageRoutes.GET("/:id", authMiddleware.RequireAuth(), messageCenterHandler.GetMessage)
			messageRoutes.POST("", authMiddleware.RequireAuth(), messageCenterHandler.SendMessage)
			messageRoutes.PUT("/:id/read", authMiddleware.RequireAuth(), messageCenterHandler.MarkAsRead)
		}

		// Files routes
		filesRoutes := api.Group("/media")
		{
			filesRoutes.POST("/upload", authMiddleware.RequireAuth(), filesHandler.UploadFile)
			filesRoutes.GET("/:filename", filesHandler.GetFile)
		}

		// Serialization routes
		serialRoutes := api.Group("/serialization")
		{
			serialRoutes.GET("", serializationHandler.GetSerializations)
			serialRoutes.POST("", authMiddleware.RequireAuth(), serializationHandler.CreateSerialization)
		}
		*/
	}
	
	// Serve static files
	router.Static("/media", cfg.Media.Root)

	return router
}