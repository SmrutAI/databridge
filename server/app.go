package server

import (
	"context"
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"

	"github.com/SmrutAI/databridge/internal/flow"
	"github.com/SmrutAI/databridge/internal/store"
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

	e.GET("/v1/health", handleHealth())
	e.POST("/v1/index", handleIndex(registry, jobs))
	e.GET("/v1/jobs/:id", handleGetJob(jobs))
	e.GET("/v1/flows", handleListFlows(registry))
	e.POST("/v1/flows/:name/run", handleRunFlow(registry))

	return e
}

func handleHealth() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "databridge",
		})
	}
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
			ctx := context.Background()
			if err := jobs.UpdateStatus(ctx, job.ID, "running", ""); err != nil {
				return
			}
			f, buildErr := BuildFlow(req.WorkspaceID, req.Source, req.Config)
			if buildErr != nil {
				_ = jobs.UpdateStatus(ctx, job.ID, "failed", buildErr.Error())
				return
			}
			_, runErr := f.Run(ctx)
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
		if jobs == nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "job store not configured"})
		}
		id := c.Param("id")
		job, err := jobs.Get(c.Request().Context(), id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return c.JSON(http.StatusNotFound, map[string]string{"error": "job not found"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, job)
	}
}

func handleListFlows(registry *flow.FlowRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		names := registry.List()
		return c.JSON(http.StatusOK, map[string][]string{"flows": names})
	}
}

func handleRunFlow(registry *flow.FlowRegistry) echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")

		// Check existence before running so we can return a clean 404.
		known := registry.List()
		found := false
		for i := range known {
			if known[i] == name {
				found = true
				break
			}
		}
		if !found {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "flow not found: " + name})
		}

		stats, err := registry.Run(c.Request().Context(), name)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, stats)
	}
}
