package server

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/SmrutAI/ingestion-pipeline/internal/flow"
	"github.com/SmrutAI/ingestion-pipeline/internal/store"
)

// IndexRequest is the payload for POST /v1/index.
type IndexRequest struct {
	Source      string            `json:"source"`
	WorkspaceID string            `json:"workspace_id"`
	Config      map[string]string `json:"config"`
}

// IndexResponse is the response for POST /v1/index.
type IndexResponse struct {
	JobID string `json:"job_id"`
}

// NewApp creates and configures the shared Echo instance.
// registry is used to look up and run named flows.
// jobs is used to create and update async indexing jobs.
func NewApp(registry *flow.FlowRegistry, jobs *store.JobStore) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	e.POST("/v1/index", handleIndex(registry, jobs))
	e.GET("/v1/jobs/:id", handleGetJob(jobs))

	return e
}

func handleIndex(registry *flow.FlowRegistry, jobs *store.JobStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req IndexRequest
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if req.WorkspaceID == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspace_id is required"})
		}

		if jobs == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "job store not configured"})
		}

		job, err := jobs.Create(c.Request().Context(), req.WorkspaceID, 0)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		go func() {
			flowName := req.Source
			if flowName == "" {
				flowName = "local"
			}
			ctx := context.Background()
			if err := jobs.UpdateStatus(ctx, job.ID, "running", ""); err != nil {
				return
			}
			_, runErr := registry.Run(ctx, flowName)
			status := "done"
			errMsg := ""
			if runErr != nil {
				status = "failed"
				errMsg = runErr.Error()
			}
			_ = jobs.UpdateStatus(ctx, job.ID, status, errMsg)
		}()

		return c.JSON(http.StatusAccepted, &IndexResponse{JobID: job.ID})
	}
}

func handleGetJob(jobs *store.JobStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		job, err := jobs.Get(c.Request().Context(), id)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "job not found"})
		}
		return c.JSON(http.StatusOK, job)
	}
}
