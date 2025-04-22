package bulkloagen

import (
	"context"
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/netbox-community/go-netbox/v4"
	"go.uber.org/zap"

	"github.com/exaring/bulk-loagen/internal"
	"github.com/exaring/bulk-loagen/pkg/config"
)

//go:embed templates
var templateFiles embed.FS

//go:embed static
var staticFiles embed.FS

type Service struct {
	chi.Router
	cfg    *config.Config
	logger *zap.Logger

	nb *netbox.APIClient
}

func NewService(cfg *config.Config) (*Service, error) {
	s := &Service{
		cfg:    cfg,
		Router: chi.NewRouter(),
		logger: zap.L().With(zap.String("component", internal.Name)),
	}

	s.logger.Sugar().Infof("initializing %s", internal.Name)

	s.nb = netbox.NewAPIClientFor(
		cfg.NetBoxScheme+"://"+cfg.NetBoxHost,
		cfg.NetBoxToken,
	)

	s.Get("/", s.index)
	s.Get("/api/v1/devices/{deviceID:[0-9]+}", s.devices)
	s.Get("/api/v1/loa/rear-ports", s.rearPorts)
	s.Handle("/static/*", http.FileServer(http.FS(staticFiles)))

	return s, nil
}

func (s *Service) index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFS(templateFiles, "templates/index.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot parse template: %v", err), http.StatusInternalServerError)
		return
	}

	data := &TemplateData{}
	data.Version = internal.Version
	data.ExampleDeviceURL = "/api/v1/devices/{deviceID}"

	if err := t.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Cannot execute template: %v", err), http.StatusInternalServerError)
		return
	}
}

func (s *Service) devices(w http.ResponseWriter, r *http.Request) {
	deviceID := chi.URLParam(r, "deviceID")

	rearPortID, err := strconv.Atoi(r.URL.Query().Get("rear_port"))
	if err != nil {
		rearPortID = 0
	}

	devID, err := strconv.ParseInt(deviceID, 10, 32)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot parse device id as integer: %v", err), http.StatusInternalServerError)
		return
	}

	devRes, _, err := s.nb.DcimAPI.DcimDevicesRetrieve(context.Background(), int32(devID)).Execute()
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get device read: %v", err), http.StatusInternalServerError)
		return
	}

	res, _, err := s.nb.DcimAPI.DcimRearPortsList(context.Background()).DeviceId([]int32{devRes.Id}).Execute()
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get rear port list: %v", err), http.StatusInternalServerError)
		return
	}

	t, err := template.ParseFS(templateFiles, "templates/devices.html")
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot parse template: %v", err), http.StatusInternalServerError)
		return
	}

	data := &TemplateData{}
	data.Version = internal.Version

	rearp := map[int32]string{}

	for _, v := range res.GetResults() {
		data.Device = v.Device.GetName()
		rearp[v.GetId()] = v.GetName()
	}

	data.RearPorts = rearp
	data.RearPortID = rearPortID

	tenant := "default"

	if setTenant := retrieveNullable(devRes.Tenant); setTenant.Slug != "" {
		tenant = setTenant.Slug
	}

	if err := s.mergeTenantInfo(tenant, data); err != nil {
		http.Error(w, fmt.Sprintf("Cannot merge tenant information: %v", err), http.StatusInternalServerError)
		return
	}

	if err := t.ExecuteTemplate(w, "devices.html", data); err != nil {
		http.Error(w, fmt.Sprintf("Cannot execute template: %v", err), http.StatusInternalServerError)
		return
	}
}

func (s *Service) rearPorts(w http.ResponseWriter, r *http.Request) {
	rearPortID, err := strconv.Atoi(r.URL.Query().Get("port"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot parse query parameter port as integer: %v", err), http.StatusInternalServerError)
		return
	}

	rppRes, _, err := s.nb.DcimAPI.DcimRearPortsRetrieve(context.Background(), int32(rearPortID)).Execute()
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get rear port read: %v", err), http.StatusInternalServerError)
		return
	}

	devRes, _, err := s.nb.DcimAPI.DcimDevicesRetrieve(context.Background(), rppRes.GetDevice().Id).Execute()
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get device read: %v", err), http.StatusInternalServerError)
		return
	}

	siteRes, _, err := s.nb.DcimAPI.DcimSitesRetrieve(context.Background(), devRes.GetSite().Id).Execute()
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get site read: %v", err), http.StatusInternalServerError)
		return
	}

	if !devRes.Rack.IsSet() {
		http.Error(w, "Cannot get device's rack", http.StatusInternalServerError)
		return
	}

	rackRes, _, err := s.nb.DcimAPI.DcimRacksRetrieve(context.Background(), devRes.GetRack().Id).Execute()
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get rack read: %v", err), http.StatusInternalServerError)
		return
	}

	facilityID := retrieveNullable(rackRes.FacilityId)
	rackPos := retrieveNullable(devRes.Position)
	devName := retrieveNullable(devRes.Name)

	rppName := rppRes.Name
	if rppName == "" {
		http.Error(w, "Cannot get rear port name", http.StatusInternalServerError)
		return
	}

	data := &TemplateData{
		Partner:       r.URL.Query().Get("partner"),
		PartnerStreet: r.URL.Query().Get("partner_street"),
		PartnerCity:   r.URL.Query().Get("partner_city"),
		Site:          siteRes.GetFacility(),
		DemarcPanel:   fmt.Sprintf("Rack %s U%g - %s", facilityID, rackPos, devName),
		DemarcPort:    rppName,
	}

	tenant := "default"

	if devRes.Tenant.IsSet() && devRes.GetTenant().Slug != "" {
		tenant = devRes.GetTenant().Slug
	}

	if err := s.mergeTenantInfo(tenant, data); err != nil {
		http.Error(w, fmt.Sprintf("Cannot merge tenant information: %v", err), http.StatusInternalServerError)
	}

	pdf, err := Generate(*data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot generate loa: %v", err), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("LOA_%s_%s_%s.pdf", strings.ReplaceAll(r.URL.Query().Get("partner"), " ", ""), time.Now().Format("2006-01-02"), siteRes.Name)

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	_, _ = io.Copy(w, pdf)
}

func (s *Service) mergeTenantInfo(tenant string, data *TemplateData) error {
	if t, ok := s.cfg.Tenants[tenant]; ok {
		data.OurName = t.Name
		data.OurNameShort = t.Short
		data.OurStreet = t.Street
		data.OurCity = t.City
		data.OurNocName = t.NOC
		data.OurNocEmail = t.Email
		data.OurNocPhone = t.Phone
		data.ExpiryDays = t.ExpiryDays

		return nil
	}

	return fmt.Errorf("%w: %s", errors.New("unknown tenant"), tenant)
}

type nullable[K any] interface {
	IsSet() bool
	Get() *K
}

func retrieveNullable[N any, K nullable[N]](nullable K) N {
	if nullable.IsSet() && nullable.Get() != nil {
		return *nullable.Get()
	}

	return *new(N)
}
