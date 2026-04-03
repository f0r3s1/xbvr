package api

import (
	"net/http"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/tasks"
)

type HealthResource struct{}

func (i HealthResource) WebService() *restful.WebService {
	tags := []string{"Health"}

	ws := new(restful.WebService)
	ws.Path("/api/health").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)

	ws.Route(ws.POST("/scan").To(i.startScan).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.GET("/report").To(i.getReport).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/cancel").To(i.cancel).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	ws.Route(ws.POST("/fix").To(i.fix).
		Metadata(restfulspec.KeyOpenAPITags, tags))

	return ws
}

func (i HealthResource) startScan(req *restful.Request, resp *restful.Response) {
	if tasks.IsHealthRunning() {
		resp.WriteHeaderAndEntity(http.StatusConflict, map[string]string{"error": "health check already running"})
		return
	}
	go tasks.RunHealthCheck()
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "started"})
}

func (i HealthResource) getReport(req *restful.Request, resp *restful.Response) {
	report := tasks.GetLastHealthReport()
	if report == nil {
		resp.WriteHeaderAndEntity(http.StatusOK, map[string]interface{}{
			"report":  nil,
			"running": tasks.IsHealthRunning(),
		})
		return
	}
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]interface{}{
		"report":  report,
		"running": tasks.IsHealthRunning(),
	})
}

func (i HealthResource) cancel(req *restful.Request, resp *restful.Response) {
	tasks.CancelHealthCheck()
	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "cancelling"})
}

type FixRequest struct {
	Action string `json:"action"`
}

func (i HealthResource) fix(req *restful.Request, resp *restful.Response) {
	var r FixRequest
	if err := req.ReadEntity(&r); err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := tasks.FixHealthIssue(r.Action); err != nil {
		resp.WriteHeaderAndEntity(http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resp.WriteHeaderAndEntity(http.StatusOK, map[string]string{"status": "started"})
}
