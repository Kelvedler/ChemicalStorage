package view

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/microcosm-cc/bluemonday"

	"github.com/Kelvedler/ChemicalStorage/pkg/middleware"
)

func staticFilepath(mainLogger *slog.Logger) string {
	wd, err := os.Getwd()
	if err != nil {
		mainLogger.Error(err.Error())
		os.Exit(1)
	}
	return wd + "/static/"
}

func BaseRouter(
	dbpool *pgxpool.Pool,
	sanitize *bluemonday.Policy,
	validate *validator.Validate,
	mainLogger *slog.Logger,
) *httprouter.Router {
	router := httprouter.New()
	handlerContext := middleware.NewHandlerContext(dbpool, sanitize, validate)

	router.GET("/favicon.ico", middleware.UnrestrictedNoAuth.Wrapper(Favicon, handlerContext))
	router.ServeFiles("/static/*filepath", http.Dir(staticFilepath(mainLogger)))
	router.GET("/", middleware.Unrestricted.Wrapper(Index, handlerContext))
	router.GET("/sign-in", middleware.UnrestrictedNoAuth.Wrapper(SignIn, handlerContext))
	router.GET("/sign-up", middleware.UnrestrictedNoAuth.Wrapper(SignUp, handlerContext))
	router.GET("/me", middleware.Unrestricted.Wrapper(Me, handlerContext))
	router.GET("/users/", middleware.AdminOnlyView.Wrapper(Users, handlerContext))
	router.GET("/users/:userID", middleware.AdminOnlyView.Wrapper(User, handlerContext))
	router.GET("/reagent-new", middleware.AssistantOnlyView.Wrapper(ReagentCreate, handlerContext))
	router.GET("/reagents/", middleware.Unrestricted.Wrapper(Reagents, handlerContext))
	router.GET("/reagents/:reagentID", middleware.Unrestricted.Wrapper(Reagent, handlerContext))
	router.GET(
		"/reagents/:reagentID/instance-new",
		middleware.AssistantOnlyView.Wrapper(ReagentInstanceCreate, handlerContext),
	)
	router.GET(
		"/reagents/:reagentID/instances/:instanceID",
		middleware.LecturerAssistantView.Wrapper(ReagentInstance, handlerContext),
	)
	router.GET(
		"/storage-new",
		middleware.AssistantOnlyView.Wrapper(StorageCreate, handlerContext),
	)
	router.GET("/storages/", middleware.AssistantOnlyView.Wrapper(Storages, handlerContext))
	router.GET(
		"/storages/:storageID",
		middleware.AssistantOnlyView.Wrapper(Storage, handlerContext),
	)

	router.POST(
		"/api/v1/sign-in",
		middleware.Unrestricted.Wrapper(SignInAPI, handlerContext),
	)
	router.POST(
		"/api/v1/sign-out",
		middleware.Unrestricted.Wrapper(SignOutAPI, handlerContext),
	)
	router.POST(
		"/api/v1/sign-up",
		middleware.Unrestricted.Wrapper(SignUpAPI, handlerContext),
	)
	router.PUT(
		"/api/v1/users/:userID",
		middleware.AdminOnlyAPI.Wrapper(UserPutAPI, handlerContext),
	)
	router.GET("/api/v1/users/", middleware.AdminOnlyAPI.Wrapper(UsersAPI, handlerContext))
	router.GET(
		"/api/v1/reagents/",
		middleware.Unrestricted.Wrapper(ReagentsAPI, handlerContext),
	)
	router.POST(
		"/api/v1/reagents",
		middleware.AssistantOnlyAPI.Wrapper(ReagentCreateAPI, handlerContext),
	)
	router.PUT(
		"/api/v1/reagents/:reagentID",
		middleware.AssistantOnlyAPI.Wrapper(ReagentPutAPI, handlerContext),
	)
	router.POST(
		"/api/v1/reagents/:reagentID/instances",
		middleware.AssistantOnlyAPI.Wrapper(ReagentInstanceCreateAPI, handlerContext),
	)
	router.POST(
		"/api/v1/reagents/:reagentID/instances/:instanceID/use",
		middleware.AssistantOnlyAPI.Wrapper(ReagentInstanceUseAPI, handlerContext),
	)
	router.POST(
		"/api/v1/reagents/:reagentID/instances/:instanceID/transfer",
		middleware.AssistantOnlyAPI.Wrapper(ReagentInstanceTransferAPI, handlerContext),
	)
	router.GET(
		"/api/v1/storages",
		middleware.AssistantOnlyAPI.Wrapper(StoragesAPI, handlerContext),
	)
	router.POST(
		"/api/v1/storages",
		middleware.AssistantOnlyAPI.Wrapper(StorageCreateAPI, handlerContext),
	)
	return router
}
