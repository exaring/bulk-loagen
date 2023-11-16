package bulkloagen

import (
	"embed"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/netbox-community/go-netbox/netbox/client"
	"github.com/netbox-community/go-netbox/netbox/client/dcim"
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

	nb *client.NetBoxAPI
}

func NewService(cfg *config.Config) (*Service, error) {
	s := &Service{
		cfg:    cfg,
		Router: chi.NewRouter(),
		logger: zap.L().With(zap.String("component", internal.Name)),
	}

	s.logger.Sugar().Infof("initializing %s", internal.Name)

	transport := httptransport.New(cfg.NetBoxHost, client.DefaultBasePath, []string{cfg.NetBoxScheme})
	transport.DefaultAuthentication = httptransport.APIKeyAuth("Authorization", "header", "Token "+cfg.NetBoxToken)

	s.nb = client.New(transport, nil)

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

	dev := dcim.NewDcimDevicesReadParams()

	dev.ID, err = strconv.ParseInt(deviceID, 10, 64)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot parse device id as integer: %v", err), http.StatusInternalServerError)
		return
	}

	devRes, err := s.nb.Dcim.DcimDevicesRead(dev, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get device read: %v", err), http.StatusInternalServerError)
		return
	}

	rpp := dcim.NewDcimRearPortsListParams()
	rpp.DeviceID = &deviceID

	res, err := s.nb.Dcim.DcimRearPortsList(rpp, nil)
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

	rearp := map[int64]string{}

	for _, v := range res.GetPayload().Results {
		data.Device = *v.Device.Name
		rearp[v.ID] = *v.Name
	}

	data.RearPorts = rearp
	data.RearPortID = rearPortID

	tenant := "default"

	if devRes.GetPayload().Tenant != nil {
		tenant = *devRes.GetPayload().Tenant.Slug
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

	rpp := dcim.NewDcimRearPortsReadParams()
	rpp.ID = int64(rearPortID)

	rppRes, err := s.nb.Dcim.DcimRearPortsRead(rpp, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get rear port read: %v", err), http.StatusInternalServerError)
		return
	}

	dev := dcim.NewDcimDevicesReadParams()
	dev.ID = rppRes.GetPayload().Device.ID

	devRes, err := s.nb.Dcim.DcimDevicesRead(dev, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get device read: %v", err), http.StatusInternalServerError)
		return
	}

	site := dcim.NewDcimSitesReadParams()
	site.ID = devRes.GetPayload().Site.ID

	siteRes, err := s.nb.Dcim.DcimSitesRead(site, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get site read: %v", err), http.StatusInternalServerError)
		return
	}

	if devRes.GetPayload().Rack == nil {
		http.Error(w, "Cannot get device's rack", http.StatusInternalServerError)
		return
	}

	rack := dcim.NewDcimRacksReadParams()
	rack.ID = devRes.GetPayload().Rack.ID

	rackRes, err := s.nb.Dcim.DcimRacksRead(rack, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot get rack read: %v", err), http.StatusInternalServerError)
		return
	}

	facilityID := rackRes.GetPayload().FacilityID
	if facilityID == nil {
		http.Error(w, "Cannot get rack facility id", http.StatusInternalServerError)
		return
	}

	rackPos := devRes.GetPayload().Position
	if rackPos == nil {
		http.Error(w, "Cannot get device's rack position", http.StatusInternalServerError)
		return
	}

	devName := devRes.GetPayload().Name
	if devName == nil {
		http.Error(w, "Cannot get device name", http.StatusInternalServerError)
		return
	}

	rppName := rppRes.GetPayload().Name
	if rppName == nil {
		http.Error(w, "Cannot get rear port name", http.StatusInternalServerError)
		return
	}

	data := &TemplateData{
		Partner:       r.URL.Query().Get("partner"),
		PartnerStreet: r.URL.Query().Get("partner_street"),
		PartnerCity:   r.URL.Query().Get("partner_city"),
		Site:          siteRes.GetPayload().Facility,
		DemarcPanel:   fmt.Sprintf("Rack %s U%d - %s", *facilityID, int(*rackPos), *devName),
		DemarcPort:    *rppName,
	}

	tenant := "default"

	if devRes.GetPayload().Tenant != nil {
		tenant = *devRes.GetPayload().Tenant.Slug
	}

	if err := s.mergeTenantInfo(tenant, data); err != nil {
		http.Error(w, fmt.Sprintf("Cannot merge tenant information: %v", err), http.StatusInternalServerError)
	}

	pdf, err := Generate(*data)
	if err != nil {
		http.Error(w, fmt.Sprintf("Cannot generate loa: %v", err), http.StatusInternalServerError)
		return
	}

	filename := fmt.Sprintf("LOA_%s_%s_%s.pdf", r.URL.Query().Get("partner"), time.Now().Format("2006-01-02"), *siteRes.GetPayload().Name)

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

	return errors.New("unknown tenant")
}
